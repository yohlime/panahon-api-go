FROM golang:1.20-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags "-s" -o main main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY --chmod=0755 start.sh .
COPY db/migration ./db/migration

EXPOSE 8080
CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]
