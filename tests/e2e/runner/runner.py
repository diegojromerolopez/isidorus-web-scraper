import os
import time
import unittest
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


class TestScrapingE2E(unittest.TestCase):
    def setUp(self) -> None:
        """Clean up Database and Redis before each test."""
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
        required_queues = ["scraper-queue", "image-queue", "writer-queue"]
        for _ in range(30):
            try:
                resp = sqs.list_queues()
                urls = resp.get("QueueUrls", [])
                found = 0
                for q_url in urls:
                    for req in required_queues:
                        if req in q_url:
                            found += 1
                if found >= len(required_queues):
                    print("SQS queues are ready!")
                    return
            except Exception as e:
                print(f"Error checking SQS: {e}")
            time.sleep(1)
        self.fail("SQS queues failed to become ready.")

    def wait_for_db(self) -> None:
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
            # Truncate tables. Order matters due to FKs.
            # scraping -> scraped_pages -> page_terms/links/images
            # Actually FK is page->scraping.
            # scraped_pages has FK to scrapings.
            # page_terms has FK to scraped_pages.
            cur.execute(
                "TRUNCATE TABLE page_images, page_links, page_terms, "
                "scraped_pages, scrapings CASCADE;"
            )
            conn.commit()
            cur.close()
            conn.close()
            print("Database cleaned.")
        except Exception as e:
            print(f"Error cleaning database: {e}")
            # Don't fail setup, maybe it's empty or connection failed
            # (will fail test anyway)

    def cleanup_redis(self) -> None:
        try:
            r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
            r.flushall()
            print("Redis cleaned.")
        except Exception as e:
            print(f"Error cleaning Redis: {e}")

    def wait_for_api(self) -> None:
        for _ in range(30):
            try:
                requests.get(f"{API_URL}/docs", timeout=5)
                return
            except requests.RequestException:
                time.sleep(1)
        self.fail("API failed to become ready.")

    def test_scraping_flow(self) -> None:
        # 1. Trigger Scraping
        payload = {"url": "http://mock-website:8000/index.html", "depth": 2}
        response = requests.post(f"{API_URL}/scrape", json=payload, timeout=5)
        self.assertEqual(
            response.status_code, 200, f"Failed to trigger scraping: {response.text}"
        )

        data = response.json()
        self.assertIn("scraping_id", data)
        scraping_id = data["scraping_id"]
        print(f"Scraping started with ID: {scraping_id}")

        # 2. Check PENDING status
        # Give a small moment for async DB write if needed,
        # but API should return ID immediately.
        # Check status immediately
        status_resp = requests.get(
            f"{API_URL}/scrape?scraping_id={scraping_id}", timeout=5
        )
        self.assertEqual(status_resp.status_code, 200)
        status = status_resp.json().get("status")
        # It could be PENDING or COMPLETED (fast mock), usually PENDING.
        self.assertIn(status, ["PENDING", "COMPLETED"])

        # 3. Poll for Completion
        final_status = status
        results = []
        for _ in range(30):
            if final_status == "COMPLETED":
                break
            time.sleep(1)
            status_resp = requests.get(
                f"{API_URL}/scrape?scraping_id={scraping_id}", timeout=5
            )
            if status_resp.status_code == 200:
                body = status_resp.json()
                final_status = body.get("status")
                if final_status == "COMPLETED":
                    results = body.get("data", [])

        self.assertEqual(final_status, "COMPLETED", "Scraping timed out or failed.")

        # 4. Check Terms
        self.assertTrue(len(results) > 0, "No pages returned.")
        found_terms = False
        for page in results:
            if page.get("terms"):
                found_terms = True
                break
        self.assertTrue(found_terms, "No terms found in results.")

        # 5. Check Images (with retry for consistency)
        found_image = False
        for _ in range(60):
            if found_image:
                break

            # Fetch fresh results
            try:
                status_resp = requests.get(
                    f"{API_URL}/scrape?scraping_id={scraping_id}", timeout=5
                )
                if status_resp.status_code == 200:
                    body = status_resp.json()
                    results = body.get("data", [])
                    for page in results:
                        images = page.get("images", [])
                        for img in images:
                            if "darth.png" in img["url"]:
                                found_image = True
                                break
                        if found_image:
                            break
            except Exception as e:
                print(f"Error fetching results during image check: {e}")

            if not found_image:
                print("Image not found yet, retrying...")
                time.sleep(1)

        self.assertTrue(found_image, "Image 'darth.png' not found in scraping results.")

        print("Test passed!")

    def test_cycle_detection(self) -> None:
        print("Starting Cycle Detection Test...")
        # 1. Trigger Scraping on Cycle Page A
        # A -> B -> A (loop)
        payload = {"url": "http://mock-website:8000/cycle_a.html", "depth": 5}
        # Depth 5 would cause infinite loop without cycle detection

        response = requests.post(f"{API_URL}/scrape", json=payload, timeout=5)
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
            status_resp = requests.get(
                f"{API_URL}/scrape?scraping_id={scraping_id}", timeout=5
            )
            if status_resp.status_code == 200:
                body = status_resp.json()
                final_status = body.get("status")
                if final_status == "COMPLETED":
                    results = body.get("data", [])
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


if __name__ == "__main__":
    unittest.main()
