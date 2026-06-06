# 短剧生产工作台 · 前端镜像(多阶段:依赖 → Next standalone 构建 → 精简运行时)。
# 依赖 next.config 的 output:"standalone"(已开启)产出自带最小 node_modules 的 server.js。
# syntax=docker/dockerfile:1

# ---- deps ----
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# ---- build ----
FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* 在构建期被内联进 JS bundle。它是浏览器访问后端的地址,
# 必须用浏览器可达的 URL(默认本机发布出来的后端端口),而非 compose 内部服务名。
ARG NEXT_PUBLIC_API=http://localhost:8080
ENV NEXT_PUBLIC_API=$NEXT_PUBLIC_API
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- runtime ----
FROM node:20-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY --from=build /app/public ./public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
