FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-websocket-chatroom .

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/go-websocket-chatroom /app/go-websocket-chatroom
COPY public /app/public
COPY users.example.json /app/users.example.json

EXPOSE 8088

CMD ["/app/go-websocket-chatroom"]
