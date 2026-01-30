from tortoise import fields, models  # type: ignore


class Scraping(models.Model):
    id = fields.IntField(pk=True)
    url = fields.TextField()
    status = fields.TextField(default="PENDING")
    created_at = fields.DatetimeField(auto_now_add=True)
    completed_at = fields.DatetimeField(null=True)

    class Meta:
        table = "scrapings"


class ScrapedPage(models.Model):
    id = fields.IntField(pk=True)
    scraping = fields.ForeignKeyField(
        "models.Scraping", related_name="pages", source_field="scraping_id"
    )
    url = fields.TextField()
    scraped_at = fields.DatetimeField(auto_now_add=True)

    class Meta:
        table = "scraped_pages"


class PageTerm(models.Model):
    id = fields.IntField(pk=True)
    scraping = fields.ForeignKeyField(
        "models.Scraping", related_name="terms", source_field="scraping_id"
    )
    page = fields.ForeignKeyField(
        "models.ScrapedPage", related_name="terms", source_field="page_id"
    )
    term = fields.TextField()
    frequency = fields.IntField(default=1)

    class Meta:
        table = "page_terms"


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
