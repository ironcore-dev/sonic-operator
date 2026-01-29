# Switch agent

The switch agent runs on the switch and exposes device and interface operations via gRPC. The controller connects to this agent to observe and enforce state.

## Binaries
- `cmd/agent/main.go`: gRPC server deployed on the switch.
- `cmd/agent_cli/main.go`: CLI client for the agent API (useful for diagnostics).

## Capabilities (high level)
- Get device info (MAC, HWSKU, SONiC OS version).
- List ports and interfaces.
- Get interface state.
- Set interface admin state.
- Get neighbor info (when available).

## Notes
The current implementation uses SONiC Redis as the data source for switch state.
