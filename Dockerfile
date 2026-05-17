# --- frontend build stage ---
# Builds the Telegram Mini App SPA into frontend/dist, which the Go stage then
# embeds into the binary. Kept first so its layers cache independently of Go.
FROM node:22-alpine AS frontend

WORKDIR /app/frontend

# Cache npm install on the manifest alone.
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install

COPY frontend/ ./
RUN npm run build

# --- Go build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Replace the committed placeholder with the real SPA build before embedding.
COPY --from=frontend /app/frontend/dist ./frontend/dist

# pgx is pure Go, so CGO is not required and the binary is fully static.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tomorrow-tracker ./cmd/app

# --- runtime stage ---
FROM alpine:3.20

# tzdata so Asia/Almaty resolves at runtime;
# ca-certificates so HTTPS to api.telegram.org and TLS to PostgreSQL work.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=build /out/tomorrow-tracker /app/tomorrow-tracker
USER app

ENV TIMEZONE=Asia/Almaty

# Containerized deploys run in production mode: the server fails fast at
# startup if the embedded SPA is the placeholder rather than a real Vite
# build. The frontend build stage above always produces a real build, so a
# correctly built image starts cleanly; a broken build is caught loudly.
ENV APP_ENV=production

# Health endpoint port — Railway overrides PORT automatically.
EXPOSE 8080

ENTRYPOINT ["/app/tomorrow-tracker"]
