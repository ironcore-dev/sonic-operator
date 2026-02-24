# Install with Kustomize

## Build and push image

```sh
make docker-build docker-push IMG=<some-registry>/sonic-operator:tag
```

## Install CRDs and deploy

```sh
make install
make deploy IMG=<some-registry>/sonic-operator:tag
```
