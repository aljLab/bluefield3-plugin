#!/usr/bin/env bash
set -euo pipefail

# delete old deployment
echo
echo "Deleting old deployment"
echo
kubectl delete -f examples/devel.yaml

# delete old image
echo
echo "removing outdated docker image..."
echo
docker image rm bluefield3-driver:local

# rebuild image
echo
echo "Building docker image with new controller files..."
echo
make docker-build IMG="bluefield3-driver:local"

# load image into ctr
echo
echo "Importing the docker image to ctr k8s.io namespace..."
echo
docker save -o /tmp/driver.tar bluefield3-driver:local
sudo ctr -n k8s.io images import /tmp/driver.tar
rm /tmp/driver.tar

# reapply deployment
echo "\n Reapplying newly built Driver Pod.\n"
kubectl apply -f examples/devel.yaml