#!/bin/bash

# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0

set -eux

echo "Starting destruction of SONiC lab infrastructure..."

# Delete the c9s-clos namespace (contains all topology resources)
echo "Deleting c9s-clos namespace..."
kubectl delete namespace c9s-clos --ignore-not-found=true
sleep 10

# Delete the c9s namespace (contains clabernetes)
echo "Deleting c9s namespace..."
kubectl delete namespace c9s --ignore-not-found=true
sleep 10

# Remove kube-vip configmap
echo "Removing kube-vip configmap..."
kubectl delete configmap -n kube-system kubevip --ignore-not-found=true

# Remove kube-vip daemonset
echo "Removing kube-vip daemonset..."
kubectl delete daemonset -n kube-system kube-vip-ds --ignore-not-found=true

# Remove kube-vip cloud controller
echo "Removing kube-vip cloud controller deployment..."
kubectl delete deployment -n kube-vip kube-vip-cloud-provider --ignore-not-found=true

# Remove kube-vip namespace if empty
echo "Cleaning up kube-vip namespace..."
kubectl delete namespace kube-vip --ignore-not-found=true

# Remove RBAC resources for kube-vip
echo "Removing kube-vip RBAC resources..."
kubectl delete clusterrole system:kube-vip-role --ignore-not-found=true
kubectl delete clusterrole system:kube-vip-cloud-controller-role --ignore-not-found=true
kubectl delete clusterrolebinding system:kube-vip-binding --ignore-not-found=true
kubectl delete clusterrolebinding system:kube-vip-cloud-controller-binding --ignore-not-found=true
kubectl delete serviceaccount -n kube-system kube-vip --ignore-not-found=true
kubectl delete serviceaccount -n kube-vip kube-vip-cloud-controller --ignore-not-found=true

echo "Destruction complete!"
echo "All SONiC lab resources have been removed."
