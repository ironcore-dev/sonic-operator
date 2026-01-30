# Getting started

## Prerequisites
- go version v1.24.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

## Build and deploy
1. Build and push the image:

```sh
make docker-build docker-push IMG=<some-registry>/sonic-operator:tag
```

2. Install CRDs:

```sh
make install
```

3. Deploy the controller:

```sh
make deploy IMG=<some-registry>/sonic-operator:tag
```

## Create resources
Apply sample manifests and then edit them for your environment:

```sh
kubectl apply -k config/samples/
```

## Uninstall
```sh
kubectl delete -k config/samples/
make uninstall
make undeploy
```
