git pull

make manifests
make install

#NODEIP=`grep "${HOSTNAME}" /etc/hosts | cut -d' ' -f1`
NODEIP=50.65.75.178
export METRICS_BIND_ADDRESS=${NODEIP}:8000
export HEALTH_PROBE_BIND_ADDRESS=${NODEIP}:8001
export FLEXLB_TLS_CA_CERT=../certs/ca.crt
export FLEXLB_TLS_CLIENT_CERT=../certs/client.crt
export FLEXLB_TLS_CLIENT_KEY=../certs/client.key
export FLEXLB_REFRESH_INTERVAL=30
export FLEXLB_NAMESPACE=kube-system
export FLEXLB_TRAFFIC_NETWORK=192.168.1.0/24

make run