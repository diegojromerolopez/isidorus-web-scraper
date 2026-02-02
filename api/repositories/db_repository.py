from typing import Any

from api import models


class DbRepository:
    async def find_websites_by_term(self, term: str) -> list[str]:
        """
        Finds all unique website URLs that contain the specified search term.
        """
        from typing import cast  # pylint: disable=import-outside-toplevel

        return cast(
            list[str],
            await models.ScrapedPage.filter(terms__term=term)
            .distinct()
            .values_list("url", flat=True),
        )

    async def find_terms_by_website(self, website_url: str) -> list[dict[str, Any]]:
        """
        Retrieves all terms and their occurrence counts for a specific website URL.
        """
        terms = await models.PageTerm.filter(page__url=website_url).values(
            "term", "frequency"
        )
        return [{"term": t["term"], "occurrence": t["frequency"]} for t in terms]

    async def create_scraping(self, url: str) -> int:
        """
        Creates a new scraping record.
        """
        scraping = await models.Scraping.create(url=url, status="PENDING")
        return int(scraping.id)

    async def get_scraping(self, scraping_id: int) -> dict[str, Any] | None:
        """
        Retrieves a scraping by ID.
        """
        scraping = await models.Scraping.get_or_none(id=scraping_id)
        if scraping:
            return {
                "id": scraping.id,
                "url": scraping.url,
                "status": scraping.status,
                "created_at": scraping.created_at,
                "completed_at": scraping.completed_at,
            }
        return None

    async def get_scrape_results(self, scraping_id: int) -> list[dict[str, Any]]:
        """
        Retrieves the scrape results (URLs, terms, and images) for a given scraping.
        """
        pages = (
            await models.ScrapedPage.filter(scraping_id=scraping_id)
            .order_by("url")
            .prefetch_related("terms", "images")
        )

        results = []
        for page in pages:
            # page.terms is a related manager/list
            terms_list = [
                {"term": t.term, "frequency": t.frequency} for t in page.terms
            ]
            images_list = [
                {"url": i.image_url, "explanation": i.explanation} for i in page.images
            ]

            results.append(
                {
                    "url": page.url,
                    "terms": terms_list,
                    "images": images_list,
                    "summary": page.summary,
                }
            )

        return results
