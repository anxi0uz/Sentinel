FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/scoring-engine ./services/scoring-engine/cmd

FROM docker.io/library/alpine:3.21
RUN addgroup -g 1000 app && adduser -u 1000 -G app -D -H -s /sbin/nologin app
COPY --from=builder /bin/scoring-engine /app/scoring-engine
COPY --chown=app:app services/scoring-engine/configs/ /app/configs/
COPY --chown=app:app services/scoring-engine/migrations/ /app/migrations/
USER app
WORKDIR /app
CMD ["/app/scoring-engine"]
