FROM golang:1.26-alpine3.23 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=$(cat VERSION)" -trimpath -a -o server ./cmd/server/

# Config (app.env, app_conf.yaml) is mounted at runtime, not baked in.
FROM alpine:3.23
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
COPY --chmod=0755 start.sh .
COPY internal/db/migration ./internal/db/migration

CMD [ "/app/server" ]
