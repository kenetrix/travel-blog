# Docker Compose

Runs the same setup as the [Proxmox container](../proxmox/README.md) in two containers: `api` (the Go API, SQLite and photos) and `web` (nginx with the built frontend, proxying `/api/` to `api`). In this folder:

```bash
docker compose up -d
```

- Open `https://<host>` (self-signed certificate; the API sets `Secure` cookies, so HTTPS is required). Behind your own reverse proxy, terminate TLS there and forward to `http://<host>:80` (port 80 serves the app when `X-Forwarded-Proto: https` is set). The API's port 8080 is not published.
- Ports: set `HTTP_PORT` and `HTTPS_PORT` (in the environment or a `.env` file here) when 80 or 443 are taken. The HTTP redirect always points to port 443, so with another `HTTPS_PORT` open `https://<host>:<port>` directly.
- Own certificate: put `travel-blog.crt` and `travel-blog.key` into the `certs` volume, or mount a folder with them at `/etc/nginx/ssl`, and restart `web`.
- First login: the first start creates the administrator `admin` with a generated password:

  ```bash
  docker compose exec api cat data/initial-admin.txt
  ```

  The file is kept until the password is changed, which the first login requires.
- Data: SQLite databases and photos are in the `data` volume and survive updates. Inspect the database with `docker compose exec api sqlite3 data/travel-blog.db`.
- Update: `docker compose pull && docker compose up -d`. Pin a release with `TRAVEL_BLOG_VERSION=<tag>` (from `v1.0.0` on; before that only the newest exists); the default is `latest`.
- Images: every `v*` tag publishes `ghcr.io/kenetrix/travel-blog-api` and `ghcr.io/kenetrix/travel-blog-web` for `amd64` and `arm64`, tagged with the version and `latest` (`.github/workflows/release.yml`). Before `v1.0.0` only the newest release's images are kept, like the GitHub release. To build them from this checkout instead: `docker compose up -d --build`.
- Single sign-on (OIDC, e.g. Pocket ID): put the `OIDC_*` variables from [docs/oidc.md](../docs/oidc.md) in `oidc.env` next to `compose.yaml` and run `docker compose up -d`.
