#!/usr/bin/env sh
# Regenerates internal/adapter/http/api/api.gen.go from api/openapi.yaml.
set -e
cd "$(dirname "$0")/.."
go generate ./internal/adapter/http/api
echo "Generated internal/adapter/http/api/api.gen.go"
