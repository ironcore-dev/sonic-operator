# Provisioning (ZTP + ONIE)

The provisioning server serves ZTP scripts and ONIE installer artifacts over HTTP. It can run as part of the controller manager or as a standalone binary.

## Manager flags
- `--http-server-address`: bind address for the provisioning server.
- `--ztp-config-file`: JSON file with ZTP parameters (default `/etc/ztp.json`).
- `--ztp-mode`: ZTP source, either `templates` (default) or `configmap`. Modes are strict: ConfigMap mode never falls back to templates.
- `--onie-installer-dir`: directory containing ONIE installer files (default `/var/lib/sonic-operator/onie`).

## ZTP
- Scripts are rendered from templates in `internal/ztp/templates`.
- The source IP of the requesting switch is used to select parameters from the ZTP config file.
- The ZTP script is served at `GET /ztp`.

## ConfigMap ZTP mode

Set `--ztp-mode=configmap` to select a full ZTP script from a ConfigMap. The provisioning server matches the source address of `GET /ztp` to `spec.ztp.sourceAddress` on a `Switch`, then returns the configured ConfigMap key verbatim. A missing switch, ConfigMap, or key is an error; the leaf/spine templates are never used in this mode.

```yaml
apiVersion: sonic.networking.metal.ironcore.dev/v1alpha1
kind: Switch
metadata:
  name: leaf-01
spec:
  ztp:
    sourceAddress: "2001:db8:100::11"
    scriptRef:
      namespace: sonic-operator-system
      name: leaf-01-ztp
      key: ztp.sh
```

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: leaf-01-ztp
  namespace: sonic-operator-system
data:
  ztp.sh: |
    #!/bin/bash
    set -euo pipefail
    hostnamectl set-hostname leaf-01
```

## ONIE
- Files are served from the installer directory at HTTP root (`/`).
- This supports ONIE discovery workflows for delivering SONiC or other OS installers.
