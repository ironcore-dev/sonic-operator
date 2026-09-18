# Provisioning (ZTP + ONIE)

The provisioning server serves ZTP scripts and ONIE installer artifacts over HTTP. It can run as part of the controller manager or as a standalone binary.

## Manager flags
- `--http-server-address`: bind address for the provisioning server.
- `--ztp-config-file`: JSON file with ZTP parameters (default `/etc/ztp.json`).
- `--ztp-mode`: ZTP source: `templates` (default), `configmap`, or `generated`.
- `--bootstrap-control-kubeconfig-file`: optional control kubeconfig mounted into the operator and made available to opted-in generated containers.
- `--onie-installer-dir`: directory containing ONIE installer files (default `/var/lib/sonic-operator/onie`).

## ZTP
- `templates` renders the static scripts in `internal/ztp/templates` from the JSON ZTP configuration.
- `configmap` serves the selected ConfigMap key verbatim.
- `generated` renders one complete script from the matching `Switch` object.
- The ZTP script is served at `GET /ztp`.

## ConfigMap ZTP mode

Set `--ztp-mode=configmap` to select a full ZTP script from a ConfigMap. The provisioning server matches the source address of `GET /ztp` to `spec.ztp.sourceAddress` on a `Switch`, then returns the configured ConfigMap key verbatim. A missing switch, ConfigMap, or key is an error; no generated commands are appended and the leaf/spine templates are never used in this mode.

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

## Generated ZTP mode

Set `--ztp-mode=generated` to render the entire ZTP script from the matching
`Switch` object. No ConfigMap script is read in this mode. The generated script
configures the hostname, saves SONiC configuration, writes any requested global
credential, and starts declared Docker containers.

```yaml
apiVersion: sonic.networking.metal.ironcore.dev/v1alpha1
kind: Switch
metadata:
  name: switch-1
spec:
  hostname: switch-1
  ztp:
    sourceAddress: "2001:db8:100::11"
  containers:
  - name: wirelet
    image: ghcr.io/ironcore-dev/wirelet:fixed-1
    securityContext:
      runAsUser: 65532
      runAsGroup: 65532
    args:
    - --name=switch-1
    - --interface=Ethernet0
    injectControlKubeconfig: true
```

Each container is pulled and started with host networking and Docker's
`unless-stopped` restart policy. `command` overrides the image entrypoint;
`args` are appended after `command` or the image entrypoint. `hostname`
defaults to the `Switch` object name when omitted. Containers are
best effort: a failure to prepare or start one is logged and does not prevent
other containers from being attempted.

### Volumes

Generated ZTP follows the Kubernetes Pod volume model: define named volumes in
`spec.volumes`, then refer to them from a container's `volumeMounts`. The
current generated Docker runtime supports `hostPath` volumes only:

```yaml
spec:
  volumes:
  - name: dbus
    hostPath:
      path: /var/run/dbus
      type: Directory
  containers:
  - name: sonic-agent
    image: ghcr.io/giluerre/sonic-agent:latest
    command: ["/switch-agent-server"]
    args: ["-port", "57400"]
    securityContext:
      runAsUser: 0
    volumeMounts:
    - name: dbus
      mountPath: /var/run/dbus
```

This renders a Docker mount before the image, for example
`-v /var/run/dbus:/var/run/dbus:rw`. The volume and mount names must match;
host paths and mount paths must be absolute.

### ONIE boot discovery

Generated ZTP can persistently configure the next SONiC reboot to enter ONIE
install discovery:

```yaml
spec:
  nextBootMode: InstallOS
```

The generated script creates an `onieboot.mount` unit for the `ONIE-BOOT`
partition and a `sonic-operator-onie-install.service` unit. The latter runs on
every SONiC boot and sets `next_entry=ONIE` plus `onie_mode=install`. It does
not reboot the switch immediately; the next reboot enters ONIE install mode.
`None` (or omitting `nextBootMode`) leaves the normal boot path unchanged.
This boot-lifecycle setup is rendered before hostname and bootstrap-container
steps, and remains strict: a failure to configure it stops the script.

### Global control kubeconfig

`injectControlKubeconfig: true` is an explicit opt-in to the operator's global
control kubeconfig. The default manager manifest mounts a Secret named
`control-kubeconfig` in the operator namespace at
`/etc/sonic-operator/control-kubeconfig/kubeconfig` and passes that path with
`--bootstrap-control-kubeconfig-file`.

Create the Secret before provisioning a switch that requests injection:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: control-kubeconfig
  namespace: sonic-operator-system
type: Opaque
stringData:
  kubeconfig: |
    # a deliberately restricted control-cluster kubeconfig
```

Containers use `securityContext.runAsUser` and `securityContext.runAsGroup`,
which map to Docker's `--user` option. The generated ZTP script writes a
private credential file per opted-in container, owned by that container's
configured user/group with mode `0600`, bind-mounts it read-only at
`/var/run/sonic-operator/control-kubeconfig`, and sets `KUBECONFIG` to that
path in the opted-in container. `injectControlKubeconfig: true` requires
`securityContext.runAsUser`.

This is a shared bootstrap credential. Use a restricted identity only; do not
mount an administrator kubeconfig. The current ZTP endpoint is HTTP, so this
mechanism is appropriate for controlled lab networks only until credential
delivery is protected by HTTPS and a short-lived bootstrap flow.

## ONIE
- Files are served from the installer directory at HTTP root (`/`).
- This supports ONIE discovery workflows for delivering SONiC or other OS installers.
