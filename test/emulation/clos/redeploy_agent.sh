#!/bin/bash
set -x

# ---- CONFIG ----
# Use a consistent container name. 'switch-operator-agent' is used in other scripts like init_setup.sh
CONTAINER_NAME="switch-operator-agent"
IMAGE_NAME="sonic-agent"
IMAGE_TAG="local"
IMAGE_SPEC="${IMAGE_NAME}:${IMAGE_TAG}"

# Use a temporary file for the image archive and ensure it's cleaned up on exit.
IMAGE_ARCHIVE=$(mktemp)
trap 'rm -f -- "$IMAGE_ARCHIVE"' EXIT

PROJECT_ROOT=$(git rev-parse --show-toplevel)

# ---- BUILD IMAGE ----
echo "Building image ${IMAGE_SPEC} from project root ${PROJECT_ROOT}..."
docker build -f "${PROJECT_ROOT}/Dockerfile" \
  --target sonic-agent \
  -t "${IMAGE_SPEC}" \
  "${PROJECT_ROOT}"

echo "Saving image to temporary archive ${IMAGE_ARCHIVE}..."
docker save "${IMAGE_SPEC}" -o "${IMAGE_ARCHIVE}"

# ---- DEPLOY TO VMS ----
echo "Deploying agent to SONiC nodes..."
for service in $(kubectl get -n c9s-clos svc -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n' | grep '^sonic-'); do
  h=$(kubectl get -n c9s-clos svc "$service" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
  if [ -z "$h" ]; then
    continue
  fi

  echo "Deploying to node ${h}..."

  # Copy the image archive
  sshpass -p 'admin' scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "${IMAGE_ARCHIVE}" "admin@${h}:/tmp/image.tar"

  # Execute remote commands to load image and restart container.
  # Pass variables as arguments to the remote script for safety and clarity.
  sshpass -p 'admin' ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "admin@${h}" bash -s -- "${CONTAINER_NAME}" "${IMAGE_SPEC}" <<'EOF'
    set -eux

    REMOTE_CONTAINER_NAME="$1"
    REMOTE_IMAGE_SPEC="$2"

    echo "Loading docker image on remote host..."
    docker load < /tmp/image.tar

    echo "Stopping and removing old container '${REMOTE_CONTAINER_NAME}' if it exists..."
    docker rm -f "${REMOTE_CONTAINER_NAME}" || true

    echo "Starting new container '${REMOTE_CONTAINER_NAME}' with image '${REMOTE_IMAGE_SPEC}'..."
    docker run -d --user 0 \
      --name "${REMOTE_CONTAINER_NAME}" \
      --entrypoint /switch-agent-server \
      --network host \
      --restart unless-stopped \
      -v /var/run/dbus:/var/run/dbus:rw \
      "${REMOTE_IMAGE_SPEC}" \
      -port 57400
EOF
done

echo "Redeployment script finished successfully."
