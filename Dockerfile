FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY app.env .
COPY start.sh .
COPY wait-for.sh .

RUN chmod +x /app/start.sh
RUN chmod +x /app/wait-for.sh

COPY db/migration ./migration

EXPOSE 8080

CMD ["/app/main"]
ENTRYPOINT [ "/app/start.sh" ]