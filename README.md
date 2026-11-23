# bluefield3-plugin
The Bluefield 3 plugin enables the integration of the BlueField 3 DPU into IML/Loom as a programmable target. It can be installed via Helm. The BlueField 3 is P4-programmable through the DOCA pipeline language (DPL). In order to use the BlueField in Loom, the user has to install and configure the BlueField to ensure reachability over the P4Runtime-compatible gRPC interface via the out-of-band interface.

## Description
The plugin contains an operator and a driver. The operator is deployed on installation and watches the creation of Bluefil
eld3Targets. If a target instance is created, the operator creates a corresponding `P4Target` instance and deploys a driver pod.
The driver pod connects to the BlueField over the OOB-interface and manages the lifecycle of network functions on the DPL pipeline.
To install the plugin, the docker images need to be built first using `make docker-build-operator` and `make docker-build-driver`.

## Getting Started

### BlueField 3 Configuration
To be able to configure the BlueField 3, it needs to be reachable over the Out-of-Band interface. The `tmfifo_net0` interface should be up to install software on the BlueField over `rshim`.

#### Install DOCA on Host and BlueField
- Install DOCA on the host (for DOCA 3.4.0 follow the instructions [here](https://networking-docs.nvidia.com/doca/archive/3-4-0/doca-host-installation-and-upgrade))
- Uprade the BlueField Bundle (DOCA + Firmware) on the BlueField to match the host version. For DOCA 3.4.0 follow the instrcutions [here](https://networking-docs.nvidia.com/doca/archive/3-4-0/bf-bundle-installation-and-upgrade)

#### Deploy DPL container
Deploy the DPL runtime service on the ARM subsystem: [instructions](https://networking-docs.nvidia.com/doca/archive/3-4-0/dpl-container-deployment).
Configure DPL runtime service to your needs:
[instructions](https://networking-docs.nvidia.com/doca/archive/3-4-0/dpl-service-configuration).

#### Fetch DPL Development container
Find instructions to download the DPL development container [here](https://networking-docs.nvidia.com/doca/archive/3-4-0/dpl-development-container).
#### (Optional) Create scalable functions
One can create scalable functions (similar to virtual functions, but configured on the BlueField instead of on the host).
To create a scalable function on the first physical function with scalable function number 10, which is managed on the remote host, use:
```
sudo /opt/mellanox/iproute2/sbin/mlxdevm port add pci/0000:03:00.0 flavour pcisf pfnum 0 sfnum 10 controller 1

# activate (find the pci addr with `sudo /opt/mellanox/iproute2/sbin/mlxdevm port show`)
/opt/mellanox/iproute2/sbin/mlxdevm port function set pci/0000:03:00.0/<id> \
    hw_addr <choose-a-free-mac> \
    trust on \
    state active
```

For detailed instructions, refer to [here](https://networking-docs.nvidia.com/doca/archive/3-4-0/bluefield-scalable-functions#Create-and-Deploy-an-SF).


### Prerequisites
- go version v1.24.6+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- configured DPL pipeline running on the BlueField's ARM subsystem
- reachable DPL runtime dameon P4Runtime-compatible gRPC server
- IML installed into the K8s cluster (see [here](https://iml.readthedocs.io/en/latest/getting-started/installation/) for installation details)

### To Deploy on the cluster
**Build and push the operator and driver images locally or push to a registry:**
```
# OPERATOR
make docker-build-operator IMG_OPERATOR="<IMG>:<TAG>"

# for building locally
make docker-build-operator IMG_OPERATOR="bluefield3-operator:local"

# DRIVER
make docker-build-driver IMG_DRIVER="<IMG>:<TAG>"

# for building locally
make docker-build-driver IMG_DRIVER="bluefield3-driver:local"
```
> The images might need to be imported into the container runtime manually.
> Try docker saving the docker image and importing it to the ctr like so `sudo ctr -n k8s.io images import /tmp/your-saved-img.tar`

**Install via Helm**
To install the CRD and deploy the operator, install the Helm chart provided in `./deploy/chart`
```
helm install <release> ./deploy/chart  \
    --namespace <NS>-system --create-namespace \
    --set manager.image.repository=<IMG> \
    --set manager.image.tag=<TAG> \
    --set manager.image.pullPolicy=IfNotPresent
```
For locally built images:
```
helm install release-01 ./deploy/chart  \
    --namespace bf3-system --create-namespace \
    --set manager.image.repository=bluefield3-operator \
    --set manager.image.tag=local \
    --set manager.image.pullPolicy=IfNotPresent
```

### To Uninstall
Uninstall the release via Helm:
`helm uninstall <release> -n <NS>-system`

## Usage
### Adding a `Bluefield3Target`
Run `kubectl apply -f ./examples/simple/bf3target.yaml` to create a dynamically proxied BlueField3Target.
Verify with `kubectl describe p4target bf3-target`. If nothing comes up, set the correct interfaces towards the BlueField and ensure that the nodeName in the manifest (in `./examples/simple/bf3target.yaml`) matches the hostname of the host the BlueField is installed on.
### Deploying a network function
To deploy a network function on the BlueField, run `kubectl apply -f examples/simple/simple-nf.yaml`. The network function forwards all traffic and counts packets and bytes traversing the forwarding table.
### Integrating the network function into a service chain
Run `kubectl apply -f examples/simple/simple-apps.yaml`, then `kubectl apply -f examples/simple/simple-chain.yaml` to create a web server (nginx) and a web client and a service chain Client -> Server connecting the two applications.
### Verifying traffic flow
Find the IP address of the web server (e.g. by using `kubectl get pods` and `kubectl describe pod <server-pod-name>` and finding the IP in the 10.123.0.0./16 subnet) and execute a shell in the client pod (`kubectl exec -it <client-pod-name> -- sh`). Now you can `ping` the server or `wget` the nginx default page using the server's IP.
Verify that traffic traverses the network function by using the DPL development container: `dpl nspect -a <OOB-IP> externs`, which prints the counter values in the terminal.
Alternatvely, use the metrics server exposed by the driver. To this end, get the driver IP from the P4target resource (`kubectl describe p4target bf3-target`) and retrieve the counters from there (`curl <DRIVER-IP>:8081/api/counters`). 
### Removing examples
To remove the deployed examples, run `kubectl delete -f examples/simple/simple-chain.yml`, `kubectl delete -f examples/simple/simple-apps.yml`, `kubectl delete -f examples/simple/simple-nf.yml` and `kubectl delete -f examples/simple/bf3target.yml`
## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

