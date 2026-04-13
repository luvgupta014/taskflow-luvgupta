FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/api

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server .
COPY --from=builder /build/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
