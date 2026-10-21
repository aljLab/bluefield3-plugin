#!/usr/bin/env bash
set -euo pipefail

# delete old deployment
echo
echo "==> Deleting old deployment"
echo
kubectl delete -f examples/devel-operator.yaml --ignore-not-found=true

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

# reapply deployment
echo
echo "==> Reapplying newly built Operator Pod."
echo
kubectl apply -f examples/devel-operator.yaml