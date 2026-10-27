#!/usr/bin/env bash
set -euo pipefail

# delete old image
echo
echo "==> removing outdated docker image..."
echo
docker image rm bluefield3-operator:local || true

# rebuild image
echo
echo "==> Building docker image with new controller files..."
echo
make docker-build-operator IMG_OPERATOR="bluefield3-operator:local"

# load image into ctr
echo
echo "==> Importing the docker image to ctr k8s.io namespace..."
echo
docker save -o /tmp/operator.tar bluefield3-operator:local
sudo ctr -n k8s.io images import /tmp/operator.tar
rm /tmp/operator.tar
