#!/usr/bin/env bash

# License: MIT
# Source: https://github.com/kenetrix/travel-blog

source /dev/stdin <<<"$FUNCTIONS_FILE_PATH"
color
verb_ip6
catch_errors
setting_up_container
network_check
update_os

msg_info "Installing Dependencies"
$STD apk add --no-cache nginx openssl curl ca-certificates sqlite
msg_ok "Installed Dependencies"

msg_info "Installing Travel Blog"
case "$(uname -m)" in
x86_64) ARCH="amd64" ;;
aarch64) ARCH="arm64" ;;
*) msg_error "Unsupported architecture" && exit 1 ;;
esac
RELEASE=$(curl -fsSL https://api.github.com/repos/kenetrix/travel-blog/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
if [[ -z "$RELEASE" ]]; then
  msg_error "No release found on github.com/kenetrix/travel-blog"
  exit 1
fi
TMP=$(mktemp -d)
curl -fsSL "https://github.com/kenetrix/travel-blog/releases/download/${RELEASE}/travel-blog-linux-${ARCH}.tar.gz" -o "$TMP/app.tar.gz"
tar -xzf "$TMP/app.tar.gz" -C "$TMP"
addgroup -S travelblog
adduser -S -D -H -G travelblog -h /opt/travel-blog -s /sbin/nologin travelblog
# nginx has to traverse /opt/travel-blog to reach web/; only data/ is private
install -d -m 755 /opt/travel-blog
install -d -o travelblog -g travelblog -m 750 /opt/travel-blog/data
install -m 755 "$TMP/travel-blog-api" /opt/travel-blog/travel-blog-api
cp -r "$TMP/web" /opt/travel-blog/web
echo "$RELEASE" >/opt/travel-blog/version.txt
rm -rf "$TMP"
msg_ok "Installed Travel Blog ${RELEASE}"

msg_info "Creating Service"
# The Go runtime is told to stay well inside the container's memory
cat <<'EOF' >/etc/init.d/travel-blog
#!/sbin/openrc-run

name="travel-blog"
description="Travel Blog API"
# The API reads ./data relative to its working directory, so start it through a shell that cd's there and logs.
# An optional oidc.env (OIDC_* variables, see docs/oidc.md) enables single sign-on.
command="/bin/sh"
command_args="-c 'cd /opt/travel-blog && set -a && { [ ! -f oidc.env ] || . ./oidc.env; } && exec ./travel-blog-api >>/var/log/travel-blog.log 2>&1'"
command_user="travelblog:travelblog"
command_background=true
pidfile="/run/travel-blog.pid"

export GOMEMLIMIT=64MiB
export GOGC=50

depend() {
    need net
}

start_pre() {
    checkpath -f -m 640 -o travelblog:travelblog /var/log/travel-blog.log
}
EOF
chmod +x /etc/init.d/travel-blog
$STD rc-update add travel-blog default
$STD rc-service travel-blog start
msg_ok "Created Service"

# The API sets Secure session cookies, so the browser has to reach it over HTTPS
msg_info "Configuring nginx"
IP=$(ip -4 -o addr show scope global | awk '{split($4,a,"/"); print a[1]; exit}')
install -d /etc/nginx/ssl
openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
  -keyout /etc/nginx/ssl/travel-blog.key -out /etc/nginx/ssl/travel-blog.crt \
  -subj "/CN=travel-blog" -addext "subjectAltName=DNS:travel-blog,IP:${IP}" &>/dev/null
chmod 600 /etc/nginx/ssl/travel-blog.key
sed -i 's/^worker_processes .*/worker_processes 1;/' /etc/nginx/nginx.conf
rm -f /etc/nginx/http.d/default.conf
cat <<'EOF' >/etc/nginx/http.d/travel-blog.conf
# Behind a TLS-terminating reverse proxy the original scheme arrives in X-Forwarded-Proto
map $http_x_forwarded_proto $tb_proto {
    default $scheme;
    https   https;
}

# Port 80 serves the app for a proxy that terminated TLS, and redirects everyone else to HTTPS
server {
    listen 80 default_server;
    listen [::]:80 default_server;

    if ($tb_proto != "https") {
        return 301 https://$host$request_uri;
    }

    include /etc/nginx/travel-blog-app.inc;
}

server {
    listen 443 ssl default_server;
    listen [::]:443 ssl default_server;
    ssl_certificate /etc/nginx/ssl/travel-blog.crt;
    ssl_certificate_key /etc/nginx/ssl/travel-blog.key;

    include /etc/nginx/travel-blog-app.inc;
}
EOF
cat <<'EOF' >/etc/nginx/travel-blog-app.inc
root /opt/travel-blog/web;
index index.html;
client_max_body_size 25m;

gzip on;
gzip_types text/css application/javascript application/json image/svg+xml;

location /api/ {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $tb_proto;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}

location /assets/ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}

location ~ ^/(sw\.js|registerSW\.js|manifest\.webmanifest|index\.html)$ {
    add_header Cache-Control "no-cache";
}

location / {
    try_files $uri /index.html;
}
EOF
nginx -t &>/dev/null
$STD rc-update add nginx default
$STD rc-service nginx restart
msg_ok "Configured nginx"

motd_ssh
customize
cleanup_lxc
