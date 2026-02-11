import os
import sys
import time

import requests

# Configuration
API_URL = os.environ.get("API_URL", "http://api:8000")
API_KEY = os.environ.get("API_KEY", "test-api-key-123")
SCRAPE_URL = os.environ.get("SCRAPE_URL", "https://news.ycombinator.com/news")
SCRAPE_DEPTH = int(os.environ.get("SCRAPE_DEPTH", 1))
POLL_INTERVAL = 2  # seconds
TIMEOUT = 300  # seconds


def wait_for_api() -> None:
    """Waits for the API to be available."""
    start_time = time.time()
    print(f"Waiting for API at {API_URL}...")
    while time.time() - start_time < TIMEOUT:
        try:
            response = requests.get(f"{API_URL}/health", timeout=5)
            if response.status_code == 200:
                print("API is ready.")
                return
        except requests.RequestException:
            pass
        time.sleep(POLL_INTERVAL)
    print("Timeout waiting for API.")
    sys.exit(1)


def create_scraping() -> int:
    """Submits a scraping job."""
    headers = {"X-API-Key": API_KEY}
    payload = {
        "url": SCRAPE_URL,
        "depth": SCRAPE_DEPTH,
        "term": "python",  # Default term to ensure we find something
    }

    print(f"Submitting scrape job for {SCRAPE_URL}...")
    try:
        response = requests.post(
            f"{API_URL}/scrape", json=payload, headers=headers, timeout=30
        )
        response.raise_for_status()
        job_data = response.json()
        scraping_id = job_data.get("scraping_id") or job_data.get(
            "id"
        )  # Handle potential response variations
        print(f"Job submitted successfully. Job ID: {scraping_id}")
        return int(scraping_id)
    except requests.RequestException as e:
        print(f"Failed to submit job: {e}")
        if hasattr(e, "response") and e.response is not None:
            print(f"Response: {e.response.text}")
        sys.exit(1)


def monitor_scraping(scraping_id: int) -> None:
    """Polls the job status until completion."""
    headers = {"X-API-Key": API_KEY}
    start_time = time.time()

    print(f"Monitoring job {scraping_id}...")
    while time.time() - start_time < TIMEOUT:
        try:
            response = requests.get(
                f"{API_URL}/scraping/{scraping_id}", headers=headers, timeout=30
            )
            response.raise_for_status()
            data = response.json()

            # The API returns {"scraping": {...}}
            job = data.get("scraping", {})
            if not job:
                print("Job not found in response.")
                sys.exit(1)

            status = job.get("status")
            print(f"Job Status: {status}")

            if status == "COMPLETED":
                print("Job completed successfully!")
                return
            if status in ("FAILED", "ERROR"):
                print("Job failed.")
                sys.exit(1)

        except requests.RequestException as e:
            print(f"Error checking job status: {e}")

        time.sleep(POLL_INTERVAL)

    print("Timeout waiting for job completion.")
    sys.exit(1)


def main():
    wait_for_api()
    # Give a small buffer for Auth Admin to sync keys if this is a fresh start
    time.sleep(5)
    scraping_id = create_scraping()
    monitor_scraping(scraping_id)


if __name__ == "__main__":
    main()
