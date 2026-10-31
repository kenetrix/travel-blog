#!/bin/sh
# Run by the nginx image's entrypoint. Creates a self-signed certificate once; it lives
# in a volume, so browsers keep their exception. Replace both files to use your own.
set -eu

dir=/etc/nginx/ssl
[ -f "$dir/travel-blog.crt" ] && [ -f "$dir/travel-blog.key" ] && exit 0

mkdir -p "$dir"
openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
    -keyout "$dir/travel-blog.key" -out "$dir/travel-blog.crt" \
    -subj "/CN=travel-blog" -addext "subjectAltName=DNS:travel-blog,DNS:localhost" 2>/dev/null
chmod 600 "$dir/travel-blog.key"
echo "$0: created a self-signed certificate in $dir"
