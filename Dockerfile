FROM golang:1.24.5-alpine AS builder

RUN apk add --no-cache git curl

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .

RUN go build -o /app/tmp/main ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /app/tmp/main ./main
COPY --from=builder /go/bin/goose ./goose
COPY --from=builder /app/db/migrations ./db/migrations

CMD ./goose -dir ./db/migrations postgres "$DATABASE_URL" up && ./main
