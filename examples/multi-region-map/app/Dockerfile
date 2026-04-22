FROM python:3.12-alpine

WORKDIR /app

# Install minimal dependencies
RUN apk add --no-cache gcc musl-dev

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENV REGION="unknown"
ENV PORT="8080"
ENV LISTEN_PORT="8080"

EXPOSE 8080

CMD ["python", "app.py"]
