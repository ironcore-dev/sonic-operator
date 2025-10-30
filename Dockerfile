# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.25.3 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the Go source (relies on .dockerignore to filter)
COPY . .

# Build
# the GOARCH has no default value to allow the binary to be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
FROM builder AS manager-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS manager
WORKDIR /
COPY --from=manager-builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

# Build agent binaries
FROM builder AS agent-builder
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o switch-proxy-server cmd/agent/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o switch-proxy-client cmd/agent_cli/main.go

# Stage 2: Final image based on SONiC VS (Arm only right now)
FROM byteocean/docker-sonic-gnmi-vs:arm64-20250818 AS sonic-agent-vs

RUN apt-get update && apt-get install -y \
    sudo \
    systemctl \
    dbus \
    iproute2 \
    net-tools \
    iputils-ping \
    procps \
    && rm -rf /var/lib/apt/lists/*

# Create /opt directory if it doesn't exist
RUN mkdir -p /opt

# Copy the built binaries from builder stage
COPY --from=agent-builder /workspace/switch-proxy-server /opt/
COPY --from=agent-builder /workspace/switch-proxy-client /opt/

# Copy network setup scripts and service configurations
COPY hack/agent/setup-network.sh /opt/
COPY hack/agent/sonic-setup.sh /opt/
COPY hack/agent/sonic-ready-check.sh /opt/
COPY hack/agent/custom-services.conf /etc/supervisor/conf.d/

# Make binaries and scripts executable
RUN chmod +x /opt/switch-proxy-server /opt/switch-proxy-client /opt/setup-network.sh /opt/sonic-setup.sh /opt/sonic-ready-check.sh

# Optional: Add /opt to PATH for easier execution
ENV PATH="/opt:${PATH}"

# Set default environment variables for service configuration
ENV SETUP_NETWORK=true
ENV SWITCH_PROXY_PORT=50052
ENV START_SERVICES=true
ENV TLS_CERT_PATH=/etc/tls/publickey.cer
# Note: TLS_KEY_PATH will be set at runtime for security

# Expose the service ports
EXPOSE 50051 50052

# Set entrypoint
# ENTRYPOINT ["/opt/entrypoint-simple.sh"]

# Default command - when no args provided, start services
# CMD []

# Default command (can be overridden)
# CMD ["/bin/bash"]

# Label the image
LABEL description="SONiC VS with switch-proxy server and client"
LABEL version="test"




