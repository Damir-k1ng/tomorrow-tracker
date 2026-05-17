# Tomorrow Tracker

Telegram bot that helps **Tomorrow School Astana** students track their mandatory weekly study hours during the **Pool** phase.

Students must study at least **30 hours per week**. Tomorrow Tracker lets them:

- ▶️ start / ⏹ stop study sessions with a single tap
- ⏱ see today's hours, weekly hours, and remaining hours toward the 30h goal
- 📅 view the Pool weekly schedule inside Telegram

All bot messages are in **Russian**. All time calculations use **Asia/Almaty**.

---

## Features

| Button | What it does |
|--------|--------------|
| ▶️ Начать сессию | Opens a new study session (rejects if one is already active) |
| ⏹ Завершить сессию | Closes the active session and shows session length + day/week/remaining |
| ⏱ Мои часы | Snapshot of today, week, remaining, and active session status |
| 📅 Расписание | Shows the Pool weekly schedule |
| 🏆 Топ-10 | Weekly Top-10 leaderboard + your own position if you're below the top |

### Anti-cheat

The leaderboard caps each session at **12 hours**. Sessions with negative or invalid timestamps are ignored. Aggregation runs in a single SQL query — no per-user fan-out, no full-table loads.

---

## Tech stack

- **Go** (latest stable, tested on 1.23+)
- **PostgreSQL** via [`jackc/pgx/v5`](https://github.com/jackc/pgx) + `pgxpool` — pure Go, no CGO, no ORM
- **Telegram Bot API** via [`go-telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api) — long-polling mode
- **Structured logging** with `log/slog` (stdlib)
- **Docker** + **docker compose** for one-command runs

The data layer hides behind repository interfaces (`UserRepository`, `SessionRepository`), so business logic never touches SQL directly.

---

## Project structure

```
tomorrow-tracker/
├── cmd/app/                # main entrypoint (startup flow + graceful shutdown)
├── internal/
│   ├── bot/                # Telegram client, router, keyboard
│   ├── handlers/           # bot handlers: start, session, hours, schedule, leaderboard
│   ├── middleware/         # bot recovery + structured per-update logging
│   ├── services/           # business logic (sessions, progress, leaderboard, streaks, admin)
│   ├── repositories/       # pgx data access behind interfaces
│   ├── models/             # plain data structs
│   ├── database/           # pgxpool connection + schema migrations
│   ├── api/                # HTTP server: health + /api/v1 foundation
│   ├── config/             # env-based config loader
│   └── utils/              # tz-safe time math, formatting
├── pkg/logger/             # slog factory
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## Creating the Telegram bot

1. Open [@BotFather](https://t.me/BotFather) in Telegram.
2. Send `/newbot`.
3. Pick a display name (e.g. `Tomorrow Tracker`).
4. Pick a unique username ending in `bot` (e.g. `tomorrow_tracker_bot`).
5. Copy the token you receive — that's your `TELEGRAM_BOT_TOKEN`.

Optional polishing inside BotFather:

- `/setdescription` — short description shown in the bot profile

The slash-command list (`/study`, `/stop`, `/hours`, `/schedule`, `/top`,
`/help`, `/start`) is **registered automatically** by the bot on startup via
`setMyCommands` — no manual BotFather `/setcommands` step is needed.

---

## Environment variables

Copy `.env.example` to `.env` and fill in the token + database URL.

| Variable | Default | Description |
|----------|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | _(required)_ | Token from @BotFather |
| `DATABASE_URL` | _(required)_ | PostgreSQL connection string (Railway injects it automatically) |
| `TIMEZONE` | `Asia/Almaty` | All daily/weekly/streak boundaries use this zone |
| `WEEKLY_TARGET_HOURS` | `30` | Weekly study goal |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `PORT` | `8080` | HTTP port for the health + API endpoints (Railway sets this) |
| `ADMIN_TELEGRAM_ID` | `0` | Telegram ID auto-promoted to the `admin` role; `0` (default) disables auto-promotion — set it explicitly to enable |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated CORS allow-list for the future Mini App |

---

## Startup flow

On boot the app: **(1)** loads env → **(2)** connects PostgreSQL → **(3)** pings the database (bounded 10s timeout) → **(4)** applies schema migrations → **(5)** starts the Telegram polling loop. A failure at steps 2–4 logs a clear `FATAL:` line and exits non-zero, so a misconfigured database never produces a half-running bot.

---

## Local run

You need a PostgreSQL instance. The fastest way is the bundled compose db:

```bash
cp .env.example .env
# edit .env: set TELEGRAM_BOT_TOKEN, and point DATABASE_URL at your Postgres

go mod download
go run ./cmd/app
```

The schema is created automatically on first launch (`CREATE TABLE IF NOT EXISTS`), then the bot starts polling.

To build a static binary instead:

```bash
CGO_ENABLED=0 go build -o tomorrow-tracker ./cmd/app
./tomorrow-tracker
```

---

## Docker run

`docker compose` brings up PostgreSQL **and** the bot together:

```bash
cp .env.example .env
# edit .env and set TELEGRAM_BOT_TOKEN (DATABASE_URL is overridden by compose)

docker compose up --build
```

The Postgres data lives in the `pgdata` named volume, so it survives `docker compose down`. To stop:

```bash
docker compose down
```

---

## Railway deployment

1. Create a new Railway project and add a **PostgreSQL** plugin — Railway exposes `DATABASE_URL` automatically.
2. Deploy this repo as a service (Railway builds the `Dockerfile`).
3. Set `TELEGRAM_BOT_TOKEN` in the service variables.
4. `PORT` is injected by Railway; the bot serves a tiny HTTP health endpoint (`/` and `/health`) on it so the platform sees the service as healthy while the bot polls Telegram in the background.
5. Graceful shutdown: on `SIGTERM` the bot stops polling, the API server drains, and the `pgxpool` connections are closed.

---

## API foundation (Phase 1)

Alongside the Telegram bot, the app serves an HTTP API foundation for a future Mini App and Admin Panel. The bot itself is unaffected — this is purely additive.

- **Roles** — `users.role` (`user` / `admin` / `moderator`). The account in `ADMIN_TELEGRAM_ID` is auto-promoted to `admin` on its next bot/API interaction. Role is checked via `user.Role`, never a hardcoded ID.
- **Mini App auth** — `POST /api/v1/auth/verify` with header `Authorization: tma <initData>` validates the Telegram WebApp `initData` (HMAC-SHA256 signature + `auth_date` freshness) and returns the user with their role.
- **Middleware** — CORS, per-IP token-bucket rate limiting (no Redis), `RequireTelegramAuth`, `RequireAdmin` (403 for non-admins), request timeouts, and panic recovery.
- **Responses** — every `/api/v1` reply uses `{"success":true,"data":…}` or `{"success":false,"error":"…"}`.

### Admin API (Phase 2)

All admin endpoints require an authenticated admin (`Authorization: tma <initData>`), are rate-limited, and run under a 30s request timeout.

| Method & path | Purpose |
|---------------|---------|
| `GET /api/v1/admin/stats` | Operational counters (users, sessions, hours, streaks, new users 7d) |
| `GET /api/v1/admin/users` | Paginated users list — `?page&limit&search&sort` (deterministic ordering) |
| `GET /api/v1/admin/users/{id}` | Profile, streak, total hours, active session, last 20 sessions |
| `PATCH /api/v1/admin/sessions/{id}` | Correct a **finished** session — body `{duration_minutes?, is_valid?, anti_cheat_flags?, reason}` |
| `GET /api/v1/admin/audit-logs` | Paginated audit log — `?page&limit&action&admin_id&sort` |
| `GET /api/v1/admin/export/users` | CSV export — `?from&to` required (max 366 days) |
| `GET /api/v1/admin/export/sessions` | CSV export — `?from&to` required (max 366 days) |

- **Session correction** accepts `duration_minutes` (0–720), `is_valid` (anti-cheat), and `anti_cheat_flags` (a whitelisted array: `manual_review`, `suspicious_duration`, `rapid_restarts`, `overlap_detected`, `admin_invalidated`). At least one of those plus a mandatory `reason` is required; unknown body fields and unknown flags are rejected with `400`.
- The correction, its `admin_actions` audit row, and any streak recomputation commit in **one transaction** — all-or-nothing.
- **Streak consistency:** when a correction changes whether a session qualifies for the streak (`is_valid` flips, or `duration_minutes` crosses the 30-minute minimum), the affected user's `current_streak` / `best_streak` / `last_study_at` are **deterministically recomputed** from their sessions inside the same transaction. The leaderboard stays query-based and is never recomputed.
- **Audit log** (`admin_actions`) is immutable and append-only — enforced at the database level by a trigger that rejects every `UPDATE` / `DELETE` / `TRUNCATE`. Every correction and export writes a record (`PATCH_SESSION`, `EXPORT_USERS`, `EXPORT_SESSIONS`) with `before_data`/`after_data` JSONB snapshots; `entity_type` is constrained to a fixed taxonomy.
- The correction endpoint carries an extra **per-admin** rate limit (10/min) on top of the shared admin limiter, keyed by admin identity rather than IP.

---

## Business rules

- Only **one active session per user** — pressing ▶️ twice gives a friendly warning.
- Pressing ⏹ with no active session gives a friendly warning, never an error.
- Today / week / remaining are **always** computed in `Asia/Almaty`. The week starts **Monday 00:00**.
- The active session contributes its in-progress minutes to *today* and *week* totals up to "now", so progress numbers are live.
- Bot restarts are safe — sessions are persisted; an active session before restart is still active afterwards.

---

## Logging

Every update produces one structured JSON log line:

```json
{"time":"...","level":"INFO","msg":"update handled","update_id":42,"user_id":12345,"text":"⏱ Мои часы","took":31000000}
```

Errors include the error string. Panics in handlers are caught by `middleware.Recover` and logged with a stack trace; the bot keeps running.

---

## Author

**Damir Kabdulla** — [@King_traff](https://t.me/King_traff)

Удачи в бассейне 💪
