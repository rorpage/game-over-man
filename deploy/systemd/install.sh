#!/usr/bin/env bash
# Installs game-over-man binary + systemd timer.
# Tries to download a pre-built binary from GitHub Releases; falls back to
# building from source if Go is available.
# Run as root: sudo ./install.sh
set -euo pipefail

INSTALL_DIR=/usr/local/bin
CONFIG_DIR=/etc/game-over-man
DATA_DIR=/var/lib/game-over-man
UNIT_DIR=/etc/systemd/system
REPO=rorpage/game-over-man
SERVICE_USER=game-over-man
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root: sudo $0"
  exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  *) GOARCH="" ;;
esac

BINARY_INSTALLED=false

# Try downloading a pre-built release binary
if [ -n "$GOARCH" ] && command -v curl &>/dev/null; then
  echo "Fetching latest release from GitHub..."
  LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | cut -d'"' -f4 || true)

  if [ -n "$LATEST" ]; then
    BINARY_URL="https://github.com/$REPO/releases/download/$LATEST/game-over-man-linux-$GOARCH"
    echo "Downloading $LATEST binary for linux/$GOARCH..."
    if curl -fsSL "$BINARY_URL" -o "$INSTALL_DIR/game-over-man"; then
      chmod +x "$INSTALL_DIR/game-over-man"
      echo "Installed $LATEST to $INSTALL_DIR/game-over-man"
      BINARY_INSTALLED=true
    else
      echo "Download failed, will try building from source."
    fi
  else
    echo "No releases found, will try building from source."
  fi
fi

# Fall back to building from source
if [ "$BINARY_INSTALLED" = false ]; then
  if ! command -v go &>/dev/null; then
    echo ""
    echo "Go is not installed and no pre-built binary was found."
    echo "Options:"
    echo "  1. Install Go from https://go.dev/dl/ and re-run this script"
    echo "  2. Download a binary directly from https://github.com/$REPO/releases"
    echo "     and copy it to $INSTALL_DIR/game-over-man"
    exit 1
  fi

  echo "Building from source with Go..."
  REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
  if [ -f "$REPO_ROOT/go.mod" ]; then
    (cd "$REPO_ROOT" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_DIR/game-over-man" .)
  else
    BUILD_DIR=$(mktemp -d)
    trap "rm -rf $BUILD_DIR" EXIT
    git clone --depth 1 "https://github.com/$REPO.git" "$BUILD_DIR"
    (cd "$BUILD_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_DIR/game-over-man" .)
  fi
  echo "Built and installed to $INSTALL_DIR/game-over-man"
fi

# Create dedicated system user
if ! id "$SERVICE_USER" &>/dev/null; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
  echo "Created system user: $SERVICE_USER"
fi

# Create directories with appropriate ownership
mkdir -p "$CONFIG_DIR" "$DATA_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
chmod 750 "$DATA_DIR" "$CONFIG_DIR"

# Install unit files
install -m 644 "$SCRIPT_DIR/game-over-man.service" "$UNIT_DIR/"
install -m 644 "$SCRIPT_DIR/game-over-man.timer"   "$UNIT_DIR/"

# Set up env file from example if not present
if [ ! -f "$CONFIG_DIR/env" ]; then
  install -m 640 "$SCRIPT_DIR/env.example" "$CONFIG_DIR/env"
  chown "root:$SERVICE_USER" "$CONFIG_DIR/env"
  echo ""
  echo "  -> Edit $CONFIG_DIR/env with your NOTIFICATION_URL."
fi

if [ ! -f "$CONFIG_DIR/config.json" ]; then
  echo "  -> Copy your config.json to $CONFIG_DIR/config.json."
fi

echo ""
systemctl daemon-reload
systemctl enable game-over-man.timer
echo "Timer enabled. Once your config.json and env file are in place:"
echo "  sudo systemctl start game-over-man.timer"
echo ""
echo "Commands:"
echo "  systemctl status game-over-man.timer    -- next scheduled run"
echo "  systemctl start game-over-man.service   -- run immediately"
echo "  journalctl -u game-over-man -f          -- follow logs"
