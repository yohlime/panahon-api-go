FROM golang:1.21-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -a -o server ./cmd/server/

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/server .
COPY app.env .
COPY --chmod=0755 start.sh .
COPY internal/db/migration ./internal/db/migration

CMD [ "/app/server" ]
