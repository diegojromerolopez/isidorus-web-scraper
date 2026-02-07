import json
import logging

from shared.clients.sqs_client import SQSClient
from workers.page_summarizer.services.summarizer_factory import SummarizerFactory

logger = logging.getLogger(__name__)


class SummarizerService:
    # pylint: disable=too-few-public-methods
    # pylint: disable=too-many-arguments,too-many-positional-arguments
    def __init__(
        self,
        sqs_client: SQSClient,
        writer_queue_url: str,
        indexer_queue_url: str | None = None,
        llm_provider: str = "openai",
        llm_api_key: str | None = None,
    ):
        self.__sqs_client = sqs_client
        self.__writer_queue_url = writer_queue_url
        self.__indexer_queue_url = indexer_queue_url
        self.__llm = SummarizerFactory.get_llm(llm_provider, llm_api_key)

    async def process_message(self, message_body: str) -> None:
        try:
            body = json.loads(message_body)
            scraping_id = body.get("scraping_id")
            user_id = body.get("user_id")
            url = body.get("url")
            content = body.get("content")

            if not scraping_id or not content:
                logger.warning("Missing required fields (scraping_id, content)")
                return

            logger.info("Summarizing page: %s for scraping %s", url, scraping_id)

            # Generate Summary
            summary = await SummarizerFactory.summarize_text(self.__llm, content)
            logger.info("Generated summary for %s", url)

            # Send to Writer
            writer_msg = {
                "type": "page_summary",
                "scraping_id": scraping_id,
                "url": url,
                "summary": summary,
            }

            await self.__sqs_client.send_message(writer_msg, self.__writer_queue_url)
            logger.info("Sent summary for %s to writer queue", url)

            # Send to Indexer
            if self.__indexer_queue_url:
                indexer_msg = {
                    "url": url,
                    "content": content,
                    "summary": summary,
                    "scraping_id": scraping_id,
                    "user_id": user_id,
                }
                await self.__sqs_client.send_message(
                    indexer_msg, self.__indexer_queue_url
                )
                logger.info("Sent data for %s to indexer queue", url)

        except Exception as e:  # pylint: disable=broad-exception-caught
            # Catch-all to prevent worker crash on single message failure
            logger.error("Error processing page summary: %s", e)
