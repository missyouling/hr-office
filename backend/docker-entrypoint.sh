#!/bin/sh
set -e

DATA_DIR="${SIAPP_DATA_DIR:-/app/data}"
UPLOAD_DIR="${SIAPP_UPLOAD_DIR:-}"

prepare_dir() {
  dir_path="$1"
  if [ -z "$dir_path" ]; then
    return
  fi
  if [ ! -d "$dir_path" ]; then
    mkdir -p "$dir_path" 2>/dev/null || true
  fi
  chown -R siapp:siapp "$dir_path" 2>/dev/null || true
}

prepare_dir "$DATA_DIR"
if [ -z "$UPLOAD_DIR" ]; then
  prepare_dir "$DATA_DIR/uploads"
else
  prepare_dir "$UPLOAD_DIR"
fi

exec su-exec siapp "$@"
