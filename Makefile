# Run your Go server with air (hot reload)
dev:
	air

# Build the binary
build:
	go build -o bin/server ./cmd/server

# Run migrations
migrate-up:
	goose -dir ./db/migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir ./db/migrations postgres "$$DATABASE_URL" down

# Generate SQLC code
sqlc-generate:
	sqlc generate

# Format Go code
fmt:
	go fmt ./...

# Run tests (later)
test:
	go test ./...

