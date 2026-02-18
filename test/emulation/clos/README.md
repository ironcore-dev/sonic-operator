# SONiC Lab Topology (CLOS)

A containerized network lab environment running SONiC switches in a CLOS topology, orchestrated on Kubernetes using Clabernetes.

## Overview

This project sets up a complete network topology with:
- **2 Spine switches** (SONiC VMs)
- **2 Leaf switches** (SONiC VMs)
- **2 Client nodes** (Linux multitool containers)

The topology implements a standard data center CLOS architecture.

### Topology Diagram

![CLOS Topology](clos_topology.svg)

## Prerequisites

The following tools must be installed on the host:

- **kind** - Kubernetes in Docker (for local Kubernetes cluster)
- **kubectl** - Kubernetes command-line tool
- **sshpass** - SSH password automation utility

## Project Structure

```
clos/
├── clos.clab.yml      - Network topology definition (YAML)
├── deploy.sh          - Deployment automation script
├── init_setup.sh      - Node initialization and agent setup
├── destroy.sh         - Infrastructure cleanup script
└── README.md          - This file
```

## Dependencies


### Software Packages
```
docker          - Container runtime
kubernetes      - Container orchestration
helm            - Package manager
kubectl         - Kubernetes CLI
sshpass         - SSH password utility
jq              - JSON processor
```

### Kubernetes Services
- Clabernetes - Deployed via Helm in `c9s` namespace
- kube-vip - RBAC and manifests applied to cluster
- kube-vip Cloud Controller - Deployed in `kube-vip` namespace


## Configuration Details

### IP Management
- kube-vip External IP Range: `172.18.1.10 - 172.18.1.250`
- Services exposed via kube-vip ARP mode on eth0

## Setup Steps

### 1. Prerequisites
Ensure all dependencies are installed and Kubernetes cluster is ready:

### 2. Initialize Kind Cluster
Run the initialization setup from the root level Makefile by:
```bash
make setup-test-e2e
```

### 3. Deploy the Lab Environment
Deploy the full topology to Kubernetes:
```bash
./deploy.sh
```

**What it does**:
- Installs Clabernetes via Helm in `c9s` namespace
- Applies kube-vip RBAC policies
- Deploys kube-vip cloud controller
- Creates kube-vip configmap with IP range
- Deploys kube-vip ARP daemonset
- Converts containerlab topology to Kubernetes resources
- Applies topology configuration to cluster
- Waits for services to be ready (180 seconds)
- Configure DNS, Pulls and starts Sonic Agenton port 57400 for each SONiC node via SSH
- Displays external IPs for all services

### 4. Access the Lab
After successful deployment, retrieve external IPs:
```bash
# View all services with external IPs
kubectl get -n c9s-clos svc

# SSH into a specific SONiC node (default credentials: admin/admin)
ssh admin@<external-ip>

# Example
ssh admin@172.18.1.15
```

### 5. Cleanup
Tear down the entire lab environment:
```bash
./destroy.sh
```

**What it does**:
- Deletes the `c9s-clos` namespace (all topology resources)
- Deletes the `c9s` namespace (Clabernetes)
- Removes kube-vip configmap, daemonset, and cloud controller
- Cleans up kube-vip RBAC resources
- Removes all related Kubernetes objects
