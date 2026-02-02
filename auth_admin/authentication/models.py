import uuid

from django.conf import settings
from django.db import models


class APIKey(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    user = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name="api_keys",
        help_text="The user this API Key belongs to",
    )
    name = models.CharField(
        max_length=100, unique=True, help_text="A descriptive name for the API Key"
    )
    prefix = models.CharField(max_length=8, editable=False)
    hashed_key = models.CharField(max_length=128, editable=False)
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    expires_at = models.DateTimeField(
        null=True, blank=True, help_text="Optional expiration date"
    )
    last_used_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        db_table = "api_keys"
        verbose_name = "API Key"
        verbose_name_plural = "API Keys"

    def __str__(self) -> str:
        return f"{self.name} ({self.prefix}...)"
