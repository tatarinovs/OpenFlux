#!/bin/bash
set -e

echo "==> Stopping old un-isolated service..."
systemctl stop openflux-exit.service 2>/dev/null || true
systemctl disable openflux-exit.service 2>/dev/null || true
rm -f /etc/systemd/system/openflux-exit.service

echo "==> Cleaning global iptables rule on host..."
while iptables -C OUTPUT -p tcp --tcp-flags RST RST -j DROP 2>/dev/null; do
    iptables -D OUTPUT -p tcp --tcp-flags RST RST -j DROP
done

echo "==> Installing netns script and systemd units..."
install -m 755 /tmp/ofx-netns.sh /usr/local/sbin/ofx-netns.sh
install -m 644 /tmp/openflux-netns.service /etc/systemd/system/openflux-netns.service
install -m 644 /tmp/openflux.service /etc/systemd/system/openflux.service

echo "==> Writing /etc/openflux.env..."
cat << 'EOF' > /etc/openflux.env
OPENFLUX_URL=https://disk.yandex.ru/i/m12xOtr6rHrQkA
OPENFLUX_TRANSPORT=yandex
EOF
chmod 600 /etc/openflux.env

echo "==> Initializing network namespace..."
/usr/local/sbin/ofx-netns.sh

echo "==> Reloading systemd and enabling new services..."
systemctl daemon-reload
systemctl enable --now openflux-netns.service
systemctl enable --now openflux.service

echo "==> Cleanup temp files..."
rm -f /tmp/ofx-netns.sh /tmp/openflux-netns.service /tmp/openflux.service /tmp/remote_setup.sh

echo "==> Exit node redeployment finished successfully!"
