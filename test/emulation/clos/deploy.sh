#!/bin/bash
set -eu
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
  sleep 180

  # Run script on each sonic node
  echo "Provisioning SONiC nodes..."
  for service in $(kubectl get -n c9s-clos svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n' | grep '^sonic-'); do
    h=$(kubectl get -n c9s-clos svc "$service" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
    if [ ! -z "$h" ]; then
      echo "Running init_setup.sh on $h"
      sshpass -p 'admin' ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null admin@"$h" 'bash -s' < init_setup.sh || true
    fi
  done

fi


echo ""
echo "=========================================="
echo "SONiC Lab Topology - External IPs"
echo "=========================================="
for service in $(kubectl get -n c9s-clos svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n'); do
  ip=$(kubectl get -n c9s-clos svc "$service" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
  if [ -n "$ip" ]; then
    echo "$service -> $ip"
  fi
done

echo ""
echo "Script ended successfully"
