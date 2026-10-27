#!/usr/bin/env bash
set -euo pipefail

# delete old deployment
echo
echo "==> Deleting old deployment"
echo
kubectl delete -f dist/install.yaml --ignore-not-found=true

# delete old operator service accounts
echo
echo "==> Deleting old operator service accounts, roles, bindings"
echo
kubectl delete -f config/rbac/operator-service-account.yaml --ignore-not-found=true
kubectl delete -f config/rbac/operator-role.yaml --ignore-not-found=true
kubectl delete -f config/rbac/operator-role-binding.yaml --ignore-not-found=true

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

# re-make the installer
echo
echo "==> Importing the docker image to ctr k8s.io namespace..."
echo
rm -r dist/
make build-installer


# reapply deployment
echo
echo "==> Reapplying newly built operator Pod."
echo
kubectl apply -f dist/install.yaml