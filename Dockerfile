FROM golang:1.21.0-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -a -o main main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY --chmod=0755 start.sh .
COPY db/migration ./db/migration

CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]
