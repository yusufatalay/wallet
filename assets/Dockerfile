# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

# Build the specific service binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./assets

# Final Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 50053
CMD ["./server"]
