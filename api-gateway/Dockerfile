# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./api-gateway

# Final Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway .
COPY --from=builder /app/api-gateway/static ./static
COPY --from=builder /app/proto ./proto

EXPOSE 8080
CMD ["./gateway"]
