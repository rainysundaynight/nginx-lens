#!/usr/bin/env bash
# ---------- Установщик nginx-lens ----------
# Скачивает бинарники с GitHub Releases и кладёт в PATH.

set -euo pipefail

REPO="rainysundaynight/nginx-lens"
VERSION="${NGINX_LENS_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${NGINX_LENS_CONFIG_DIR:-/opt/nginx-lens}"

detect_platform() {
  OS=$(uname -s)
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Неподдерживаемая архитектура: $ARCH"; exit 1 ;;
  esac
  case "$OS" in
    Linux)  PLATFORM="Linux" ;;
    Darwin) PLATFORM="Darwin" ;;
    *) echo "Неподдерживаемая ОС: $OS (используйте install.ps1 на Windows)"; exit 1 ;;
  esac
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | sed -n 's/.*"tag_name": "v\(.*\)".*/\1/p' | head -1)
    if [ -z "$VERSION" ]; then
      echo "Не удалось определить последний релиз. Задайте NGINX_LENS_VERSION=2.3.0"
      exit 1
    fi
    echo "Последний релиз: v${VERSION}"
  fi
}

install_binaries() {
  detect_platform
  resolve_version
  ARCHIVE="nginx-lens_${VERSION}_${PLATFORM}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"
  TMP=$(mktemp -d)
  trap 'rm -rf "$TMP"' EXIT
  echo "Скачивание ${URL}..."
  curl -fsSL "$URL" -o "$TMP/${ARCHIVE}"
  tar xzf "$TMP/${ARCHIVE}" -C "$TMP"
  mkdir -p "$INSTALL_DIR"
  for bin in nginx-lens nginx-lens-agent nginx-lens-hub; do
    if [ -f "$TMP/$bin" ]; then
      install -m 755 "$TMP/$bin" "$INSTALL_DIR/$bin"
      echo "Установлен: $INSTALL_DIR/$bin"
    fi
  done
  if [ -f "$TMP/example-config.yaml" ]; then
    mkdir -p "$CONFIG_DIR"
    if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
      cp "$TMP/example-config.yaml" "$CONFIG_DIR/config.yaml"
      echo "Конфиг: $CONFIG_DIR/config.yaml"
    fi
  fi
}

install_binaries
echo ""
echo "Готово. Следующие шаги:"
echo "  sudo nginx-lens init    # создаст ${CONFIG_DIR}/config.yaml"
echo "  # отредактируйте defaults.nginx_config_path в конфиге"
echo "  nginx-lens config validate"
echo "  nginx-lens validate"
