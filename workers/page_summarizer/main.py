import asyncio
import logging

from shared.clients.otel_client import init_telemetry
from shared.clients.sqs_client import SQSClient
from workers.page_summarizer.config import Configuration
from workers.page_summarizer.services.summarizer_service import SummarizerService

init_telemetry("page-summarizer")

# Logging setup
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("Page Summarizer Worker started")

    # Configuration
    config = Configuration.from_env()

    if not config.input_queue_url or not config.writer_queue_url:
        logger.error("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
        return

    # Dependency Injection
    sqs_client = SQSClient.create(config)

    summarizer_service = SummarizerService(
        sqs_client=sqs_client,
        writer_queue_url=config.writer_queue_url,
        indexer_queue_url=config.indexer_queue_url,
        llm_provider=config.llm_provider,
        llm_api_key=config.llm_api_key,
    )

    # Main Loop
    while True:
        try:
            messages = await sqs_client.receive_messages(config.input_queue_url)
            for message in messages:
                # Extract and activate OpenTelemetry context
                from opentelemetry import context, propagate

                token = None
                try:
                    # Extract trace context from SQS MessageAttributes
                    message_attributes = message.get("MessageAttributes", {})
                    trace_context = {
                        k: v["StringValue"]
                        for k, v in message_attributes.items()
                        if "StringValue" in v
                    }
                    if trace_context:
                        extracted_context = propagate.extract(trace_context)
                        token = context.attach(extracted_context)
                except Exception:  # pylint: disable=broad-exception-caught
                    pass

                try:
                    await summarizer_service.process_message(message["Body"])
                finally:
                    if token is not None:
                        context.detach(token)

                await sqs_client.delete_message(
                    config.input_queue_url, message["ReceiptHandle"]
                )

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Polling error: %s", e)
            await asyncio.sleep(1)


if __name__ == "__main__":  # pragma: no cover
    asyncio.run(main())
