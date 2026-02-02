import os
from typing import Any

from fastapi import Depends, FastAPI, HTTPException
from pydantic import BaseModel
from tortoise.contrib.fastapi import register_tortoise  # pylint: disable=import-error

from api.dependencies import get_db_service, get_scraper_service
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService

app = FastAPI()


class ScrapeRequest(BaseModel):
    url: str
    depth: int = 1


@app.post("/scrape", response_model=dict[str, int])
async def start_scrape(
    request: ScrapeRequest,
    scraper_service: ScraperService = Depends(get_scraper_service),
) -> dict[str, int]:
    try:
        scraping_id = await scraper_service.start_scraping(request.url, request.depth)
        return {"scraping_id": scraping_id}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/scrape")
async def get_scrape_status(
    scraping_id: int, service: ScraperService = Depends(get_scraper_service)
) -> dict[str, Any]:
    try:
        status = await service.get_scraping_status(scraping_id)
        if not status:
            raise HTTPException(status_code=404, detail="Scraping not found")

        response = {"status": status["status"], "scraping": status}

        if status["status"] == "COMPLETED":
            results = await service.get_scraping_results(scraping_id)
            response["data"] = results

        return response
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/search")
async def search(
    t: str, service: DbService = Depends(get_db_service)
) -> dict[str, list[str]]:
    if not t:
        raise HTTPException(status_code=400, detail="Term 't' is required")

    try:
        results = await service.search_websites(t)
        return {"websites": results}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/terms")
async def terms(
    w: str, service: DbService = Depends(get_db_service)
) -> dict[str, list[dict[str, Any]]]:
    if not w:
        raise HTTPException(status_code=400, detail="Website 'w' is required")

    try:
        results = await service.get_website_terms(w)
        return {"terms": results}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


# Tortoise ORM Init with Connection Pooling
db_url = os.getenv("DATABASE_URL", "postgres://user:pass@localhost:5432/isidorus")
# Append connection pool settings
if "?" not in db_url:
    db_url += "?minsize=10&maxsize=50"
else:
    db_url += "&minsize=10&maxsize=50"

register_tortoise(
    app,
    db_url=db_url,
    modules={"models": ["api.models"]},
    generate_schemas=False,
    add_exception_handlers=True,
)
