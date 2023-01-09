# remove cluster config
kubectl delete -f config/samples/crd_v1_flexlbcluster.yaml

# uninstall controller
kubectl delete -f config/controller

# uninstall crds
kubectl delete -f config/crd/bases

# uninstall rbac
kubectl delete -f config/rbac

