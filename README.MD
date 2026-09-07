# RateWatch
An educational Telegram bot in Go that tracks cryptocurrency rates on Binance
(BTC, ETH, TON) and sends a notification when the price crosses a threshold set
by the user. All data is stored in memory, without a database.
## How it works
- **Telegram**: the bot polls telegram in a separate goroutine.
- **Binance**: for each pair (`btcusdt`, `ethusdt`, `tonusdt`) its own
  `@trade` websocket stream is opened, each in its own goroutine.
- **Matching**: a single merger goroutine (`select` over all channels
  at once) holds the active rules and checks them on every new trade —
  without mutexes, because only this goroutine reads and writes the state.
- Shutdown — via `Ctrl+C`, through `context`
## Running
You need to put the telegram token in .env (see .env.example for more details)
```
go run cmd/ratewatch/main.go
```
## Bot commands
- `/help` — a short reference.
- A rule in the format `<currency> <operator> <amount>`, for example:
  ```
  btc > 50000
  eth < 3000
  ```
## Limitations / what's not done
- Rules are not persisted between restarts (in memory only).
- There is no command to view/delete already added rules.
- The currency list is hardcoded (`btc`, `eth`, `ton`).
