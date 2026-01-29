# Overview

switch-operator is a Kubernetes-native, declarative operator for onboarding and managing the lifecycle of bare-metal network switches.

## Architecture
- **Controller manager** runs in the cluster, reconciles CRDs, and maintains status.
- **Switch agent** runs on the switch and exposes device/port/interface operations via gRPC.
- **Provisioning server** serves ZTP scripts and ONIE installer artifacts over HTTP.

## Reconciliation flow
1. A `Switch` CR is created to represent a physical switch and its management endpoint.
2. The controller connects to the switch agent and observes device state.
3. The controller creates/updates `SwitchInterface` CRs based on discovered interfaces.
4. Interface admin state in `SwitchInterface.spec` is enforced on the device.
5. Status fields are updated on `Switch`, `SwitchInterface` based on observed state.

## Provisioning flow
- ZTP scripts are rendered from templates and served at `GET /ztp`.
- ONIE installers are served from a configured directory at HTTP root (`/`).
- The provisioning server can run embedded in the manager or as a standalone binary.
