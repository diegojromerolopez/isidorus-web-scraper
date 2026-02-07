import os
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

from workers.image_explainer.services.explainer_factory import (
    ExplainerFactory,
    MockLLM,
)


class TestExplainerFactory(unittest.IsolatedAsyncioTestCase):
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
    def test_get_explainer_openai_with_key(self) -> None:
        with patch(
            "workers.image_explainer.services.explainer_factory.ChatOpenAI"
        ) as mock_openai:
            ExplainerFactory.get_explainer("openai", "sk-test")
            self.assertEqual(os.environ["OPENAI_API_KEY"], "sk-test")
            mock_openai.assert_called_once()

    @patch.dict(os.environ, {}, clear=True)
    def test_get_explainer_gemini_with_key(self) -> None:
        with patch(
            "workers.image_explainer.services.explainer_factory.ChatGoogleGenerativeAI"
        ) as mock_gemini:
            ExplainerFactory.get_explainer("gemini", "key-test")
            self.assertEqual(os.environ["GOOGLE_API_KEY"], "key-test")
            mock_gemini.assert_called_once()

    @patch.dict(os.environ, {}, clear=True)
    def test_get_explainer_anthropic_with_key(self) -> None:
        with patch(
            "workers.image_explainer.services.explainer_factory.ChatAnthropic"
        ) as mock_anthropic:
            ExplainerFactory.get_explainer("anthropic", "key-test")
            self.assertEqual(os.environ["ANTHROPIC_API_KEY"], "key-test")
            mock_anthropic.assert_called_once()

    def test_get_explainer_mock(self) -> None:
        llm = ExplainerFactory.get_explainer("mock")
        self.assertIsInstance(llm, MockLLM)

    def test_get_explainer_unknown(self) -> None:
        llm = ExplainerFactory.get_explainer("unknown_provider")
        self.assertIsInstance(llm, MockLLM)

    def test_mock_llm_invoke(self) -> None:
        llm = MockLLM()
        res = llm.invoke("prompt")
        self.assertEqual(res.content, "Mocked explanation for testing")

    @patch("workers.image_explainer.services.explainer_factory.ChatOllama")
    def test_get_explainer_ollama(self, mock_ollama: MagicMock) -> None:
        mock_instance = MagicMock()
        mock_ollama.return_value = mock_instance

        llm = ExplainerFactory.get_explainer("ollama")

        mock_ollama.assert_called_once()
        self.assertEqual(llm, mock_instance)

    @patch("workers.image_explainer.services.explainer_factory.HuggingFaceEndpoint")
    def test_get_explainer_huggingface(self, mock_hf: MagicMock) -> None:
        mock_instance = MagicMock()
        mock_hf.return_value = mock_instance

        llm = ExplainerFactory.get_explainer("huggingface")

        mock_hf.assert_called_once()
        self.assertEqual(llm, mock_instance)

    async def test_explain_image_mock(self) -> None:
        llm = MockLLM()
        explanation = await ExplainerFactory.explain_image(
            llm, "data:image/jpeg;base64,..."
        )
        self.assertEqual(explanation, "Mocked explanation for testing")

    async def test_explain_image_real_llm(self) -> None:
        llm = MagicMock()
        llm.ainvoke = AsyncMock()
        mock_response = MagicMock()
        mock_response.content = "Real explanation"
        llm.ainvoke.return_value = mock_response

        # Ensure it doesn't match tinyllama
        llm.model = "gpt-4o"

        explanation = await ExplainerFactory.explain_image(
            llm, "data:image/jpeg;base64,..."
        )
        self.assertEqual(explanation, "Real explanation")
        llm.ainvoke.assert_called_once()

    async def test_explain_image_tinyllama(self) -> None:
        llm = MagicMock()
        llm.model = "tinyllama"

        explanation = await ExplainerFactory.explain_image(
            llm, "data:image/jpeg;base64,..."
        )
        self.assertIn("does not support vision", explanation)

    async def test_explain_image_exception(self) -> None:
        llm = MagicMock()
        llm.model = "some-model"
        llm.ainvoke = AsyncMock(side_effect=Exception("LLM Error"))

        explanation = await ExplainerFactory.explain_image(
            llm, "data:image/jpeg;base64,..."
        )
        self.assertEqual(explanation, "Explanation unavailable")


if __name__ == "__main__":
    unittest.main()
