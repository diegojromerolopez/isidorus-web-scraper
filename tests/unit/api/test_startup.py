import os
import unittest
from unittest.mock import MagicMock, patch


class TestStartup(unittest.TestCase):
    @patch("tortoise.contrib.fastapi.register_tortoise")
    @patch.dict(
        os.environ, {"DATABASE_URL": "postgres://user:pass@host:5432/db"}, clear=True
    )
    def test_tortoise_init_default(self, mock_register: MagicMock) -> None:
        import importlib

        import api.main

        importlib.reload(api.main)

        mock_register.assert_called()
        call_args = mock_register.call_args
        self.assertIn("db_url", call_args[1])
        self.assertIn("minsize=10", call_args[1]["db_url"])

    @patch("tortoise.contrib.fastapi.register_tortoise")
    @patch.dict(
        os.environ,
        {"DATABASE_URL": "postgres://user:pass@host:5432/db?sslmode=disable"},
        clear=True,
    )
    def test_tortoise_init_existing_params(self, mock_register: MagicMock) -> None:
        import importlib

        import api.main

        importlib.reload(api.main)

        mock_register.assert_called()
        call_args = mock_register.call_args
        self.assertIn("&minsize=10", call_args[1]["db_url"])
