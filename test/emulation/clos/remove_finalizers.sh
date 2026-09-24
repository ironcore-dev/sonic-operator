#!/bin/bash

set -x
# odstránenie finalizerov
kubectl get cell -o json | \
jq '.items[] | .metadata.finalizers = [] | .metadata.name' \
| xargs -I {} kubectl patch cell {} --type merge -p '{"metadata":{"finalizers":[]}}'

# delete všetkých CR
kubectl delete cell --all

#for name in $(kubectl get switch -o jsonpath='{.items[*].metadata.name}'); do
#  kubectl patch switch "$name" \
#    --subresource=status \
#    --type merge \
#    -p '{"status":{"Phase":"Pending"}}'
#done
