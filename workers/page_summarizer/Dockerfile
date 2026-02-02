FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY workers/page_summarizer/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the shared package
COPY shared/ /app/shared/

# Copy the page_summarizer package
COPY workers/page_summarizer/ /app/workers/page_summarizer/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.page_summarizer.main"]
