import json
import logging

from shared.clients.sqs_client import SQSClient
from workers.page_summarizer.services.summarizer_factory import SummarizerFactory

logger = logging.getLogger(__name__)


class SummarizerService:
    # pylint: disable=too-few-public-methods
    def __init__(
        self,
        sqs_client: SQSClient,
        writer_queue_url: str,
        llm_provider: str = "openai",
    ):
        self.__sqs_client = sqs_client
        self.__writer_queue_url = writer_queue_url
        self.__llm = SummarizerFactory.get_llm(llm_provider)

    async def process_message(self, message_body: str) -> None:
        try:
            body = json.loads(message_body)
            scraping_id = body.get("scraping_id")
            page_id = body.get("page_id")  # Optional, might not be known by scraper
            url = body.get("url")
            content = body.get("content")

            if not scraping_id or not content:
                logger.warning("Missing required fields (scraping_id, content)")
                return

            logger.info("Summarizing page: %s for scraping %s", url, scraping_id)

            # Generate Summary
            summary = SummarizerFactory.summarize_text(self.__llm, content)
            logger.info("Generated summary for %s", url)

            # Send to Writer
            writer_msg = {
                "type": "page_summary",
                "scraping_id": scraping_id,
                "url": url,
                "summary": summary,
            }

            if page_id:
                writer_msg["page_id"] = page_id

            await self.__sqs_client.send_message(writer_msg, self.__writer_queue_url)
            logger.info("Sent summary for %s to writer queue", url)

        except Exception as e:  # pylint: disable=broad-exception-caught
            # Catch-all to prevent worker crash on single message failure
            logger.error("Error processing page summary: %s", e)
