FROM golang:1.20-alpine AS builder

RUN apk update && apk add curl gcc g++ libc-dev

ARG KAIGARA_VERSION=v1.0.34
# Install Kaigara
RUN curl -Lso /usr/bin/kaigara https://github.com/openware/kaigara/releases/download/${KAIGARA_VERSION}/kaigara \
  && chmod +x /usr/bin/kaigara

WORKDIR /build
ENV CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build ./cmd/rango


FROM alpine:3.18

RUN apk add ca-certificates
WORKDIR app
COPY --from=builder /build/rango ./
COPY --from=builder /usr/bin/kaigara /usr/bin/kaigara
RUN mkdir -p /app/config

CMD ["./rango"]
