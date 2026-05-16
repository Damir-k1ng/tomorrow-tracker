# --- build stage ---
FROM golang:1.23-alpine AS build

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Pure-Go SQLite driver, so CGO is not required and the binary is fully static.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tomorrow-tracker ./cmd/app

# --- runtime stage ---
FROM alpine:3.20

# tzdata so Asia/Almaty resolves at runtime;
# ca-certificates so HTTPS to api.telegram.org works.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/tomorrow-tracker /app/tomorrow-tracker

# Storage volume holds the SQLite database between restarts.
RUN mkdir -p /app/storage && chown -R app:app /app
USER app

ENV DATABASE_PATH=/app/storage/tomorrow.db \
    TIMEZONE=Asia/Almaty

ENTRYPOINT ["/app/tomorrow-tracker"]
