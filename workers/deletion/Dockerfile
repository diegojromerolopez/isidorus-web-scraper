FROM python:3.14-slim

WORKDIR /app

# Install dependencies
COPY workers/deletion/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the shared package
COPY shared/ /app/shared/
# Copy the api package (for models)
COPY api/ /app/api/

# Copy the deletion package
COPY workers/deletion/ /app/workers/deletion/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.deletion.main"]
