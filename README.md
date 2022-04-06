# FlexLB kubernetes controller

FlexLB kubernetes controller to add load balancer endpoint for service

## Build

### Clone code

```sh
git clone https://gitee.com/flexlb/flexlb-kube-controller.git
```

### Build binary

#### For Linux
```sh
# build binary
make

# build docker
docker build -t flexlb-kube-controller:latest .

# push docker
docker push flexlb-kube-controller:latest
```

### Run

#### Install CRDs

```sh
# copy target kubernetes config to ~/.kube/config

# install CRDs
kubectl apply -f config/crd/bases
```

#### Deploy

```sh
# install rbac
kubectl apply -f config/rbac

# edit config/controller/flexlb-client-certs.yaml, change to target flexlb-api client certificate
base64 -w 0 ../certs/ca.crt
base64 -w 0 ../certs/client.crt
base64 -w 0 ../certs/client.key

# install controller
kubectl apply -f config/controller
```