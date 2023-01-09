# FlexLB kubernetes controller

FlexLB kubernetes controller to add load balancer endpoint for service

## Build

### Clone code

```sh
git clone https://github.com/flexlet/flexlb-kube-controller.git
```

### Build binary

```sh
# edit build/profile to change settings
# build container image
sh build/build.sh
```

### Run

#### Install

```sh
# copy target kubernetes config to ~/.kube/config

# extract package
mkdir flexlb-kube-controller
tar -zxf flexlb-kube-controller-0.4.2.tar.gz -C flexlb-kube-controller

# copy flexlb ca and client certs to the directory
cp ca.crt client.key client.crt flexlb-kube-controller/certs

# install
cd flexlb-kube-controller
sh install.sh
```

#### Run on the fly

```sh
# set parameters
NODEIP=<kube-controller-ip>
export METRICS_BIND_ADDRESS=${NODEIP}:8000
export HEALTH_PROBE_BIND_ADDRESS=${NODEIP}:8001
export FLEXLB_TLS_CA_CERT=../certs/ca.crt
export FLEXLB_TLS_CLIENT_CERT=../certs/client.crt
export FLEXLB_TLS_CLIENT_KEY=../certs/client.key
export FLEXLB_REFRESH_INTERVAL=30
export FLEXLB_NAMESPACE=kube-system
export FLEXLB_TRAFFIC_NETWORK=192.168.1.0/24

# run on the fly
make run
```

#### Test

```sh
# edit config/samples/crd_v1_flexlbcluster.yaml, change flex-api endpoint
# create cluster config
kubect apply -f config/samples/crd_v1_flexlbcluster.yaml

# create instance manually
kubectl apply -f config/samples/crd_v1_flexlbinstance.yaml

# create a load-balancer service and test it

```
