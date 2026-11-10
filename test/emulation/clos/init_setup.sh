#!/bin/bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

echo "Configuring DNS..."
if [ -d "/etc/resolvconf/resolv.conf.d" ]; then
  echo "nameserver 8.8.8.8" | sudo tee /etc/resolvconf/resolv.conf.d/head 
  sudo /sbin/resolvconf --enable-updates 
  sudo /sbin/resolvconf -u 
  sudo /sbin/resolvconf --disable-updates
else
  echo "Warning: resolvconf not found, skipping DNS configuration"
fi

echo "Node initialization completed successfully"
