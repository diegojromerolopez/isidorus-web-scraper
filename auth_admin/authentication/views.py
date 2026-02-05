import hashlib
import secrets
import uuid
from typing import Any

from django.contrib.auth import authenticate
from rest_framework import status
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import APIKey


class LoginAPIView(APIView):
    authentication_classes: list[Any] = []
    permission_classes: list[Any] = []

    def post(self, request: Any) -> Response:
        username = request.data.get("username")
        password = request.data.get("password")

        user = authenticate(username=username, password=password)

        if user is not None:
            # Generate a new API Key
            prefix = secrets.token_hex(4)
            raw_key = secrets.token_hex(32)
            key_string = f"{prefix}.{raw_key}"
            hashed_key = hashlib.sha256(key_string.encode()).hexdigest()

            # Create the key record
            APIKey.objects.create(
                user=user,
                name=f"Key for {username} - {uuid.uuid4().hex[:8]}",
                prefix=prefix,
                hashed_key=hashed_key,
            )

            return Response({"api_key": key_string}, status=status.HTTP_200_OK)

        return Response(
            {"error": "Invalid credentials"}, status=status.HTTP_401_UNAUTHORIZED
        )
