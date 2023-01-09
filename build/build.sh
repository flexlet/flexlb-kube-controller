#!/bin/bash

BASE_DIR="$(readlink -f $(dirname $0)/..)"
BUILD_DIR="${BASE_DIR}/build"
RELEASE_DIR="${BASE_DIR}/release"
CONFIG_DIR="${BASE_DIR}/config"
IMAGE_REPO="ghcr.io/flexlet"
PROD="flexlb-kube-controller"

# inject build options
source ${BUILD_DIR}/profile

# build target
TARGET_DIR="${BUILD_DIR}/target"
PKG_DIR="${TARGET_DIR}/${PROD}-${VERSION}"

# build binary
cd ${BASE_DIR}
CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -o ${PKG_DIR}/${PROD} main.go

# build container image
cd ${PKG_DIR}
docker build -t ${IMAGE_REPO}/${PROD}:${VERSION} -f ${BUILD_DIR}/Dockerfile
docker push ${IMAGE_REPO}/${PROD}:${VERSION}
rm -rf ${PROD}

# copy certs, install script
cp -r ${RELEASE_DIR}/* ${PKG_DIR}/
cp -r ${CONFIG_DIR} ${PKG_DIR}/

# create tarball
cd ${PKG_DIR}/
tar -zcf ${TARGET_DIR}/${PROD}-${VERSION}.tar.gz *
echo "${TARGET_DIR}/${PROD}-${VERSION}.tar.gz"
