FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY workers/image_extractor/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the shared package
COPY shared/ /app/shared/

# Copy the image_extractor package
COPY workers/image_extractor/ /app/workers/image_extractor/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.image_extractor.main"]
