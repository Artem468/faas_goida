FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN chmod +x ./bin/goida_lang

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Копируем всё из билдера
COPY --from=builder /app/main .
COPY --from=builder /app/bin ./bin
COPY --from=builder /app/calculator.goida .

# Даем права (на всякий случай)
RUN chmod +x ./bin/goida_lang

CMD ["sh", "-c", "./main"]
