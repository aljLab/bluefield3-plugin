#!/usr/bin/env bash
set -euo pipefail

# update repo
git pull

# delete old deployment
kubectl delete -f examples/devel.yaml

# delete old image
docker image rm bluefield3-driver:local

# rebuild image
make docker-build IMG="bluefield3-driver:local"

# load image into ctr
docker save -o /tmp/driver.tar bluefield3-driver:local
sudo ctr -n k8s.io images import /tmp/driver.tar
rm /tmp/driver.tar

# reapply deployment
kubectl apply -f examples/devel.yaml