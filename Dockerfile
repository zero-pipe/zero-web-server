# syntax=docker/dockerfile:1

# ---------- build ----------
FROM golang:1.25-bookworm AS build

WORKDIR /src

# 先复制依赖声明与本地 replace 目标（3rdpart/），再下载依赖以利用缓存。
COPY go.mod go.sum ./
COPY 3rdpart ./3rdpart
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/zero-web-server ./cmd/server

# ---------- runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 zws

WORKDIR /app

COPY --from=build /out/zero-web-server /app/zero-web-server
COPY configs /app/configs

RUN mkdir -p /app/logs && chown -R zws:zws /app

USER zws

EXPOSE 18080 5060/udp 5060/tcp

VOLUME ["/app/logs"]

ENTRYPOINT ["/app/zero-web-server"]
CMD ["-config", "/app/configs/config.yaml"]
