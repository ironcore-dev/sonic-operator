# Provisioning (ZTP + ONIE)

The provisioning server serves ZTP scripts and ONIE installer artifacts over HTTP. It can run as part of the controller manager or as a standalone binary.

## Manager flags
- `--http-server-address`: bind address for the provisioning server.
- `--ztp-config-file`: JSON file with ZTP parameters (default `/etc/ztp.json`).
- `--onie-installer-dir`: directory containing ONIE installer files (default `/var/lib/switch-operator/onie`).

## ZTP
- Scripts are rendered from templates in `internal/ztp/templates`.
- The source IP of the requesting switch is used to select parameters from the ZTP config file.
- The ZTP script is served at `GET /ztp`.

## ONIE
- Files are served from the installer directory at HTTP root (`/`).
- This supports ONIE discovery workflows for delivering SONiC or other OS installers.
