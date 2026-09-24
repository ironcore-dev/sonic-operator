# hack/host-services

Everything in this directory extends [sonic-hostservices](https://github.com/sonic-net/sonic-host-services) — the SONiC framework that allows containers to invoke privileged operations on the host via DBus.

## Purpose

Provide custom host modules that extend container-to-host communication beyond what the upstream `sonic-host-server` ships by default.

## Structure

| Path | Description |
|------|-------------|
| `modules/` | Custom host module implementations (Python, `HostModule` subclasses) |
| `install-custom-hostservice-modules` | Shell script that installs the modules into a running SONiC switch and restarts `sonic-hostservice.service` |

## Modules

### `modules/onie.py`

DBus endpoint (`host_modules.onie`) that sets ONIE install boot mode from within a container. Exposes a single method:

- **`set_install_mode()`** — mounts the `ONIE-BOOT` partition and writes the appropriate grub environment variables so the switch reboots into ONIE install mode on the next boot.

## Usage

Copy `modules/*.py` to `/usr/local/lib/python3.11/dist-packages/host_modules/` on the target switch, then run:

```bash
./install-custom-hostservice-modules
```

The script patches `sonic-host-server` to import and register the new modules, then restarts the service.
