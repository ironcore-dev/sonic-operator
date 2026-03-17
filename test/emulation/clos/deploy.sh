#!/bin/bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0

set -eu

# Setup Kind cluster for e2e tests if it does not exist
KIND_CLUSTER="clos-lab-kind"
echo "Setting up Kind cluster for tests..."
if ! command -v kind &> /dev/null; then
    echo "Kind is not installed. Please install Kind manually."
    exit 1
fi

if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER}$"; then
    echo "Kind cluster '${KIND_CLUSTER}' already exists. Skipping creation."
else
    echo "Creating Kind cluster '${KIND_CLUSTER}'..."
    kind create cluster --name "${KIND_CLUSTER}"
fi

# Go to git repo root
pushd "$(git rev-parse --show-toplevel)" || exit 1

echo "Installing CRDs..."
make install
# Return to original directory
popd || exit 1

HELM="docker run --network host -ti --rm -v $(pwd):/apps -w /apps \
    -v $HOME/.kube:/root/.kube -v $HOME/.helm:/root/.helm \
    -v $HOME/.config/helm:/root/.config/helm \
    -v $HOME/.cache/helm:/root/.cache/helm \
    alpine/helm:3.12.3"

CLABVERTER="sudo docker run --user $(id -u) -v $(pwd):/clabernetes/work --rm  ghcr.io/srl-labs/clabernetes/clabverter"

$HELM upgrade --install --create-namespace --namespace c9s \
    clabernetes oci://ghcr.io/srl-labs/clabernetes/clabernetes

kubectl apply -f https://kube-vip.io/manifests/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/kube-vip/kube-vip-cloud-provider/main/manifest/kube-vip-cloud-controller.yaml
kubectl create configmap --namespace kube-system kubevip \
  --from-literal range-global=172.18.1.10-172.18.1.250 || true

#set up the kube-vip CLI
KVVERSION=$(curl -sL https://api.github.com/repos/kube-vip/kube-vip/releases | \
  jq -r ".[0].name")
KUBEVIP="docker run --network host \
  --rm ghcr.io/kube-vip/kube-vip:$KVVERSION"
#install kube-vip load balancer daemonset in ARP mode
$KUBEVIP manifest daemonset --services --inCluster --arp --interface eth0 | \
kubectl apply -f -


echo "Checking for configuration changes..."
CONFIG=$($CLABVERTER --stdout --naming non-prefixed)

if echo "$CONFIG" | kubectl diff -f - > /dev/null 2>&1; then
  echo "No changes detected, skipping apply and wait"
else
  echo "Changes detected, applying configuration..."
  echo "$CONFIG" | kubectl apply -f -
  
  # Wait for services to be ready
  echo "Waiting for services to be ready..."
  kubectl wait --namespace c9s --for=condition=ready --timeout=300s pods --all
  kubectl wait --namespace c9s-clos --for=condition=ready --timeout=300s pods --all


  # Run script on each sonic node
  echo "Provisioning SONiC nodes..."
  for service in $(kubectl get -n c9s-clos svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n' | grep '^sonic-' | grep -v '\-vx$'); do
      until IP=$(kubectl get svc "$service" -n c9s-clos -o jsonpath='{.status.loadBalancer.ingress[0].ip}') && [ -n "$IP" ]; do
        echo "Waiting for external IP..."
        sleep 1
      done
      
    h=$(kubectl get -n c9s-clos svc "$service" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
    if [ ! -z "$h" ]; then
      echo "Running init_setup.sh on $h"
      max_attempts=36 # 36 attempts with 10 seconds sleep = 6 minutes total wait time
      attempt=1
      while [ $attempt -le $max_attempts ]; do
        if sshpass -p 'admin' ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null admin@"$h" 'bash -s' < init_setup.sh; then
          echo "Successfully provisioned $h"
          break
        else
          if [ $attempt -lt $max_attempts ]; then
            echo "Provisioning attempt $attempt of $max_attempts failed for $h. Retrying in 10 seconds..."
            sleep 10
          else
            echo "Failed to provision $h after $max_attempts attempts"
          fi
        fi
        ((attempt++))
      done
    fi
  done

fi


echo ""
echo "=========================================="
echo "SONiC Lab Topology - External IPs"
echo "=========================================="
for service in $(kubectl get -n c9s-clos svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n'| grep -v '\-vx$'); do
  ip=$(kubectl get -n c9s-clos svc "$service" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
  if [ -n "$ip" ]; then
    echo "$service -> $ip"

    if [[ "$service" == *sonic* ]]; then
      cat <<EOF | kubectl apply -f -
      apiVersion: sonic.networking.metal.ironcore.dev/v1alpha1
      kind: Switch
      metadata:
        labels:
          app.kubernetes.io/name: sonic-operator
          app.kubernetes.io/managed-by: kustomize
        name: $service
        namespace: c9s-clos    
      spec:
        management:
          host: $ip
          port: "57400"
          credentials:
            name: switchcredentials-sample
        macAddress: "aa:bb:cc:dd:ee:ff"
EOF
    fi
  fi
  
done

echo ""
echo "Script ended successfully"
