#!/usr/bin/env bash
# Install scripts (install/, ct/headers/) are fetched from this repository, the framework itself from community-scripts
export COMMUNITY_SCRIPTS_URL="${COMMUNITY_SCRIPTS_URL:-https://raw.githubusercontent.com/kenetrix/travel-blog/main/proxmox}"
_cs_boot="${COMMUNITY_SCRIPTS_CORE_DIR:-$(dirname "${BASH_SOURCE[0]}")/../../core}/core/build.func"
source "$_cs_boot" 2>/dev/null || source <(curl -fsSL "${COMMUNITY_SCRIPTS_CORE_URL:-https://raw.githubusercontent.com/community-scripts/core/main}/core/build.func")
# License: MIT
# Source: https://github.com/kenetrix/travel-blog

APP="Travel Blog"
var_tags="${var_tags:-travel;blog}"
# Alpine + nginx + Go API + SQLite idles at ~60 MB RAM / ~60 MB disk; 256 MB leaves room for photo uploads, 1 GB for the photos
var_cpu="${var_cpu:-1}"
var_ram="${var_ram:-256}"
var_disk="${var_disk:-1}"
var_os="${var_os:-alpine}"
var_version="${var_version:-3.24}"
var_arm64="${var_arm64:-yes}"
var_unprivileged="${var_unprivileged:-1}"

header_info "$APP"
variables
color
catch_errors

function update_script() {
  header_info
  if [[ ! -d /opt/travel-blog ]]; then
    msg_error "No ${APP} Installation Found!"
    exit
  fi

  local arch latest current
  case "$(uname -m)" in
  x86_64) arch="amd64" ;;
  aarch64) arch="arm64" ;;
  *) msg_error "Unsupported architecture" && exit ;;
  esac
  latest=$(curl -fsSL https://api.github.com/repos/kenetrix/travel-blog/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
  current=$(cat /opt/travel-blog/version.txt 2>/dev/null || true)
  if [[ -z "$latest" ]]; then
    msg_error "Could not determine the latest release"
    exit
  fi
  if [[ "$latest" == "$current" ]]; then
    msg_ok "${APP} is already up to date (${current})"
    exit
  fi

  # The sqlite3 command line tool, to inspect or repair the database
  apk add --no-cache sqlite >/dev/null

  msg_info "Stopping Service"
  rc-service travel-blog stop
  msg_ok "Stopped Service"

  msg_info "Updating ${APP} to ${latest}"
  local tmp
  tmp=$(mktemp -d)
  curl -fsSL "https://github.com/kenetrix/travel-blog/releases/download/${latest}/travel-blog-linux-${arch}.tar.gz" -o "$tmp/app.tar.gz"
  tar -xzf "$tmp/app.tar.gz" -C "$tmp"
  # data/ (database and photos) is never touched
  install -m 755 "$tmp/travel-blog-api" /opt/travel-blog/travel-blog-api
  rm -rf /opt/travel-blog/web
  cp -r "$tmp/web" /opt/travel-blog/web
  echo "$latest" >/opt/travel-blog/version.txt
  rm -rf "$tmp"
  msg_ok "Updated ${APP}"

  msg_info "Starting Service"
  rc-service travel-blog start
  msg_ok "Started Service"
  msg_ok "Updated successfully!"

  # A release with roles creates the administrator on its first start. The
  # credentials stay in this file until the password has been changed.
  for _ in 1 2 3 4 5; do
    [[ -f /opt/travel-blog/data/initial-admin.txt ]] && break
    sleep 1
  done
  if [[ -f /opt/travel-blog/data/initial-admin.txt ]]; then
    echo -e "${INFO}${YW}Administrator login (you must choose a new password at the first login):${CL}"
    while read -r line; do
      echo -e "${TAB}${BGN}${line}${CL}"
    done </opt/travel-blog/data/initial-admin.txt
  fi
  exit
}

start
build_container
description

msg_ok "Completed successfully!\n"
echo -e "${CREATING}${GN}${APP} setup has been successfully initialized!${CL}"
echo -e "${INFO}${YW}Access it using the following URL (self-signed certificate):${CL}"
echo -e "${TAB}${GATEWAY}${BGN}https://${IP}${CL}"

# The first start creates the administrator; its generated login is shown
# here once, and the account has to choose its own password at the first login
for _ in 1 2 3 4 5 6 7 8 9 10; do
  pct exec "$CTID" -- test -f /opt/travel-blog/data/initial-admin.txt && break
  sleep 1
done
if CREDENTIALS=$(pct exec "$CTID" -- cat /opt/travel-blog/data/initial-admin.txt 2>/dev/null); then
  echo -e "${INFO}${YW}Administrator login (you must choose a new password at the first login):${CL}"
  while read -r line; do
    echo -e "${TAB}${BGN}${line}${CL}"
  done <<<"$CREDENTIALS"
  echo -e "${INFO}${YW}The login stays in /opt/travel-blog/data/initial-admin.txt until the password is changed.${CL}"
fi
