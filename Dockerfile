FROM golang:1.14-alpine AS builder
ARG BUILD_ARGS

WORKDIR /build
ENV CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download
RUN apk add build-base

COPY . .
RUN set -x \
￼  && go build $BUILD_ARGS ./cmd/rango

FROM alpine:3.12

RUN apk add ca-certificates
WORKDIR app
COPY --from=builder /build/rango ./
RUN mkdir -p /app/config

CMD ["./rango"]
