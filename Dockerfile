FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source
COPY go.mod go.sum ./
RUN go mod download && go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

# Generate Swagger docs from annotations, then build the binary
RUN swag init -g cmd/api/main.go -o docs/swagger && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /api ./cmd/api

# ---- runner ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api /app/api

EXPOSE 8080

CMD ["/app/api"]
