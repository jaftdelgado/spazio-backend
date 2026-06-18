FROM golang:1.26.4-alpine3.24 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

COPY . .

RUN sqlc generate -f sqlc/sqlc.yaml
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api/main.go

FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata libwebp-tools

COPY --from=builder /out/api /app/api

EXPOSE 8080

CMD ["/app/api"]
