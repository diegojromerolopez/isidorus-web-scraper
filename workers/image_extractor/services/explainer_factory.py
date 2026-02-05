import logging
import os
from typing import Any

# pylint: disable=import-error
from langchain_anthropic import ChatAnthropic
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

        return MockResponse("Mocked explanation for testing")


class ExplainerFactory:
    # pylint: disable=too-few-public-methods
    @staticmethod
    def get_explainer(provider: str = "openai", api_key: str | None = None) -> Any:
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
            "openai": lambda: ChatOpenAI(model="gpt-4o-mini"),
            "gemini": lambda: ChatGoogleGenerativeAI(model="gemini-pro-vision"),
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
            return result()

        logger.warning("Unknown provider '%s', falling back to Mock provider", provider)
        return MockLLM()

    @staticmethod
    def explain_image(llm: Any, image_url: str) -> str:
        # Simplified LangChain call for image explanation
        # Note: Vision models usually take base64 or URL depending on provider
        try:
            # We use a simple prompt for now
            response = llm.invoke(f"Describe this image: {image_url}")
            return response.content if hasattr(response, "content") else str(response)
        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("LLM explanation failed: %s", e)
            return "Explanation unavailable"
