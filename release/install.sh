#!/bin/bash

function fn_print_help() {
    echo "$(basename $0) [options]
    Options:
        -e ENDPOINT            required, flexlb api endpoint
        -r CA_CERT_FILE        required, root ca cert file
        -c CLIENT_CERT_FILE    required, client cert file
        -k CLIENT_KEY_FILE     required, client key file
        -i INTERFACE           required, flexlb external interface
        -p NET_PREFIX          required, flexlb external network prefix
        -S IP_START            required, flexlb external ip range start ip
        -E IP_END              required, flexlb external ip range end ip
        -N BACKEND_NETWORK     required, flexlb backend network
        -P BACKEND_PREFIX      required, flexlb backend network prefix
        -I PROBE_POD_IMAGE     optional, controller probe pod image, default: busybox
    "
    exit -1
}

function fn_validate_params() {
    while getopts e:r:c:k:i:p:S:E:N:P:I: flag
    do
        case "${flag}" in
            e) ENDPOINT=${OPTARG};;
            r) CA_CERT_FILE=${OPTARG};;
            c) CLIENT_CERT_FILE=${OPTARG};;
            k) CLIENT_KEY_FILE=${OPTARG};;
            i) INTERFACE=${OPTARG};;
            p) NET_PREFIX=${OPTARG};;
            S) IP_START=${OPTARG};;
            E) IP_END=${OPTARG};;
            N) BACKEND_NETWORK=${OPTARG};;
            P) BACKEND_PREFIX=${OPTARG};;
            I) PROBE_POD_IMAGE=${OPTARG};;
            ?) fn_print_help
        esac
    done

    if [ "${ENDPOINT}" == "" ]; then fn_print_help; fi
    if [ "${CA_CERT_FILE}" == "" ]; then fn_print_help; fi
    if [ "${CLIENT_CERT_FILE}" == "" ]; then fn_print_help; fi
    if [ "${CLIENT_KEY_FILE}" == "" ]; then fn_print_help; fi
    if [ "${INTERFACE}" == "" ]; then fn_print_help; fi
    if [ "${NET_PREFIX}" == "" ]; then fn_print_help; fi
    if [ "${IP_START}" == "" ]; then fn_print_help; fi
    if [ "${IP_END}" == "" ]; then fn_print_help; fi
    if [ "${BACKEND_NETWORK}" == "" ]; then fn_print_help; fi
    if [ "${BACKEND_PREFIX}" == "" ]; then fn_print_help; fi
    if [ "${PROBE_POD_IMAGE}" == "" ]; then PROBE_POD_IMAGE="busybox"; fi
}

function fn_main() {
    fn_validate_params $@

    echo "==== install crds"
    kubectl apply -f config/crd/bases
    
    echo "==== install rbac"
    kubectl apply -f config/rbac
    
    
    echo "==== generate certs"
    CA_CERT=$(base64 -w 0 ${CA_CERT_FILE})
    CLIENT_CERT=$(base64 -w 0 ${CLIENT_CERT_FILE})
    CLIENT_KEY=$(base64 -w 0 ${CLIENT_KEY_FILE})
    
    SECRET_FILE=config/controller/flexlb-client-certs.yaml
    sed -i "s/ca.crt:.*$/ca.crt: ${CA_CERT}/" ${SECRET_FILE}
    sed -i "s/client.crt:.*$/client.crt: ${CLIENT_CERT}/" ${SECRET_FILE}
    sed -i "s/client.key:.*$/client.key: ${CLIENT_KEY}/" ${SECRET_FILE}

    echo "==== install controller"
    CONTROLLER_FILE=config/controller/flexlb-kube-controller.yaml
    sed -i "s/probe-pod-image=.*$/probe-pod-image=${PROBE_POD_IMAGE}/" ${CONTROLLER_FILE}
    kubectl apply -f config/controller
    
    echo "==== generate flexlb cluster config"
    CLUSTER_FILE=config/samples/crd_v1_flexlbcluster.yaml
    sed -i "s/endpoint:.*$/endpoint: ${ENDPOINT}/" ${CLUSTER_FILE}
    sed -i "s/interface:.*$/interface: ${INTERFACE}/" ${CLUSTER_FILE}
    sed -i "s/net_prefix:.*$/net_prefix: ${NET_PREFIX}/" ${CLUSTER_FILE}
    sed -i "s/start:.*$/start: ${IP_START}/" ${CLUSTER_FILE}
    sed -i "s/end:.*$/end: ${IP_END}/" ${CLUSTER_FILE}
    sed -i "s/backend_network:.*$/backend_network: ${BACKEND_NETWORK}\/${BACKEND_PREFIX}/" ${CLUSTER_FILE}
    
    echo "==== add cluster config"
    kubectl apply -f config/samples/crd_v1_flexlbcluster.yaml
    
    
    echo "==== install success, next steps:"
    echo "1) create load balancer service"
    echo "2) check instance status: kubectl get flexlbinstance"
}

fn_main $@
