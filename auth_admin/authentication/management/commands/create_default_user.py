from django.contrib.auth import get_user_model
from django.core.management.base import BaseCommand


class Command(BaseCommand):
    help = "Creates a default scraper user"

    from typing import Any

    def handle(self, *args: Any, **options: Any) -> None:
        User = get_user_model()
        if not User.objects.filter(username="scraper").exists():
            User.objects.create_user("scraper", "scraper@isidorus.com", "scraper")
            self.stdout.write(self.style.SUCCESS('Successfully created user "scraper"'))
        else:
            self.stdout.write(self.style.SUCCESS('User "scraper" already exists'))
