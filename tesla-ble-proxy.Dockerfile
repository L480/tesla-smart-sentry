FROM golang:alpine as builder
WORKDIR /tmp/build
COPY ./src .
RUN CGO_ENABLED=0 go build -o app ./cmd/tesla-ble-proxy

FROM alpine:latest
RUN apk --no-cache add tzdata
COPY --from=builder /tmp/build/app /app
EXPOSE 8080
USER 10000:10000
ENTRYPOINT ["/app"]