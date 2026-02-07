# Using Python 3.11 because langchain dependencies (numpy/pandas) do not support 3.14 yet
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY workers/image_explainer/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the shared package
COPY shared/ /app/shared/

# Copy the image_explainer package
COPY workers/image_explainer/ /app/workers/image_explainer/

ENV PYTHONPATH=/app

CMD ["python", "-m", "workers.image_explainer.main"]
