import os
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.page_summarizer.services.summarizer_factory import (
    MockLLM,
    SummarizerFactory,
)


class TestSummarizerFactory(unittest.IsolatedAsyncioTestCase):
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

    @patch("workers.page_summarizer.services.summarizer_factory.ChatOllama")
    def test_get_llm_ollama(self, mock_ollama: MagicMock) -> None:
        """Test factory returns ChatOllama for 'ollama' provider"""
        mock_instance = MagicMock()
        mock_ollama.return_value = mock_instance

        llm = SummarizerFactory.get_llm("ollama")

        mock_ollama.assert_called_once_with(
            model="tinyllama", base_url="http://localhost:11434"
        )
        self.assertEqual(llm, mock_instance)

    @patch("workers.page_summarizer.services.summarizer_factory.HuggingFaceEndpoint")
    def test_get_llm_huggingface(self, mock_hf: MagicMock) -> None:
        """Test factory returns HuggingFaceEndpoint for 'huggingface' provider"""
        mock_instance = MagicMock()
        mock_hf.return_value = mock_instance

        llm = SummarizerFactory.get_llm("huggingface")

        mock_hf.assert_called_once_with(repo_id="mistralai/Mistral-7B-Instruct-v0.2")
        self.assertEqual(llm, mock_instance)

    def test_mock_llm_get_num_tokens(self) -> None:
        llm = MockLLM()
        count = llm.get_num_tokens("hello world")
        self.assertEqual(count, 2)

    async def test_summarize_text_mock(self) -> None:
        llm = MockLLM()
        summary = await SummarizerFactory.summarize_text(llm, "some text")
        self.assertEqual(summary, "Mocked summary for testing")

    async def test_summarize_text_real_llm(self) -> None:
        # Mock LLM (any non-MockLLM object)
        llm = MagicMock()
        llm.ainvoke = AsyncMock()

        mock_response = MagicMock()
        mock_response.content = "Generated Summary"
        llm.ainvoke.return_value = mock_response

        summary = await SummarizerFactory.summarize_text(llm, "content")

        self.assertEqual(summary, "Generated Summary")
        llm.ainvoke.assert_called_once()
        # Verify content was truncated or passed as is if short
        call_args = llm.ainvoke.call_args[0][0]
        self.assertIn("Write a concise summary", call_args)
        self.assertIn("Do not repeat these instructions", call_args)
        self.assertIn("Content: content", call_args)
        self.assertIn("Summary:", call_args)

    async def test_summarize_text_truncation(self) -> None:
        llm = MagicMock()
        llm.ainvoke = AsyncMock()

        mock_response = MagicMock()
        mock_response.content = "Short Summary"
        llm.ainvoke.return_value = mock_response

        # Long text (1600 words)
        long_content = "word " * 1600
        summary = await SummarizerFactory.summarize_text(llm, long_content)

        self.assertEqual(summary, "Short Summary")
        call_args = llm.ainvoke.call_args[0][0]
        # Verify truncation (ends with ...)
        self.assertIn("word...", call_args)
        # Check that it roughly has 1500 words of content
        content_part = call_args.split("Content: ")[1].split("\n\n")[0]
        self.assertLess(len(content_part.split()), 1510)

    async def test_summarize_text_exception(self) -> None:
        llm = MagicMock()
        llm.ainvoke = AsyncMock(side_effect=Exception("Error"))

        summary = await SummarizerFactory.summarize_text(llm, "content")
        self.assertEqual(summary, "Summary unavailable")
