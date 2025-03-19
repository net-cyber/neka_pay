# Build stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install necessary packages
RUN apk --no-cache add ca-certificates postgresql-client curl

# Install migrate tool
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/bin/migrate

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/app.env .
COPY --from=builder /app/db/migration ./db/migration

# Wait for database to be ready
COPY ./wait-for.sh .
RUN chmod +x wait-for.sh

# Create a non-root user and switch to it
RUN adduser -D -g '' appuser
USER appuser

# Expose the application port
EXPOSE 8080

# Command to run the application
CMD ["./wait-for.sh", "postgres-go:5432", "--", "./main"]