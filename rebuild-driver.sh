# delete old image
echo
echo "==> deleting old driver image..."
echo
docker image rm bluefield3-driver:local || true

# rebuild image
echo
echo "==> Building docker image with new controller files..."
echo
make docker-build-driver IMG_DRIVER="bluefield3-driver:local"

# load image into ctr
echo
echo "==> Importing the docker image to ctr k8s.io namespace..."
echo
docker save -o /tmp/driver.tar bluefield3-driver:local
sudo ctr -n k8s.io images import /tmp/driver.tar
rm /tmp/driver.tar
