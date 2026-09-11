#!/bin/sh
set -e

# Setup iptables DROP RST inside container network stack
iptables -C OUTPUT -p tcp --tcp-flags RST RST -j DROP 2>/dev/null || \
  iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP

echo "=== OpenFlux Docker Exit Node ==="
echo "Transport: ${OPENFLUX_TRANSPORT:-yandex}"

EXTRA_ARGS=""
if [ -n "$OPENFLUX_KEY" ]; then
  EXTRA_ARGS="$EXTRA_ARGS -key $OPENFLUX_KEY"
fi
if [ -n "$OPENFLUX_YCOOKIE" ]; then
  EXTRA_ARGS="$EXTRA_ARGS -ycookie $OPENFLUX_YCOOKIE"
fi

exec /app/universal-bypass-tool \
  -exit-node \
  -transport "${OPENFLUX_TRANSPORT:-yandex}" \
  -url "${OPENFLUX_URL}" \
  $EXTRA_ARGS \
  -debug "$@"
