FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o game-over-man .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/game-over-man .
RUN mkdir -p /etc/game-over-man /var/lib/game-over-man
ENV CONFIG_FILE=/config/config.json \
    STATE_FILE=/data/state.json
VOLUME ["/data"]
ENTRYPOINT ["/app/game-over-man"]
