# Multi-stage build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o worklog-tracker .

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/worklog-tracker .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./worklog-tracker"]
