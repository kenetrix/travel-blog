# Proxmox helper script

Creates a lightweight LXC (Alpine 3.24, 1 vCPU, 256 MB RAM, 1 GB disk) running nginx and the Go API.
The defaults are sized so the running app uses about 60% of them. Run on the Proxmox host:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/kenetrix/travel-blog/main/proxmox/ct/travelblog.sh)"
```

- Open `https://<container-ip>` (self-signed certificate; the API sets `Secure` cookies, so HTTPS is required). Behind your own reverse proxy, terminate TLS there and forward to `http://<container-ip>:80` (port 80 serves the app when `X-Forwarded-Proto: https` is set). Do not forward to `:8080`; that is the API alone and listens on localhost only.
- First login: the first start creates the administrator `admin` with a generated password. It is printed at the end of the install (and of an update, if the account was just created) and kept in `/opt/travel-blog/data/initial-admin.txt` until the password is changed, which the first login requires.
- Database: the `sqlite3` tool is installed (also on update) to inspect or repair it.
- Update: run the same command again in the Proxmox shell and choose update. Data in `/opt/travel-blog/data` (SQLite and photos) is kept.
- Needs a GitHub release: push a tag like `v0.1.0` and `.github/workflows/release.yml` builds the `amd64` and `arm64` tarballs the installer downloads.
- Single sign-on (OIDC, e.g. Pocket ID): put the `OIDC_*` variables from [docs/oidc.md](../docs/oidc.md) in `/opt/travel-blog/oidc.env` (`chmod 600`, owner `travelblog`) and run `rc-service travel-blog restart`. Containers created before this option need the `oidc.env` line from `travelblog-install.sh` in `/etc/init.d/travel-blog`.
