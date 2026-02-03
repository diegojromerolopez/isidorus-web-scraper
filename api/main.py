from typing import Any

from fastapi import Depends, FastAPI, HTTPException
from pydantic import BaseModel
from tortoise.contrib.fastapi import register_tortoise  # pylint: disable=import-error

from api.config import Configuration
from api.dependencies import get_api_key, get_db_service, get_scraper_service
from api.models import APIKey
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService

app = FastAPI()


class ScrapeRequest(BaseModel):
    url: str
    depth: int = 1


@app.get("/health")
async def health_check() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/scrape", response_model=dict[str, int])
async def start_scrape(
    request: ScrapeRequest,
    scraper_service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
) -> dict[str, int]:
    try:
        scraping_id = await scraper_service.start_scraping(request.url, request.depth)
        return {"scraping_id": scraping_id}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/scrape")
async def get_scrape_status(
    scraping_id: int,
    service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
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
    t: str,
    service: DbService = Depends(get_db_service),
    _api_key: APIKey = Depends(get_api_key),
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
    w: str,
    service: DbService = Depends(get_db_service),
    _api_key: APIKey = Depends(get_api_key),
) -> dict[str, list[dict[str, Any]]]:
    if not w:
        raise HTTPException(status_code=400, detail="Website 'w' is required")

    try:
        results = await service.get_website_terms(w)
        return {"terms": results}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


def setup_database(application: FastAPI) -> None:
    # Tortoise ORM Init with Connection Pooling
    config = Configuration.from_env()
    db_url = config.database_url
    # Append connection pool settings
    separator = "&" if "?" in db_url else "?"
    db_url += f"{separator}minsize=10&maxsize=50"

    register_tortoise(
        application,
        db_url=db_url,
        modules={"models": ["api.models"]},
        generate_schemas=False,
        add_exception_handlers=True,
    )


setup_database(app)
