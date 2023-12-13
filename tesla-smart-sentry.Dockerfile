FROM golang:alpine as builder
WORKDIR /tmp/build
COPY ./src .
RUN CGO_ENABLED=0 go build -o app ./cmd/tesla-smart-sentry

FROM alpine:latest
RUN apk --no-cache add tzdata ca-certificates
COPY --from=builder /tmp/build/app /app
USER 10000:10000
ENTRYPOINT ["/app"]