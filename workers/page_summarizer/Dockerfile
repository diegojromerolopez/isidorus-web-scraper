FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY workers/page_summarizer/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the shared package
COPY shared/ /app/shared/

# Copy the workers package
COPY workers/ /app/workers/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.page_summarizer.main"]
