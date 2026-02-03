import os
import unittest
from unittest.mock import MagicMock, patch

from workers.page_summarizer.services.summarizer_factory import (
    MockLLM,
    SummarizerFactory,
)


class TestSummarizerFactory(unittest.TestCase):
    def setUp(self) -> None:
        # Clear specific env vars before each test
        self.env_vars_to_clear = [
            "OPENAI_API_KEY",
            "GOOGLE_API_KEY",
            "ANTHROPIC_API_KEY",
        ]
        for var in self.env_vars_to_clear:
            if var in os.environ:
                del os.environ[var]

    @patch.dict(os.environ, {}, clear=True)
    def test_get_llm_openai_with_key(self) -> None:
        with patch(
            "workers.page_summarizer.services.summarizer_factory.ChatOpenAI"
        ) as mock_openai:
            SummarizerFactory.get_llm("openai", "sk-test")
            self.assertEqual(os.environ["OPENAI_API_KEY"], "sk-test")
            mock_openai.assert_called_once()

    @patch.dict(os.environ, {}, clear=True)
    def test_get_llm_gemini_with_key(self) -> None:
        with patch(
            "workers.page_summarizer.services.summarizer_factory.ChatGoogleGenerativeAI"
        ) as mock_gemini:
            SummarizerFactory.get_llm("gemini", "key-test")
            self.assertEqual(os.environ["GOOGLE_API_KEY"], "key-test")
            mock_gemini.assert_called_once()

    @patch.dict(os.environ, {}, clear=True)
    def test_get_llm_anthropic_with_key(self) -> None:
        with patch(
            "workers.page_summarizer.services.summarizer_factory.ChatAnthropic"
        ) as mock_anthropic:
            SummarizerFactory.get_llm("anthropic", "key-test")
            self.assertEqual(os.environ["ANTHROPIC_API_KEY"], "key-test")
            mock_anthropic.assert_called_once()

    def test_get_llm_mock(self) -> None:
        llm = SummarizerFactory.get_llm("mock")
        self.assertIsInstance(llm, MockLLM)

    def test_get_llm_unknown(self) -> None:
        llm = SummarizerFactory.get_llm("unknown_provider")
        self.assertIsInstance(llm, MockLLM)

    def test_mock_llm_invoke(self) -> None:
        llm = MockLLM()
        res = llm.invoke("prompt")
        self.assertEqual(res.content, "Mocked summary for testing")

    def test_mock_llm_get_num_tokens(self) -> None:
        llm = MockLLM()
        count = llm.get_num_tokens("hello world")
        self.assertEqual(count, 2)

    def test_summarize_text_mock(self) -> None:
        llm = MockLLM()
        summary = SummarizerFactory.summarize_text(llm, "some text")
        self.assertEqual(summary, "Mocked summary for testing")

    @patch("workers.page_summarizer.services.summarizer_factory.load_summarize_chain")
    @patch(
        "workers.page_summarizer.services.summarizer_factory."
        "RecursiveCharacterTextSplitter"
    )
    def test_summarize_text_real_stuff(
        self, mock_splitter_cls: MagicMock, mock_load: MagicMock
    ) -> None:
        # Mock LLM (any non-MockLLM object)
        llm = MagicMock()

        # Mock Text Splitter
        mock_splitter = MagicMock()
        mock_splitter_cls.return_value = mock_splitter
        mock_doc = MagicMock()
        # Return 1 doc -> stuff chain
        mock_splitter.create_documents.return_value = [mock_doc]

        # Mock Chain
        mock_chain = MagicMock()
        mock_load.return_value = mock_chain
        mock_chain.run.return_value = "Generated Summary"

        summary = SummarizerFactory.summarize_text(llm, "content")

        self.assertEqual(summary, "Generated Summary")
        mock_load.assert_called_with(llm, chain_type="stuff")

    @patch("workers.page_summarizer.services.summarizer_factory.load_summarize_chain")
    @patch(
        "workers.page_summarizer.services.summarizer_factory."
        "RecursiveCharacterTextSplitter"
    )
    def test_summarize_text_real_map_reduce(
        self, mock_splitter_cls: MagicMock, mock_load: MagicMock
    ) -> None:
        llm = MagicMock()
        mock_splitter = MagicMock()
        mock_splitter_cls.return_value = mock_splitter
        # Return 2 docs -> map_reduce
        mock_splitter.create_documents.return_value = [MagicMock(), MagicMock()]

        mock_chain = MagicMock()
        mock_load.return_value = mock_chain
        mock_chain.run.return_value = "Map Reduced Summary"

        summary = SummarizerFactory.summarize_text(llm, "content")

        self.assertEqual(summary, "Map Reduced Summary")
        mock_load.assert_called_with(llm, chain_type="map_reduce")

    def test_summarize_text_exception(self) -> None:
        llm = MagicMock()
        # Pass an object that causes error in splitter or something
        with patch(
            "workers.page_summarizer.services.summarizer_factory."
            "RecursiveCharacterTextSplitter",
            side_effect=Exception("Error"),
        ):
            summary = SummarizerFactory.summarize_text(llm, "content")
            self.assertEqual(summary, "Summary unavailable")
