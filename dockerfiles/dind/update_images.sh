#!/bin/bash

FREECC_DOCKER_REPO="freecompilercamp/pwc"

# List all the docker image tags
FREECC_DOCKER_IMAGES=" \
    16.04 \
    18.04 \
    full \
    llvm10 \
    llvm10-gpu \
    rose-bug \
    rose-debug \
    rose-develop-debug-weekly \
    rose-develop-weekly \
    rose-exam \
    rose-release-weekly \
    "

# Iterate the string variable using for loop
for tag in ${FREECC_DOCKER_IMAGES}; do
    #echo ${FREECC_DOCKER_REPO}:$tag
    docker pull ${FREECC_DOCKER_REPO}:$tag
done

docker image prune -f
