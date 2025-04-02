FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o main main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY wait-for.sh .

RUN chmod +x /app/start.sh
RUN chmod +x /app/wait-for.sh

COPY db/migration ./migration

EXPOSE 8080

CMD ["/app/main"]
ENTRYPOINT [ "/app/start.sh" ]