FROM python:3.14-slim

WORKDIR /app

RUN pip install requests redis psycopg2-binary boto3

COPY runner.py .

CMD ["python", "runner.py"]
