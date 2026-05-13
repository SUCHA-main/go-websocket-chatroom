# Go WebSocket Chatroom Demo

A small real-time chatroom demo built with Go, Gorilla WebSocket, and plain HTML/CSS/JavaScript. It includes local user registration, session-based login, message history, emoji/stickers, and optional local Ollama AI assistant commands.

## Features

- Register, login, and logout
- Protected chatroom page and protected WebSocket endpoint
- Real-time multi-user chat over WebSocket
- Online user count
- System join/leave notifications
- Last 20 message history replay on reconnect
- Text, emoji, and local SVG sticker messages
- Distinct styles for self, other users, system messages, stickers, history, and AI messages
- Message length limit and character counter
- Optional local Ollama AI assistant with `@AI`, `/ai`, and `/summary`

## Tech Stack

- Go
- net/http
- github.com/gorilla/websocket
- golang.org/x/crypto/bcrypt
- Plain HTML/CSS/JavaScript
- Optional Ollama local model API

## Directory Structure

```text
.
├── go.mod
├── go.sum
├── main.go
├── users.example.json
├── public/
│   ├── app.js
│   ├── index.html
│   ├── login.html
│   ├── register.html
│   ├── style.css
│   └── stickers/
│       ├── sticker-1.svg
│       ├── sticker-2.svg
│       ├── sticker-3.svg
│       ├── sticker-4.svg
│       ├── sticker-5.svg
│       └── sticker-6.svg
└── README.md
```

## Getting Started

Install Go, then run:

```powershell
go mod tidy
go run main.go
```

Open:

```text
http://localhost:8088
```

The app creates `users.json` automatically on first run. That file stores local demo users and is intentionally ignored by Git.

## Optional Ollama AI Assistant

Start Ollama separately, then set the API URL and model if needed:

```powershell
$env:OLLAMA_URL = "http://localhost:11434"
$env:OLLAMA_MODEL = "qwen2.5:3b"
go run main.go
```

In the chatroom:

```text
@AI Explain WebSocket in one sentence
/ai Summarize what Gorilla WebSocket does
/summary
```

If Ollama is not running, the chatroom remains available and returns a friendly AI unavailable message.

## Testing

Run:

```powershell
gofmt -w main.go
go mod tidy
go test ./...
```

Manual checks:

- Visit `/` while logged out and confirm it redirects to `/login`
- Register and login with two accounts in separate browser windows
- Send text, emoji, and sticker messages
- Refresh the page and confirm recent history appears
- Try `@AI` and `/summary` with Ollama running
- Stop Ollama and confirm AI commands show a friendly error

## Screenshots

Screenshots are not included yet. Suggested placeholders:

- Login page
- Chatroom with two users online
- Emoji and sticker panel
- AI assistant reply

## Roadmap

- Add reconnect/backoff UI for WebSocket disconnects
- Add message timestamps by date group
- Add Dockerfile and compose example
- Add automated browser tests
- Add optional persistent database storage

## License

MIT
