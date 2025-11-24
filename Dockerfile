# Step 1: Modules caching
FROM golang:1.24-alpine3.20 as modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

# Step 2: Builder
FROM golang:1.24-alpine3.20 as builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -tags migrate -o /bin/app ./cmd/app

# Step 3: Final
FROM alpine:3.20

# Install migration CLI
RUN apk add --no-cache curl ca-certificates bash \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/local/bin/migrate \
    && chmod +x /usr/local/bin/migrate

# Copy everything
COPY --from=builder /app/config /config
COPY --from=builder /bin/app /app/bin/app
COPY migrations/ /app/migrations/
COPY scripts/run_migration.sh /app/scripts/run_migration.sh

RUN chmod +x /app/scripts/run_migration.sh
RUN mkdir -p /app/uploads
RUN mkdir -p /app/logger

CMD ["/app/bin/app"]

