# Build stage
FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tarpit .

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/tarpit .
COPY --from=builder /app/config ./config
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/data ./data
EXPOSE 80
ENTRYPOINT ["./tarpit"]
