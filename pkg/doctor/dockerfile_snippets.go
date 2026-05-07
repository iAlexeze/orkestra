package doctor

import (
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// DockerfileTemplate returns a starter Dockerfile snippet for the detected language.
// If the language is unknown, it returns an empty string.
func DockerfileTemplate(l orktypes.Language) string {
	switch strings.ToLower(string(l)) {

	case "go":
		return `# --- Starter Dockerfile for Go ---
FROM golang:1.22 AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN go build -o app

# Minimal runtime image
FROM gcr.io/distroless/base
COPY --from=builder /app/app /app
CMD ["/app"]`

	case "node.js":
		return `# --- Starter Dockerfile for Node.js ---
FROM node:20-alpine
WORKDIR /app

COPY package*.json ./
RUN npm install --production

COPY . .
CMD ["node", "server.js"]`

	case "python":
		return `# --- Starter Dockerfile for Python ---
FROM python:3.12-slim
WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
CMD ["python", "main.py"]`

	case "java":
		return `# --- Starter Dockerfile for Java (Maven) ---
FROM maven:3.9 AS builder
WORKDIR /app

COPY pom.xml .
COPY src ./src
RUN mvn package -DskipTests

FROM eclipse-temurin:21-jre
COPY --from=builder /app/target/*.jar /app/app.jar
CMD ["java", "-jar", "/app/app.jar"]`

	case "ruby":
		return `# --- Starter Dockerfile for Ruby ---
FROM ruby:3.3
WORKDIR /app

COPY Gemfile Gemfile.lock ./
RUN bundle install

COPY . .
CMD ["ruby", "main.rb"]`

	case "rust":
		return `# --- Starter Dockerfile for Rust ---
FROM rust:1.77 AS builder
WORKDIR /app

COPY . .
RUN cargo build --release

FROM gcr.io/distroless/base
COPY --from=builder /app/target/release/app /app
CMD ["/app"]`

	default:
		return ""
	}
}
