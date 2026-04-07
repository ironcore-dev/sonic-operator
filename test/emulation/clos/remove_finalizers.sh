#!/bin/bash

set -x
# odstránenie finalizerov
kubectl get switchinterface -o json | \
jq '.items[] | .metadata.finalizers = [] | .metadata.name' \
| xargs -I {} kubectl patch switchinterface {} --type merge -p '{"metadata":{"finalizers":[]}}'

# delete všetkých CR
kubectl delete switchinterface --all

for name in $(kubectl get switch -o jsonpath='{.items[*].metadata.name}'); do
  kubectl patch switch "$name" \
    --subresource=status \
    --type merge \
    -p '{"status":{"state":"Pending"}}'
done
