# 短剧生产工作台 · 后端镜像(多阶段:Go 编译 → 精简 alpine 运行时)。
# 提示词模板通过 Go embed 内嵌进二进制,无需额外拷贝。
# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.23-alpine AS build
WORKDIR /src
# 先拷依赖清单,利用层缓存
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime ----
FROM alpine:3.20
# HTTPS 调用 Gemini/Vertex 需要根证书;tzdata 便于日志时间
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
