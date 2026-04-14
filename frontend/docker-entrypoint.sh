#!/bin/sh
set -e

RUNTIME_FILE="/app/public/runtime-config.js"

mkdir -p "$(dirname "$RUNTIME_FILE")"

cat > "$RUNTIME_FILE" <<EOF_RUNTIME
window.__RUNTIME_CONFIG__ = {
  API_BASE: "${NEXT_PUBLIC_API_BASE_URL:-}",
  API_BASE_DOMAIN: "${NEXT_PUBLIC_API_BASE_URL_DOMAIN:-}",
  API_BASE_IP: "${NEXT_PUBLIC_API_BASE_URL_IP:-}",
  API_IPV4_FALLBACK_PORT: "${NEXT_PUBLIC_API_IPV4_FALLBACK_PORT:-}"
};
EOF_RUNTIME

echo "[entrypoint] runtime-config.js generated:"
cat "$RUNTIME_FILE"

echo "[entrypoint] starting application..."
exec "$@"
