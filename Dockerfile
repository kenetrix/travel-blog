# Images for docker/compose.yaml: target "api" (Go API) and target "web" (nginx with the
# built frontend). Build stages run natively and cross-compile, so multi-arch builds stay fast.

FROM --platform=$BUILDPLATFORM node:26.12.1-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# Shown in the footer; without it the build falls back to "dev" (no .git in the context)
ARG APP_VERSION
ENV APP_VERSION=$APP_VERSION
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/travel-blog-api ./cmd/api

FROM alpine:3.24 AS api
# sqlite to inspect or repair the database, as in the Proxmox container
RUN apk add --no-cache ca-certificates sqlite \
    && addgroup -S travelblog \
    && adduser -S -D -H -G travelblog -h /opt/travel-blog -s /sbin/nologin travelblog \
    && install -d -o travelblog -g travelblog -m 750 /opt/travel-blog/data
# The API reads ./data relative to its working directory
WORKDIR /opt/travel-blog
COPY --from=backend /out/travel-blog-api ./
USER travelblog
ENV LISTEN_ADDR=:8080
EXPOSE 8080
VOLUME /opt/travel-blog/data
ENTRYPOINT ["./travel-blog-api"]

FROM nginx:1.28-alpine AS web
RUN apk add --no-cache openssl && rm /etc/nginx/conf.d/default.conf
COPY docker/nginx/travel-blog.conf /etc/nginx/conf.d/
COPY docker/nginx/travel-blog-app.inc /etc/nginx/
COPY --chmod=755 docker/nginx/40-self-signed-cert.sh /docker-entrypoint.d/
COPY --from=frontend /src/frontend/dist /usr/share/nginx/html
EXPOSE 80 443
