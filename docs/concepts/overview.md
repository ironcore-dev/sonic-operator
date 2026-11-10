# Overview

sonic-operator is a Kubernetes-native, declarative operator for onboarding and managing the lifecycle of bare-metal network switches.

## Architecture
- **Controller manager** runs in the cluster, reconciles CRDs, and maintains status.
- **Provisioning server** serves ZTP scripts and ONIE installer artifacts over HTTP.

## Reconciliation flow
1. A `Switch` CR is created to represent a physical switch and its management endpoint.
2. The controller marks the `Switch` as `Ready` to indicate declarative acceptance for provisioning.
3. `SwitchInterface` resources remain `Pending`; device discovery and interface state enforcement are currently unavailable.
4. Both controllers maintain finalizers and release them on deletion.

The former unauthenticated Sonic agent has been removed. Device status fields are
retained for API compatibility but are no longer refreshed by the controllers.

## Provisioning flow
- ZTP scripts are generated from matching `Switch` objects or served from referenced ConfigMaps at `GET /ztp`.
- ONIE installers are served from a configured directory at HTTP root (`/`).
- ZTP is served by the provisioning server embedded in the manager. The standalone provisioning-server binary serves ONIE artifacts only.
