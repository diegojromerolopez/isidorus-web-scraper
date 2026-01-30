FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY workers/image_extractor/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the workers package
COPY workers/ /app/workers/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.image_extractor.main"]
