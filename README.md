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
- **SQLite** via [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) — pure Go, no CGO
- **Telegram Bot API** via [`go-telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api)
- **Structured logging** with `log/slog` (stdlib)
- **Docker** + **docker compose** for one-command runs

The data layer hides behind repository interfaces, so swapping SQLite for PostgreSQL later only requires a new repository implementation and a different driver in `internal/database/database.go`.

---

## Project structure

```
tomorrow-tracker/
├── cmd/app/                # main entrypoint (DI wiring + graceful shutdown)
├── internal/
│   ├── bot/                # Telegram client, router, keyboard
│   ├── handlers/           # one file per feature: start, session, hours, schedule
│   ├── middleware/         # recovery + structured per-update logging
│   ├── services/           # business logic (sessions, weekly progress)
│   ├── repositories/       # SQL data access behind interfaces
│   ├── models/             # plain data structs
│   ├── database/           # connection + schema migrations
│   ├── config/             # env-based config loader
│   └── utils/              # tz-safe time math, formatting
├── pkg/logger/             # slog factory
├── storage/                # SQLite database lives here
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
- `/setcommands` — paste the list below to enable autocomplete

```
start - Запустить бота и показать меню
```

---

## Environment variables

Copy `.env.example` to `.env` and fill in the token.

| Variable | Default | Description |
|----------|---------|-------------|
| `TELEGRAM_BOT_TOKEN` | _(required)_ | Token from @BotFather |
| `DATABASE_PATH` | `./storage/tomorrow.db` | SQLite file location |
| `TIMEZONE` | `Asia/Almaty` | All daily/weekly boundaries use this zone |
| `WEEKLY_TARGET_HOURS` | `30` | Weekly study goal |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

---

## Local run

```bash
cp .env.example .env
# edit .env and set TELEGRAM_BOT_TOKEN

go mod download
go run ./cmd/app
```

The bot will register itself with Telegram, create `storage/tomorrow.db` on first launch, and start polling for messages.

To build a static binary instead:

```bash
CGO_ENABLED=0 go build -o tomorrow-tracker ./cmd/app
./tomorrow-tracker
```

---

## Docker run

```bash
cp .env.example .env
# edit .env and set TELEGRAM_BOT_TOKEN

docker compose up --build
```

The `storage/` folder is mounted into the container so the SQLite database survives `docker compose down`. To stop:

```bash
docker compose down
```

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
