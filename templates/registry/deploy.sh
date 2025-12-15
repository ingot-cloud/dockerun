#!/usr/bin/env bash


mkdir -p ${WORK_DIR}/volumes/images
mkdir -p ${WORK_DIR}/volumes/auth
mkdir -p ${WORK_DIR}/volumes/config

# 配置htpasswd
# cp ${WORK_DIR}/auth/* ${WORK_DIR}/volumes/auth
cp ${WORK_DIR}/config/config.yml ${WORK_DIR}/volumes/config/config.yml


docker run --name ${CONTEINER_NAME} \
 --network ${DOCKER_NETWORK:-ingot-net} \
 --restart=always \
 --privileged=true \
 -p ${REGISTRY_PORT:-5000}:5000 \
 -v ${WORK_DIR}/volumes/config/config.yml:/etc/docker/registry/config.yml \
 -v ${WORK_DIR}/volumes/images:/var/lib/registry \
 -v ${WORK_DIR}/volumes/auth:/auth \
 -d ${REGISTRY_IMAGE}:${REGISTRY_VERSION}
