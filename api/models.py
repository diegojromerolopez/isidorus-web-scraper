# pylint: disable=import-error,too-few-public-methods
from tortoise import fields, models  # type: ignore


class Scraping(models.Model):
    id = fields.IntField(pk=True)
    user_id = fields.IntField(null=True)
    url = fields.TextField()

    class Meta:
        table = "scrapings"


class ScrapedPage(models.Model):
    id = fields.IntField(pk=True)
    scraping = fields.ForeignKeyField(
        "models.Scraping", related_name="pages", source_field="scraping_id"
    )
    url = fields.TextField()
    summary = fields.TextField(null=True)
    scraped_at = fields.DatetimeField(auto_now_add=True)

    class Meta:
        table = "scraped_pages"
        indexes = [("scraping", "url")]


class PageLink(models.Model):
    id = fields.IntField(pk=True)
    scraping = fields.ForeignKeyField(
        "models.Scraping", related_name="links", source_field="scraping_id"
    )
    source_page = fields.ForeignKeyField(
        "models.ScrapedPage", related_name="links", source_field="source_page_id"
    )
    target_url = fields.TextField()

    class Meta:
        table = "page_links"


class PageImage(models.Model):
    id = fields.IntField(pk=True)
    scraping = fields.ForeignKeyField(
        "models.Scraping", related_name="images", source_field="scraping_id"
    )
    page = fields.ForeignKeyField(
        "models.ScrapedPage", related_name="images", source_field="page_id"
    )
    image_url = fields.TextField()
    explanation = fields.TextField(null=True)
    s3_path = fields.TextField(null=True)

    class Meta:
        table = "page_images"


class APIKey(models.Model):
    id = fields.UUIDField(pk=True)
    user_id = fields.IntField()  # Store Django user ID
    name = fields.CharField(max_length=100, unique=True)
    prefix = fields.CharField(max_length=8)
    hashed_key = fields.CharField(max_length=128)
    is_active = fields.BooleanField(default=True)
    created_at = fields.DatetimeField(auto_now_add=True)
    expires_at = fields.DatetimeField(null=True)
    last_used_at = fields.DatetimeField(null=True)

    class Meta:
        table = "api_keys"
