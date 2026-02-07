import logging
import os
from typing import Any

from langchain_anthropic import ChatAnthropic  # pylint: disable=import-error

# pylint: disable=import-error
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_huggingface import HuggingFaceEndpoint
from langchain_ollama import ChatOllama  # type: ignore
from langchain_openai import ChatOpenAI  # type: ignore

logger = logging.getLogger(__name__)


class MockLLM:
    # pylint: disable=too-few-public-methods
    def invoke(self, _prompt: str) -> Any:
        class MockResponse:
            def __init__(self, content: str) -> None:
                self.content = content

        return MockResponse("Mocked summary for testing")

    def get_num_tokens(self, text: str) -> int:
        return len(text.split())


class SummarizerFactory:
    # pylint: disable=too-few-public-methods
    @staticmethod
    def get_llm(provider: str = "openai", api_key: str | None = None) -> Any:
        provider = provider.lower()

        if api_key:
            if provider == "openai":
                os.environ["OPENAI_API_KEY"] = api_key
            elif provider == "gemini":
                os.environ["GOOGLE_API_KEY"] = api_key
            elif provider == "anthropic":
                os.environ["ANTHROPIC_API_KEY"] = api_key
            # Add other mappings as needed

        providers = {
            "mock": MockLLM,
            "openai": lambda: ChatOpenAI(model="gpt-3.5-turbo"),
            "gemini": lambda: ChatGoogleGenerativeAI(model="gemini-pro"),
            "anthropic": lambda: ChatAnthropic(model_name="claude-3-haiku-20240307"),
            "ollama": lambda: ChatOllama(
                model="tinyllama",
                base_url=os.getenv("OLLAMA_BASE_URL", "http://localhost:11434"),
            ),
            "huggingface": lambda: HuggingFaceEndpoint(
                repo_id="mistralai/Mistral-7B-Instruct-v0.2"
            ),
        }

        if provider in providers:
            result = providers[provider]
            # Handle both class types and lambda factories
            return result()  # type: ignore[no-any-return]

        logger.warning("Unknown provider '%s', falling back to Mock provider", provider)
        return MockLLM()

    @staticmethod
    async def summarize_text(llm: Any, text: str) -> str:
        try:
            if isinstance(llm, MockLLM):
                return "Mocked summary for testing"

            # 1. Truncate text to avoid huge prompts and long processing
            # tinyllama and other small models have 2048-4096 context window
            # We truncate to ~1500 words to be safe
            words = text.split()
            if len(words) > 1500:
                text = " ".join(words[:1500]) + "..."

            # 2. Simple prompt for speed instead of complex map_reduce
            prompt = (
                "Write a concise summary of the following web page content. "
                "Do not repeat these instructions. Do not start with "
                "'Summarize the...'. "
                "Provide only the summary itself.\n\n"
                f"Content: {text}\n\n"
                "Summary:"
            )

            # 3. Use ainvoke for non-blocking I/O
            logger.info("Sending prompt to LLM (Length: %d)", len(prompt))
            response = await llm.ainvoke(prompt)
            content = (
                response.content if hasattr(response, "content") else str(response)
            )
            logger.info("Received response from LLM (Length: %d)", len(content))
            return content

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("LLM summarization failed: %s", e)
            return "Summary unavailable"
