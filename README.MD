# RateWatch
An educational Telegram bot in Go that tracks cryptocurrency rates on Binance
(BTC, ETH, TON) and sends a notification when the price crosses a threshold set
by the user. Rules are stored in PostgreSQL.
## How it works
- **Telegram**: the bot long-polls Telegram in a separate goroutine.
- **Binance**: one `@trade` websocket stream per pair, all writing to one channel.
- **Matching**: rules are loaded from the database on start and kept in memory.
  Prices are checked against memory only; the database is hit only when a rule
  is added or fired.
- Shutdown — via `Ctrl+C` or `SIGTERM`, through `context`.
## Running
Copy `.env.example` to `.env` and set `TELEGRAM_TOKEN`.
```
docker compose up --build
```
Or locally:
```
docker compose up -d postgres
go run ./cmd/ratewatch
```
## Bot commands
- `/start` — greeting.
- `/help` — a short reference.
- A rule in the format `<currency> <operator> <amount>`, for example:
  ```
  btc > 50000
  eth < 3000
  ```
## Limitations / what's not done
- There is no command to view/delete already added rules.
