import logging
from typing import Any

# pylint: disable=import-error
from langchain.chains.summarize import load_summarize_chain
from langchain.docstore.document import Document
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_anthropic import ChatAnthropic
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_huggingface import HuggingFaceEndpoint
from langchain_community.chat_models import ChatOllama  # type: ignore
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
    def get_llm(provider: str = "openai") -> Any:
        provider = provider.lower()

        providers = {
            "mock": MockLLM,
            "openai": lambda: ChatOpenAI(model="gpt-3.5-turbo"),
            "gemini": lambda: ChatGoogleGenerativeAI(model="gemini-pro"),
            "anthropic": lambda: ChatAnthropic(model_name="claude-3-haiku-20240307"),
            "ollama": lambda: ChatOllama(model="llama3"),
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
    def summarize_text(llm: Any, text: str) -> str:
        try:
            if isinstance(llm, MockLLM):
                return "Mocked summary for testing"

            text_splitter = RecursiveCharacterTextSplitter(
                chunk_size=4000, chunk_overlap=200
            )
            docs = text_splitter.create_documents([text])

            # For very short texts, map_reduce might be overkill, "stuff" is better
            chain_type = "stuff" if len(docs) == 1 else "map_reduce"

            chain = load_summarize_chain(llm, chain_type=chain_type)
            return chain.run(docs)

        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("LLM summarization failed: %s", e)
            return "Summary unavailable"
