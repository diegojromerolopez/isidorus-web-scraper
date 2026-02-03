import os
import unittest
from unittest.mock import MagicMock, patch

from workers.image_extractor.services.explainer_factory import (
    ExplainerFactory,
    MockLLM,
)


class TestMockLLM(unittest.TestCase):
    def test_invoke(self) -> None:
        """Test MockLLM invoke returns expected response"""
        llm = MockLLM()
        response = llm.invoke("test prompt")
        self.assertEqual(response.content, "Mocked explanation for testing")


class TestExplainerFactory(unittest.TestCase):
    def test_get_explainer_mock(self) -> None:
        """Test factory returns MockLLM for 'mock' provider"""
        explainer = ExplainerFactory.get_explainer("mock")
        self.assertIsInstance(explainer, MockLLM)

    def test_get_explainer_mock_uppercase(self) -> None:
        """Test factory handles uppercase provider names"""
        explainer = ExplainerFactory.get_explainer("MOCK")
        self.assertIsInstance(explainer, MockLLM)

    @patch("workers.image_extractor.services.explainer_factory.ChatOpenAI")
    def test_get_explainer_openai_with_key(self, _mock_openai: MagicMock) -> None:
        """Test factory sets OPENAI_API_KEY when key is provided"""
        with patch.dict("os.environ", {}, clear=True):
            ExplainerFactory.get_explainer("openai", "test-key")

            self.assertEqual(os.environ["OPENAI_API_KEY"], "test-key")

    @patch("workers.image_extractor.services.explainer_factory.ChatGoogleGenerativeAI")
    def test_get_explainer_gemini_with_key(self, _mock_gemini: MagicMock) -> None:
        """Test factory sets GOOGLE_API_KEY when key is provided"""
        with patch.dict("os.environ", {}, clear=True):
            ExplainerFactory.get_explainer("gemini", "test-key")

            self.assertEqual(os.environ["GOOGLE_API_KEY"], "test-key")

    @patch("workers.image_extractor.services.explainer_factory.ChatAnthropic")
    def test_get_explainer_anthropic_with_key(self, _mock_anthropic: MagicMock) -> None:
        """Test factory sets ANTHROPIC_API_KEY when key is provided"""
        with patch.dict("os.environ", {}, clear=True):
            ExplainerFactory.get_explainer("anthropic", "test-key")

            self.assertEqual(os.environ["ANTHROPIC_API_KEY"], "test-key")

    @patch("workers.image_extractor.services.explainer_factory.ChatOpenAI")
    def test_get_explainer_openai(self, mock_openai: MagicMock) -> None:
        """Test factory returns ChatOpenAI for 'openai' provider"""
        mock_instance = MagicMock()
        mock_openai.return_value = mock_instance

        explainer = ExplainerFactory.get_explainer("openai")

        mock_openai.assert_called_once_with(model="gpt-4o-mini")
        self.assertEqual(explainer, mock_instance)

    @patch("workers.image_extractor.services.explainer_factory.ChatGoogleGenerativeAI")
    def test_get_explainer_gemini(self, mock_gemini: MagicMock) -> None:
        """Test factory returns ChatGoogleGenerativeAI for 'gemini' provider"""
        mock_instance = MagicMock()
        mock_gemini.return_value = mock_instance

        explainer = ExplainerFactory.get_explainer("gemini")

        mock_gemini.assert_called_once_with(model="gemini-pro-vision")
        self.assertEqual(explainer, mock_instance)

    @patch("workers.image_extractor.services.explainer_factory.ChatAnthropic")
    def test_get_explainer_anthropic(self, mock_anthropic: MagicMock) -> None:
        """Test factory returns ChatAnthropic for 'anthropic' provider"""
        mock_instance = MagicMock()
        mock_anthropic.return_value = mock_instance

        explainer = ExplainerFactory.get_explainer("anthropic")

        mock_anthropic.assert_called_once_with(model_name="claude-3-haiku-20240307")
        self.assertEqual(explainer, mock_instance)

    @patch("workers.image_extractor.services.explainer_factory.ChatOllama")
    def test_get_explainer_ollama(self, mock_ollama: MagicMock) -> None:
        """Test factory returns ChatOllama for 'ollama' provider"""
        mock_instance = MagicMock()
        mock_ollama.return_value = mock_instance

        explainer = ExplainerFactory.get_explainer("ollama")

        mock_ollama.assert_called_once_with(model="llama3")
        self.assertEqual(explainer, mock_instance)

    @patch("workers.image_extractor.services.explainer_factory.HuggingFaceEndpoint")
    def test_get_explainer_huggingface(self, mock_hf: MagicMock) -> None:
        """Test factory returns HuggingFaceEndpoint for 'huggingface' provider"""
        mock_instance = MagicMock()
        mock_hf.return_value = mock_instance

        explainer = ExplainerFactory.get_explainer("huggingface")

        mock_hf.assert_called_once_with(repo_id="mistralai/Mistral-7B-Instruct-v0.2")
        self.assertEqual(explainer, mock_instance)

    def test_get_explainer_unknown(self) -> None:
        """Test factory returns MockLLM for unknown provider"""
        explainer = ExplainerFactory.get_explainer("unknown_provider")
        self.assertIsInstance(explainer, MockLLM)

    def test_explain_image_success(self) -> None:
        """Test explain_image with successful LLM response"""
        mock_llm = MagicMock()
        mock_response = MagicMock()
        mock_response.content = "This is a test image"
        mock_llm.invoke.return_value = mock_response

        result = ExplainerFactory.explain_image(
            mock_llm, "http://example.com/image.jpg"
        )

        self.assertEqual(result, "This is a test image")
        mock_llm.invoke.assert_called_once_with(
            "Describe this image: http://example.com/image.jpg"
        )

    def test_explain_image_no_content_attribute(self) -> None:
        """Test explain_image when response has no content attribute"""
        mock_llm = MagicMock()
        mock_response = "Simple string response"
        mock_llm.invoke.return_value = mock_response

        result = ExplainerFactory.explain_image(
            mock_llm, "http://example.com/image.jpg"
        )

        self.assertEqual(result, "Simple string response")

    def test_explain_image_exception(self) -> None:
        """Test explain_image handles exceptions gracefully"""
        mock_llm = MagicMock()
        mock_llm.invoke.side_effect = Exception("LLM API error")

        result = ExplainerFactory.explain_image(
            mock_llm, "http://example.com/image.jpg"
        )

        self.assertEqual(result, "Explanation unavailable")


if __name__ == "__main__":
    unittest.main()
