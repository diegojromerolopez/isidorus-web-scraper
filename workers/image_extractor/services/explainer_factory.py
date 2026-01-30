import logging
from typing import Any

from langchain_anthropic import ChatAnthropic  # type: ignore
from langchain_google_genai import ChatGoogleGenerativeAI  # type: ignore
from langchain_huggingface import HuggingFaceEndpoint  # type: ignore
from langchain_ollama import ChatOllama  # type: ignore
from langchain_openai import ChatOpenAI  # type: ignore

logger = logging.getLogger(__name__)


class MockLLM:
    def invoke(self, prompt: str) -> Any:
        class MockResponse:
            def __init__(self, content: str) -> None:
                self.content = content

        return MockResponse("Mocked explanation for testing")


class ExplainerFactory:
    @staticmethod
    def get_explainer(provider: str = "openai") -> Any:
        provider = provider.lower()

        if provider == "mock":
            return MockLLM()
        elif provider == "openai":
            return ChatOpenAI(model="gpt-4o-mini")
        elif provider == "gemini":
            return ChatGoogleGenerativeAI(model="gemini-pro-vision")
        elif provider == "anthropic":
            return ChatAnthropic(model_name="claude-3-haiku-20240307")
        elif provider == "ollama":
            return ChatOllama(model="llama3")
        elif provider == "huggingface":
            return HuggingFaceEndpoint(repo_id="mistralai/Mistral-7B-Instruct-v0.2")
        else:
            logger.warning(
                f"Unknown provider '{provider}', falling back to Mock provider"
            )
            return MockLLM()

    @staticmethod
    def explain_image(llm: Any, image_url: str) -> str:
        # Simplified LangChain call for image explanation
        # Note: Vision models usually take base64 or URL depending on provider
        try:
            # We use a simple prompt for now
            response = llm.invoke(f"Describe this image: {image_url}")
            return response.content if hasattr(response, "content") else str(response)
        except Exception as e:
            logger.error(f"LLM explanation failed: {e}")
            return "Explanation unavailable"
