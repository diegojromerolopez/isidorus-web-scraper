import hashlib
from typing import Any

from django.contrib.auth.models import User
from django.core.management.base import BaseCommand

from authentication.models import APIKey


class Command(BaseCommand):
    help = "Seeds a default API key for testing"

    def handle(self, *args: Any, **options: Any) -> None:
        # Create a test user if it doesn't exist
        user, created = User.objects.get_or_create(
            username="test-user", defaults={"email": "test@example.com"}
        )
        if created:
            user.set_password("test-password")
            user.save()
            self.stdout.write(self.style.SUCCESS(f"Created test user: {user.username}"))

        # Create a fixed API key for testing
        # Key: "test-api-key-123"
        raw_key = "test-api-key-123"
        hashed_key = hashlib.sha256(raw_key.encode()).hexdigest()

        api_key, created = APIKey.objects.get_or_create(
            name="Default Test Key",
            defaults={
                "user": user,
                "prefix": raw_key[:8],
                "hashed_key": hashed_key,
                "is_active": True,
            },
        )

        if created:
            self.stdout.write(self.style.SUCCESS(f"Created default API Key: {raw_key}"))
        else:
            # Update it just in case
            api_key.hashed_key = hashed_key
            api_key.prefix = raw_key[:8]
            api_key.save()
            self.stdout.write(self.style.SUCCESS(f"Updated default API Key: {raw_key}"))
