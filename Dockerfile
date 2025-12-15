# -------- build stage --------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# system deps
RUN apk add --no-cache git ca-certificates

# go mod cache
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o api ./cmd/api


# -------- runtime stage --------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# non-root user
USER nonroot:nonroot

# binary
COPY --from=builder /app/api /app/api

EXPOSE 8080

ENTRYPOINT ["/app/api"]
