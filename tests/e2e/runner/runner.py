# pylint: disable=duplicate-code
import os
import time
import unittest
from typing import Any, cast
from urllib.parse import urlparse

import boto3
import psycopg2
import redis
import requests

# Configuration
API_URL = os.getenv("API_URL", "http://api:8000")
DATABASE_URL = os.getenv(
    "DATABASE_URL", "postgres://postgres:postgres@postgres:5432/isidorus"
)
REDIS_HOST = os.getenv("REDIS_HOST", "redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localstack:4566")
AWS_REGION = os.getenv("AWS_REGION", "us-east-1")
AWS_ACCESS_KEY_ID = os.getenv("AWS_ACCESS_KEY_ID", "test")
AWS_SECRET_ACCESS_KEY = os.getenv("AWS_SECRET_ACCESS_KEY", "test")
API_KEY = os.getenv("API_KEY", "test-api-key-123")


class TestScrapingE2E(unittest.TestCase):
    def setUp(self) -> None:
        self.session = requests.Session()
        self.session.headers.update({"X-API-Key": API_KEY})
        self.wait_for_db()
        self.wait_for_sqs()
        self.cleanup_database()
        self.cleanup_redis()
        self.wait_for_api()

    def wait_for_sqs(self) -> None:
        print("Waiting for SQS queues to be ready...")
        sqs = boto3.client(
            "sqs",
            endpoint_url=AWS_ENDPOINT_URL,
            region_name=AWS_REGION,
            aws_access_key_id=AWS_ACCESS_KEY_ID,
            aws_secret_access_key=AWS_SECRET_ACCESS_KEY,
        )
        required_queues = [
            "scraper-queue",
            "image-extractor-queue",
            "writer-queue",
            "page-summarizer-queue",
        ]

        for _ in range(30):
            try:
                resp = sqs.list_queues()
                urls = resp.get("QueueUrls", [])
                # Flatten the check to reduce nesting
                found_count = sum(
                    1 for q_url in urls if any(req in q_url for req in required_queues)
                )

                if found_count >= len(required_queues):
                    print("SQS queues are ready!")
                    return
            except Exception as e:  # pylint: disable=broad-exception-caught
                print(f"Error checking SQS: {e}")
            time.sleep(1)
        self.fail("SQS queues failed to become ready.")

    def wait_for_db(self) -> None:
        """Wait until the Postgres database is ready to accept connections."""
        print("Waiting for Database to be ready...")
        result = urlparse(DATABASE_URL)
        username = result.username
        password = result.password
        database = result.path[1:]
        hostname = result.hostname
        port = result.port

        for _ in range(30):
            try:
                conn = psycopg2.connect(
                    dbname=database,
                    user=username,
                    password=password,
                    host=hostname,
                    port=port,
                )
                conn.close()
                print("Database is ready!")
                return
            except psycopg2.OperationalError:
                time.sleep(1)
        self.fail("Database failed to become ready.")

    def cleanup_database(self) -> None:
        """Truncate all relevant tables in the database."""
        try:
            # Parse DB URL
            result = urlparse(DATABASE_URL)
            username = result.username
            password = result.password
            database = result.path[1:]
            hostname = result.hostname
            port = result.port

            conn = psycopg2.connect(
                dbname=database,
                user=username,
                password=password,
                host=hostname,
                port=port,
            )
            cur = conn.cursor()
            cur.execute(
                "TRUNCATE TABLE page_images, page_links, page_terms, "
                "scraped_pages, scrapings CASCADE;"
            )
            conn.commit()
            cur.close()
            conn.close()
            print("Database cleaned.")
        except Exception as e:  # pylint: disable=broad-exception-caught
            print(f"Error cleaning database: {e}")

    def cleanup_redis(self) -> None:
        """Flush all keys in Redis."""
        try:
            r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
            r.flushall()
            print("Redis cleaned.")
        except Exception as e:  # pylint: disable=broad-exception-caught
            print(f"Error cleaning Redis: {e}")

    def wait_for_api(self) -> None:
        """Wait until the API is responsive."""
        for _ in range(30):
            try:
                # /docs does not require auth usually but let's be safe
                self.session.get(f"{API_URL}/docs", timeout=5)
                return
            except requests.RequestException:
                time.sleep(1)
        self.fail("API failed to become ready.")

    def _trigger_scraping(self, url: str, depth: int) -> int:
        """Triggers a scraping job and returns the scraping_id."""
        payload = {"url": url, "depth": depth}
        response = self.session.post(f"{API_URL}/scrape", json=payload, timeout=5)
        self.assertEqual(
            response.status_code, 200, f"Failed to trigger scraping: {response.text}"
        )
        data = response.json()
        self.assertIn("scraping_id", data)
        return int(data["scraping_id"])

    def _get_scraping_status(self, scraping_id: int) -> dict:
        """Gets the status of a scraping job."""
        response = self.session.get(
            f"{API_URL}/scrape", params={"scraping_id": scraping_id}, timeout=5
        )
        self.assertEqual(
            response.status_code, 200, f"Failed to get status: {response.text}"
        )
        return cast(dict[Any, Any], response.json())

    def _poll_scraping_completion(self, scraping_id: int) -> list:
        """Polls the API until the scraping job completes, returning the results."""
        final_status = None
        results = []

        # Wait up to 60s
        for _ in range(60):
            time.sleep(1)
            status_data = self._get_scraping_status(scraping_id)
            final_status = status_data.get("status")
            if final_status == "COMPLETED":
                results = status_data.get("data", [])
                break

        self.assertEqual(final_status, "COMPLETED", "Scraping timed out or failed.")
        return results

    def _is_image_found(self, results: list) -> bool:
        for page in results:
            for img in page.get("images", []):
                if "darth.png" in img["url"]:
                    return True
        return False

    def _check_images_persistence(self, scraping_id: int) -> bool:
        """Polls specifically for the presence of a specific image in the results."""
        for _ in range(60):
            try:
                status_data = self._get_scraping_status(scraping_id)
                results = status_data.get("data", [])
                if self._is_image_found(results):
                    return True
            except Exception as e:  # pylint: disable=broad-exception-caught
                print(f"Error fetching results during image check: {e}")

            time.sleep(1)
        return False

    def test_scraping_flow(self) -> None:
        """End-to-end test of the full scraping flow including image extraction."""
        # 1. Trigger Scraping
        scraping_id = self._trigger_scraping(
            "http://mock-website:8000/index.html", depth=2
        )
        print(f"Scraping started with ID: {scraping_id}")

        # 2. Poll for Completion
        results = self._poll_scraping_completion(scraping_id)

        # 3. Check Terms
        self.assertTrue(len(results) > 0, "No pages returned.")
        found_terms = any(page.get("terms") for page in results)
        self.assertTrue(found_terms, "No terms found in results.")

        # 4. Check Images
        found_image = self._check_images_persistence(scraping_id)
        self.assertTrue(found_image, "Image 'darth.png' not found in scraping results.")

        # 5. Check Summaries
        found_summary = self._check_summary_persistence(scraping_id)
        self.assertTrue(found_summary, "Page summary not found in results.")

    def _check_summary_persistence(self, scraping_id: int) -> bool:
        """Polls specifically for the presence of a page summary in the results."""
        for _ in range(60):
            try:
                status_data = self._get_scraping_status(scraping_id)
                results = status_data.get("data", [])
                found_summary = any(
                    page.get("summary") and "Mocked summary" in page.get("summary")
                    for page in results
                )
                if found_summary:
                    return True
            except Exception as e:  # pylint: disable=broad-exception-caught
                print(f"Error fetching results during summary check: {e}")

            time.sleep(1)
        return False

    def test_cycle_detection(self) -> None:
        print("Starting Cycle Detection Test...")
        # 1. Trigger Scraping on Cycle Page A
        # A -> B -> A (loop)
        payload = {"url": "http://mock-website:8000/cycle_a.html", "depth": 5}
        # Depth 5 would cause infinite loop without cycle detection

        response = self.session.post(f"{API_URL}/scrape", json=payload, timeout=5)
        self.assertEqual(
            response.status_code,
            200,
            f"Failed to trigger cycle scraping: {response.text}",
        )

        data = response.json()
        scraping_id = data["scraping_id"]
        print(f"Cycle Scraping started with ID: {scraping_id}")

        # 2. Poll for Completion
        final_status = "PENDING"
        results = []
        for _ in range(60):  # Wait up to 60s
            time.sleep(1)
            status_data = self._get_scraping_status(scraping_id)
            final_status = cast(str, status_data.get("status"))
            if final_status == "COMPLETED":
                results = status_data.get("data", [])
                break

        self.assertEqual(
            final_status,
            "COMPLETED",
            "Cycle scraping timed out (possible infinite loop).",
        )

        # 3. Verify Results
        # Should contain cycle_a.html and cycle_b.html (2 pages)
        # If no cycle detection, it might have many duplicates or timed out.
        print(f"Results found: {len(results)}")
        self.assertTrue(len(results) >= 2, "Should find at least A and B")
        self.assertTrue(
            len(results) <= 2, "Should NOT find more than 2 pages (duplicates)"
        )

        urls = [r["url"] for r in results]
        self.assertTrue(any("cycle_a.html" in u for u in urls), "Page A not found")
        self.assertTrue(any("cycle_b.html" in u for u in urls), "Page B not found")
        print("Cycle Detection Test passed!")

        print("Cycle Detection Test passed!")

    def test_dynamodb_job_history(self) -> None:
        print("Starting DynamoDB Job History Test...")
        url = "http://mock-website:8000/index.html"
        payload = {"url": url, "depth": 1}
        response = self.session.post(f"{API_URL}/scrape", json=payload, timeout=5)
        self.assertEqual(response.status_code, 200)

        scraping_id = response.json()["scraping_id"]
        print(f"Scraping started with ID: {scraping_id}")

        # Verify DynamoDB Item
        dynamodb = boto3.resource(
            "dynamodb",
            endpoint_url=AWS_ENDPOINT_URL,
            region_name=AWS_REGION,
            aws_access_key_id=AWS_ACCESS_KEY_ID,
            aws_secret_access_key=AWS_SECRET_ACCESS_KEY,
        )
        table = dynamodb.Table("scraping_jobs")

        item = None
        for _ in range(10):
            try:
                resp = table.get_item(Key={"job_id": str(scraping_id)})
                if "Item" in resp:
                    item = resp["Item"]
                    break
            except Exception:  # pylint: disable=broad-exception-caught
                pass
            time.sleep(1)

        self.assertIsNotNone(item, "Job not found in DynamoDB")
        assert item is not None
        self.assertEqual(item["url"], url)
        self.assertEqual(item["status"], "PENDING")
        self.assertEqual(item["status"], "PENDING")
        print("DynamoDB Job History Test passed!")


if __name__ == "__main__":
    unittest.main()
