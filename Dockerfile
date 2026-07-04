FROM golang:1.26-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./

ENV GO111MODULE="on"
ENV GOPROXY="https://goproxy.cn"
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /ad-server ./cmd/ad-server
RUN CGO_ENABLED=0 go build -o /ad-manager ./cmd/ad-manager
RUN CGO_ENABLED=0 go build -o /ad-syncer ./cmd/ad-syncer
RUN CGO_ENABLED=0 go build -o /file-gateway ./cmd/file-gateway
RUN CGO_ENABLED=0 go build -o /feature-store ./cmd/feature-store
RUN CGO_ENABLED=0 go build -o /roi-daemon ./cmd/roi-daemon
RUN CGO_ENABLED=0 go build -o /clickhouse-writer ./cmd/clickhouse-writer

FROM node:22-alpine AS web-builder

RUN corepack enable && corepack prepare pnpm@10.26.2 --activate

WORKDIR /web
COPY web/pnpm-lock.yaml web/package.json ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm run build

FROM alpine:3.19 AS backend

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=go-builder /ad-server /ad-manager /ad-syncer /file-gateway /feature-store /roi-daemon /clickhouse-writer /usr/local/bin/
COPY --from=go-builder /app/config/app.yaml /config/app.yaml

EXPOSE 8080 8081 9090 9091

FROM nginx:alpine AS frontend

COPY --from=web-builder /web/dist /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

FROM node:22-alpine AS frontend-dev

RUN corepack enable && corepack prepare pnpm@10.26.2 --activate

WORKDIR /web
COPY web/pnpm-lock.yaml web/package.json ./
RUN pnpm install --frozen-lockfile

COPY web/ ./

EXPOSE 5173
CMD ["pnpm", "run", "dev", "--host", "0.0.0.0"]
