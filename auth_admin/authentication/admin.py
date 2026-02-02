import hashlib
import secrets
from typing import Any

from django.contrib import admin, messages

from .models import APIKey


@admin.register(APIKey)
class APIKeyAdmin(admin.ModelAdmin):
    list_display = (
        "name",
        "user",
        "prefix",
        "is_active",
        "expires_at",
        "created_at",
        "last_used_at",
    )
    list_filter = ("is_active", "created_at", "expires_at", "user")
    search_fields = ("name", "prefix", "user__username")
    readonly_fields = ("prefix", "hashed_key", "last_used_at", "created_at")

    def save_model(self, request: Any, obj: APIKey, form: Any, change: bool) -> None:
        if not change:  # If creating a new APIKey
            # Generate a random key
            raw_key = secrets.token_urlsafe(32)
            obj.prefix = raw_key[:8]

            # Hash the key
            obj.hashed_key = hashlib.sha256(raw_key.encode()).hexdigest()

            # Save the object first to ensure everything is fine
            super().save_model(request, obj, form, change)

            # Use Django messages to show the key to the user (once)
            messages.success(
                request,
                "API Key created successfully! Please copy it now, "
                f"as it won't be shown again: {raw_key}",
            )
        else:
            super().save_model(request, obj, form, change)
