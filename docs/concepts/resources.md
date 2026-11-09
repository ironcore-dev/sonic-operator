# Resources

All CRDs are **cluster-scoped** and reconciled by sonic-operator.

## Switch
Represents a physical switch and its management connectivity. `Ready` indicates
declarative acceptance for provisioning. Device status fields are retained but
are not currently refreshed.

Spec fields:
- `management.host`: switch management host/IP.
- `management.port`: management port (string).
- `macAddress`: MAC address assigned to the switch.
- `ports[]`: declared list of physical port names.

Status fields:
- `state`: `Pending`, `Ready`, `Failed`.
- `macAddress`: observed switch MAC.
- `firmwareVersion`: observed SONiC OS version.
- `sku`: observed hardware SKU.
- `ports[]`: observed ports and interface references.

## SwitchInterface
Represents a single interface and its admin/operational state. Resources remain
`Pending` because device observation and admin state enforcement are currently
unavailable. Existing observed status fields are retained.

Spec fields:
- `handle`: interface handle on the device (e.g. `Ethernet0`).
- `switchRef`: reference to the owning `Switch`.
- `adminState`: desired admin state (`Up`, `Down`, `Unknown`).

Status fields:
- `adminState`: observed admin state.
- `operationalState`: observed operational state.
- `neighbor`: neighbor details (when available).
