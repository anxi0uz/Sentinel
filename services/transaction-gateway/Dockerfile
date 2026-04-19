FROM docker.io/library/golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/transaction-gateway ./services/transaction-gateway/cmd/main.go

FROM docker.io/library/alpine:3.20

RUN addgroup -g 1000 app && adduser -u 1000 -G app -D -H -s /sbin/nologin app

COPY --from=builder /bin/transaction-gateway /app/transaction-gateway
COPY --chown=app:app services/transaction-gateway/configs/ /app/configs/

USER app

WORKDIR /app

EXPOSE 8080

CMD ["/app/transaction-gateway"]
