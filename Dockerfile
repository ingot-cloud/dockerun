FROM --platform=$BUILDPLATFORM node:22.17.0-alpine AS frontend
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@10.12.4 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.26.0-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/dockerun-server . && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/dockerun ./cmd/dockerun

FROM docker:29.1.3-cli
RUN apk add --no-cache bash ca-certificates tzdata curl mariadb-client
WORKDIR /app
COPY --from=backend /out/ /usr/local/bin/
COPY --from=frontend /src/dist ./static
COPY templates ./templates
ENV DATA_DIR=/var/lib/dockerun GIN_MODE=release SERVER_ADDR=:8080 TEMPLATE_DIR=/app/templates
VOLUME /var/lib/dockerun
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD curl --fail http://localhost:8080/health || exit 1
CMD ["dockerun-server"]
