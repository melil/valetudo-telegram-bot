# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/tgbot ./cmd/bot

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/tgbot /app/tgbot

ENV TZ=Europe/Moscow

ENTRYPOINT ["/app/tgbot"]
