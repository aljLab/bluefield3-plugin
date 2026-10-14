#!/usr/bin/env bash
set -euo pipefail

# delete old deployment
echo "\n deleting old deployment \n"
kubectl delete -f examples/devel.yaml

# delete old image
echo "\n removing outdated docker image...\n"
docker image rm bluefield3-driver:local

# rebuild image
echo "\n Building docker image with new controller files...\n"
make docker-build IMG="bluefield3-driver:local"

# load image into ctr
echo "\n Importing the docker image to ctr k8s.io namespace... \n"
docker save -o /tmp/driver.tar bluefield3-driver:local
sudo ctr -n k8s.io images import /tmp/driver.tar
rm /tmp/driver.tar

# reapply deployment
echo "\n Reapplying newly built Driver Pod.\n"
kubectl apply -f examples/devel.yaml