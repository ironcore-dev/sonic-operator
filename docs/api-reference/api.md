# API Reference

## Packages
- [sonic.networking.metal.ironcore.dev/v1alpha1](#sonicnetworkingmetalironcoredevv1alpha1)


## sonic.networking.metal.ironcore.dev/v1alpha1

Package v1alpha1 contains API Schema definitions for the settings.gardener.cloud API group

Package v1alpha1 contains API Schema definitions for the networking v1alpha1 API group.

### Resource Types
- [Switch](#switch)
- [SwitchCredentials](#switchcredentials)
- [SwitchInterface](#switchinterface)



#### AdminState

_Underlying type:_ _string_





_Appears in:_
- [SwitchInterfaceSpec](#switchinterfacespec)
- [SwitchInterfaceStatus](#switchinterfacestatus)

| Field | Description |
| --- | --- |
| `Unknown` |  |
| `Up` |  |
| `Down` |  |


#### Management







_Appears in:_
- [SwitchSpec](#switchspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ |  |  |  |
| `port` _string_ |  |  |  |
| `credentials` _[ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectreference-v1-core)_ |  |  |  |


#### Neighbor



Neighbor represents a connected neighbor device.



_Appears in:_
- [SwitchInterfaceStatus](#switchinterfacestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `macAddress` _string_ | MacAddress is the MAC address of the neighbor device. |  |  |
| `systemName` _string_ | SystemName is the name of the neighbor device. |  |  |
| `interfaceHandle` _string_ | InterfaceHandle is the name of the remote switch interface. |  |  |


#### OperationState

_Underlying type:_ _string_





_Appears in:_
- [SwitchInterfaceStatus](#switchinterfacestatus)

| Field | Description |
| --- | --- |
| `Up` |  |
| `Down` |  |
| `Unknown` |  |


#### PortSpec







_Appears in:_
- [SwitchSpec](#switchspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |


#### PortStatus



PortStatus defines the observed state of a port on the Switch.



_Appears in:_
- [SwitchStatus](#switchstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the port. |  |  |
| `interfaceRefs` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#localobjectreference-v1-core) array_ | InterfaceRefs lists the references to Interfaces connected to this port. |  |  |


#### Switch



Switch is the Schema for the switch API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `sonic.networking.metal.ironcore.dev/v1alpha1` | | |
| `kind` _string_ | `Switch` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SwitchSpec](#switchspec)_ | spec defines the desired state of Switch |  |  |
| `status` _[SwitchStatus](#switchstatus)_ | status defines the observed state of Switch |  |  |


#### SwitchCredentials



SwitchCredentials is the Schema for the switchcredentials API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `sonic.networking.metal.ironcore.dev/v1alpha1` | | |
| `kind` _string_ | `SwitchCredentials` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `immutable` _boolean_ | Immutable, if set to true, ensures that data stored in the Secret cannot<br />be updated (only object metadata can be modified).<br />If not set to true, the field can be modified at any time.<br />Defaulted to nil. |  |  |
| `data` _object (keys:string, values:integer array)_ | Data contains the secret data. Each key must consist of alphanumeric<br />characters, '-', '_' or '.'. The serialized form of the secret data is a<br />base64 encoded string, representing the arbitrary (possibly non-string)<br />data value here. Described in https://tools.ietf.org/html/rfc4648#section-4 |  |  |
| `stringData` _object (keys:string, values:string)_ | stringData allows specifying non-binary secret data in string form.<br />It is provided as a write-only input field for convenience.<br />All keys and values are merged into the data field on write, overwriting any existing values.<br />The stringData field is never output when reading from the API. |  |  |
| `type` _[SecretType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secrettype-v1-core)_ | Used to facilitate programmatic handling of secret data.<br />More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types |  |  |


#### SwitchInterface



SwitchInterface is the Schema for the switchinterfaces API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `sonic.networking.metal.ironcore.dev/v1alpha1` | | |
| `kind` _string_ | `SwitchInterface` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SwitchInterfaceSpec](#switchinterfacespec)_ | spec defines the desired state of SwitchInterface |  |  |
| `status` _[SwitchInterfaceStatus](#switchinterfacestatus)_ | status defines the observed state of SwitchInterface |  |  |


#### SwitchInterfaceSpec



SwitchInterfaceSpec defines the desired state of SwitchInterface



_Appears in:_
- [SwitchInterface](#switchinterface)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `handle` _string_ | Handle uniquely identifies this interface on the switch. |  |  |
| `switchRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#localobjectreference-v1-core)_ | SwitchRef is a reference to the Switch this interface is connected to. |  |  |
| `adminState` _[AdminState](#adminstate)_ | AdminState represents the desired administrative state of the interface. |  |  |


#### SwitchInterfaceState

_Underlying type:_ _string_





_Appears in:_
- [SwitchInterfaceStatus](#switchinterfacestatus)

| Field | Description |
| --- | --- |
| `Pending` |  |
| `Ready` |  |
| `Failed` |  |


#### SwitchInterfaceStatus



SwitchInterfaceStatus defines the observed state of SwitchInterface.



_Appears in:_
- [SwitchInterface](#switchinterface)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `adminState` _[AdminState](#adminstate)_ | AdminState represents the desired administrative state of the interface. |  |  |
| `operationalState` _[OperationState](#operationstate)_ | OperationalState represents the actual operational state of the interface. |  |  |
| `state` _[SwitchInterfaceState](#switchinterfacestate)_ | State represents the high-level state of the SwitchInterface. |  |  |
| `neighbor` _[Neighbor](#neighbor)_ | Neighbor is a reference to the connected neighbor device, if any. |  |  |
| `macAddress` _string_ | MacAddress is the MAC address assigned to this interface. |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#condition-v1-meta) array_ | The status of each condition is one of True, False, or Unknown. |  |  |


#### SwitchSpec



SwitchSpec defines the desired state of Switch



_Appears in:_
- [Switch](#switch)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `management` _[Management](#management)_ |  |  |  |
| `macAddress` _string_ | MacAddress is the MAC address assigned to this interface. |  |  |
| `ports` _[PortSpec](#portspec) array_ | Ports the physical ports available on the Switch. |  |  |


#### SwitchState

_Underlying type:_ _string_

SwitchState represents the high-level state of the Switch.



_Appears in:_
- [SwitchStatus](#switchstatus)

| Field | Description |
| --- | --- |
| `Pending` |  |
| `Ready` |  |
| `Failed` |  |


#### SwitchStatus



SwitchStatus defines the observed state of Switch.



_Appears in:_
- [Switch](#switch)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `state` _[SwitchState](#switchstate)_ | State represents the high-level state of the Switch. |  |  |
| `ports` _[PortStatus](#portstatus) array_ | Ports represents the status of each port on the Switch. |  |  |
| `macAddress` _string_ | MACAddress is the MAC address assigned to this switch. |  |  |
| `firmwareVersion` _string_ | FirmwareVersion is the firmware version running on this switch. |  |  |
| `sku` _string_ | SKU is the stock keeping unit of this switch. |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#condition-v1-meta) array_ | The status of each condition is one of True, False, or Unknown. |  |  |


