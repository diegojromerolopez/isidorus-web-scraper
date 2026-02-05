from typing import TypedDict

from fastapi import Depends, FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from tortoise.contrib.fastapi import register_tortoise  # pylint: disable=import-error

from api.config import Configuration
from api.dependencies import (
    get_api_key,
    get_db_service,
    get_scraper_service,
)
from api.models import APIKey
from api.repositories.db_repository import TermOccurrence
from api.services.db_service import DbService
from api.services.scraper_service import (
    FullScrapingRecord,
    NotAuthorizedError,
    ScraperService,
    ScrapingNotFoundError,
)

app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "http://localhost:3000",
        "http://localhost:8000",
        "http://localhost:8001",
    ],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class ScrapeRequest(BaseModel):
    url: str
    depth: int = 1


class MessageResponse(TypedDict):
    message: str


class ScrapingsMeta(TypedDict):
    page: int
    size: int
    total: int


class ScrapingsResponse(TypedDict):
    scrapings: list[FullScrapingRecord]
    meta: ScrapingsMeta


class SearchResponse(TypedDict):
    websites: list[str]


class StatusResponse(TypedDict):
    status: str


class TermsResponse(TypedDict):
    terms: list[TermOccurrence]


class ScrapeResponse(TypedDict):
    scraping_id: int


class ScrapingResponse(TypedDict):
    scraping: FullScrapingRecord


@app.get("/health")
async def health_check() -> StatusResponse:
    return {"status": "ok"}


@app.post("/scrape")
async def scrape(
    request: ScrapeRequest,
    scraper_service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
) -> ScrapeResponse:
    try:
        # Extract user_id from the APIKey dependency
        user_id = _api_key.user_id if _api_key else None
        scraping_id = await scraper_service.start_scraping(
            request.url, request.depth, user_id
        )
        return {"scraping_id": scraping_id}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/scraping/{scraping_id}")
async def scraping(
    scraping_id: int,
    service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
) -> ScrapingResponse:
    try:
        full_scraping = await service.get_full_scraping(scraping_id)
        if not full_scraping:
            raise HTTPException(status_code=404, detail="Scraping not found")

        pages = []
        if full_scraping["status"] == "COMPLETED":
            pages = await service.get_scraping_results(scraping_id)

        full_scraping["pages"] = pages
        return {"scraping": full_scraping}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/scrapings")
async def scrapings(
    page: int = 1,
    size: int = 10,
    service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
) -> ScrapingsResponse:
    """
    List scrapings for the authenticated user.
    """
    if not _api_key.user_id:
        # Should be handled by get_api_key usually, but for safety
        raise HTTPException(status_code=401, detail="User context required")

    try:
        offset = (page - 1) * size
        full_scrapings, total = await service.get_full_scrapings(
            user_id=_api_key.user_id, offset=offset, limit=size
        )
        return {
            "scrapings": full_scrapings,
            "meta": {"page": page, "size": size, "total": total},
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@app.get("/search")
async def search(
    t: str,
    service: DbService = Depends(get_db_service),
    _api_key: APIKey = Depends(get_api_key),
) -> SearchResponse:
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
) -> TermsResponse:
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


@app.delete("/scraping/{scraping_id}")
async def delete_scraping(
    scraping_id: int,
    service: ScraperService = Depends(get_scraper_service),
    _api_key: APIKey = Depends(get_api_key),
) -> MessageResponse:
    """
    Deletes a scraping job.
    Delegates existence and ownership checks to the ScraperService.
    """
    try:
        success = await service.delete_scraping(scraping_id, _api_key.user_id)
        if not success:
            raise HTTPException(status_code=500, detail="Failed to enqueue deletion")
        return {"message": "Scraping deletion enqueued"}
    except ScrapingNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    except NotAuthorizedError as e:
        raise HTTPException(status_code=403, detail=str(e)) from e
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e
