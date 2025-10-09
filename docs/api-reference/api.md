<p>Packages:</p>
<ul>
<li>
<a href="#networking.metal.ironcore.dev%2fv1alpha1">networking.metal.ironcore.dev/v1alpha1</a>
</li>
</ul>
<h2 id="networking.metal.ironcore.dev/v1alpha1">networking.metal.ironcore.dev/v1alpha1</h2>
<div>
<p>Package v1alpha1 contains API Schema definitions for the settings.gardener.cloud API group</p>
</div>
Resource Types:
<ul></ul>
<h3 id="networking.metal.ironcore.dev/v1alpha1.AdminState">AdminState
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceSpec">SwitchInterfaceSpec</a>, <a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">SwitchInterfaceStatus</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Down&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Up&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.Management">Management
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchSpec">SwitchSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>host</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>port</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>credentials</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.Neighbor">Neighbor
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">SwitchInterfaceStatus</a>)
</p>
<div>
<p>Neighbor represents a connected neighbor device.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>macAddress</code><br/>
<em>
string
</em>
</td>
<td>
<p>MacAddress is the MAC address of the neighbor device.</p>
</td>
</tr>
<tr>
<td>
<code>systemName</code><br/>
<em>
string
</em>
</td>
<td>
<p>SystemName is the name of the neighbor device.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.OperationState">OperationState
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">SwitchInterfaceStatus</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Down&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Up&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.PortSpec">PortSpec
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchSpec">SwitchSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.PortStatus">PortStatus
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchStatus">SwitchStatus</a>)
</p>
<div>
<p>PortStatus defines the observed state of a port on the Switch.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the port.</p>
</td>
</tr>
<tr>
<td>
<code>interfaceRefs</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#localobjectreference-v1-core">
[]Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>InterfaceRefs lists the references to Interfaces connected to this port.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.Switch">Switch
</h3>
<div>
<p>Switch is the Schema for the switch API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata,omitempty,omitzero</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>metadata is a standard object metadata</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchSpec">
SwitchSpec
</a>
</em>
</td>
<td>
<p>spec defines the desired state of Switch</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>management</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.Management">
Management
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.PortSpec">
[]PortSpec
</a>
</em>
</td>
<td>
<p>Ports the physical ports available on the Switch.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status,omitempty,omitzero</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchStatus">
SwitchStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>status defines the observed state of Switch</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchCredentials">SwitchCredentials
</h3>
<div>
<p>SwitchCredentials is the Schema for the switchcredentials API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Standard object&rsquo;s metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>immutable</code><br/>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable, if set to true, ensures that data stored in the Secret cannot
be updated (only object metadata can be modified).
If not set to true, the field can be modified at any time.
Defaulted to nil.</p>
</td>
</tr>
<tr>
<td>
<code>data</code><br/>
<em>
map[string][]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>Data contains the secret data. Each key must consist of alphanumeric
characters, &lsquo;-&rsquo;, &lsquo;_&rsquo; or &lsquo;.&rsquo;. The serialized form of the secret data is a
base64 encoded string, representing the arbitrary (possibly non-string)
data value here. Described in <a href="https://tools.ietf.org/html/rfc4648#section-4">https://tools.ietf.org/html/rfc4648#section-4</a></p>
</td>
</tr>
<tr>
<td>
<code>stringData</code><br/>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>stringData allows specifying non-binary secret data in string form.
It is provided as a write-only input field for convenience.
All keys and values are merged into the data field on write, overwriting any existing values.
The stringData field is never output when reading from the API.</p>
</td>
</tr>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#secrettype-v1-core">
Kubernetes core/v1.SecretType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to facilitate programmatic handling of secret data.
More info: <a href="https://kubernetes.io/docs/concepts/configuration/secret/#secret-types">https://kubernetes.io/docs/concepts/configuration/secret/#secret-types</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchInterface">SwitchInterface
</h3>
<div>
<p>SwitchInterface is the Schema for the switchinterfaces API</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata,omitempty,omitzero</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>metadata is a standard object metadata</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceSpec">
SwitchInterfaceSpec
</a>
</em>
</td>
<td>
<p>spec defines the desired state of SwitchInterface</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>switchRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>SwitchRef is a reference to the Switch this interface is connected to.</p>
</td>
</tr>
<tr>
<td>
<code>adminState</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.AdminState">
AdminState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdminState represents the desired administrative state of the interface.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status,omitempty,omitzero</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">
SwitchInterfaceStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>status defines the observed state of SwitchInterface</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceSpec">SwitchInterfaceSpec
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterface">SwitchInterface</a>)
</p>
<div>
<p>SwitchInterfaceSpec defines the desired state of SwitchInterface</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>switchRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>SwitchRef is a reference to the Switch this interface is connected to.</p>
</td>
</tr>
<tr>
<td>
<code>adminState</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.AdminState">
AdminState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdminState represents the desired administrative state of the interface.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceState">SwitchInterfaceState
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">SwitchInterfaceStatus</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceStatus">SwitchInterfaceStatus
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterface">SwitchInterface</a>)
</p>
<div>
<p>SwitchInterfaceStatus defines the observed state of SwitchInterface.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>adminState</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.AdminState">
AdminState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdminState represents the desired administrative state of the interface.</p>
</td>
</tr>
<tr>
<td>
<code>operationalState</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.OperationState">
OperationState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>OperationalState represents the actual operational state of the interface.</p>
</td>
</tr>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchInterfaceState">
SwitchInterfaceState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>State represents the high-level state of the SwitchInterface.</p>
</td>
</tr>
<tr>
<td>
<code>neighbor</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.Neighbor">
Neighbor
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Neighbor is a reference to the connected neighbor device, if any.</p>
</td>
</tr>
<tr>
<td>
<code>macAddress</code><br/>
<em>
string
</em>
</td>
<td>
<p>MacAddress is the MAC address assigned to this interface.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The status of each condition is one of True, False, or Unknown.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchSpec">SwitchSpec
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.Switch">Switch</a>)
</p>
<div>
<p>SwitchSpec defines the desired state of Switch</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>management</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.Management">
Management
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.PortSpec">
[]PortSpec
</a>
</em>
</td>
<td>
<p>Ports the physical ports available on the Switch.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchState">SwitchState
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.SwitchStatus">SwitchStatus</a>)
</p>
<div>
<p>SwitchState represents the high-level state of the Switch.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="networking.metal.ironcore.dev/v1alpha1.SwitchStatus">SwitchStatus
</h3>
<p>
(<em>Appears on:</em><a href="#networking.metal.ironcore.dev/v1alpha1.Switch">Switch</a>)
</p>
<div>
<p>SwitchStatus defines the observed state of Switch.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.SwitchState">
SwitchState
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>State represents the high-level state of the Switch.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#networking.metal.ironcore.dev/v1alpha1.PortStatus">
[]PortStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ports represents the status of each port on the Switch.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The status of each condition is one of True, False, or Unknown.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
