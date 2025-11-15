FROM golang:1.25.3-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

# swagger
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

RUN swag init -g ./cmd/main.go --output ./docs

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/server \
    ./cmd

FROM alpine:3.19

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/docs ./docs

RUN chown -R appuser:appuser /app

USER appuser

ARG PORT=8080
ENV PORT=${PORT}
EXPOSE ${PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

CMD ["/app/server"]