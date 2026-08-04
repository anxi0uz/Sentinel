FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/event-generator ./services/event-generator/cmd

FROM docker.io/library/alpine:3.21
RUN addgroup -g 1000 app && adduser -u 1000 -G app -D -H -s /sbin/nologin app
COPY --from=builder /bin/event-generator /app/event-generator
USER app
WORKDIR /app
CMD ["/app/event-generator"]
