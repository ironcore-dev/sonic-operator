#!/bin/bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

IMAGE="ghcr.io/ironcore-dev/sonic-agent:sha-966298d"

echo "Configuring DNS..."
if [ -d "/etc/resolvconf/resolv.conf.d" ]; then
  echo "nameserver 8.8.8.8" | sudo tee /etc/resolvconf/resolv.conf.d/head 
  sudo /sbin/resolvconf --enable-updates 
  sudo /sbin/resolvconf -u 
  sudo /sbin/resolvconf --disable-updates
else
  echo "Warning: resolvconf not found, skipping DNS configuration"
fi
echo "Removing old agent container if it exists..."
docker rm -f switch-operator-agent 2>/dev/null || true

echo "Pulling agent image..."
docker pull "$IMAGE"

echo "Starting agent container..."
docker run --pull always -d --name switch-operator-agent --entrypoint /switch-agent-server --network host --restart unless-stopped  "$IMAGE" -port 57400

echo "Agent setup completed successfully"
