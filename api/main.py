from typing import Any

from fastapi import BackgroundTasks, Depends, FastAPI, HTTPException
from pydantic import BaseModel
from tortoise.contrib.fastapi import register_tortoise  # pylint: disable=import-error

from api.config import Configuration
from api.dependencies import (
    get_api_key,
    get_db_repository,
    get_db_service,
    get_scraper_service,
)
from api.models import APIKey
from api.repositories.db_repository import DbRepository
from api.services.db_service import DbService
from api.services.scraper_service import ScraperService

app = FastAPI()

from fastapi.middleware.cors import CORSMiddleware

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:8000", "http://localhost:8001"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


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
        # Extract user_id from the APIKey dependency
        user_id = _api_key.user_id if _api_key else None
        scraping_id = await scraper_service.start_scraping(
            request.url, request.depth, user_id
        )
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


@app.get("/scrapings")
async def list_scrapings(
    page: int = 1,
    size: int = 10,
    service: DbService = Depends(get_db_service),
    _api_key: APIKey = Depends(get_api_key),
) -> dict[str, Any]:
    """
    List scrapings for the authenticated user.
    """
    if not _api_key.user_id:
        # Should be handled by get_api_key usually, but for safety
        raise HTTPException(status_code=401, detail="User context required")

    try:
        offset = (page - 1) * size
        scrapings, total = await service.get_scrapings(
            user_id=_api_key.user_id, offset=offset, limit=size
        )
        return {
            "data": scrapings,
            "meta": {"page": page, "size": size, "total": total},
        }
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
@app.delete("/scrapings/{scraping_id}")
async def delete_scraping(
    scraping_id: int,
    service: ScraperService = Depends(get_scraper_service),
    db_repository: DbRepository = Depends(get_db_repository),
    _api_key: APIKey = Depends(get_api_key),
) -> dict[str, str]:
    """
    Deletes a scraping job.
    Sends a message to the Deletion worker to handle clean up asynchronously.
    """
    scraping = await db_repository.get_scraping(scraping_id)
    if not scraping:
        raise HTTPException(status_code=404, detail="Scraping not found")

    if scraping["user_id"] != _api_key.user_id:
        raise HTTPException(status_code=403, detail="Not authorized to delete this scraping")

    # Orchestrate deletion via Deletion worker
    success = await service.enqueue_deletion(scraping_id)
    if not success:
        raise HTTPException(status_code=500, detail="Failed to enqueue deletion")

    return {"message": "Scraping deletion enqueued"}
