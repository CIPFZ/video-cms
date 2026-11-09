FROM golang:1.27.1-alpine AS builder

WORKDIR /app
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY} GOMAXPROCS=2 CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test -trimpath -p 2 ./... && \
    go build -trimpath -o main ./cmd/server && \
    go build -trimpath -o migrate ./cmd/migrate

FROM alpine:3.23
WORKDIR /app
ARG ALPINE_MIRROR=

# Required for timezones and ca-certs
RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR%/}#" /etc/apk/repositories; fi && \
    apk --no-cache add tzdata ca-certificates

COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY --from=builder /app/configs ./configs
COPY start.sh .

RUN sed -i 's/\r$//' start.sh && chmod +x start.sh

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=60s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/ready || exit 1

CMD ["./start.sh"]
