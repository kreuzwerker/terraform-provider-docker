#!/bin/bash
set -e

log() {
  echo ""
  echo "##################################"
  echo "-------> $1"
  echo "##################################"
}

setup() {
  export DOCKER_REGISTRY_ADDRESS="127.0.0.1:15000"
  export DOCKER_REGISTRY_USER="testuser"
  export DOCKER_REGISTRY_PASS="testpwd"
  export DOCKER_PRIVATE_IMAGE="127.0.0.1:15000/tftest-service:v1"
  sh "$(pwd)"/scripts/testing/setup_private_registry.sh
}

run() {
  TF_ACC=1 go test ./docker -v -timeout 120m
  
  # for a single test comment the previous line and uncomment the next line
  #TF_LOG=INFO TF_ACC=1 go test -v github.com/terraform-providers/terraform-provider-docker/docker -run ^TestAccDockerContainer_port$ -timeout 360s
  
  # keep the return value for the scripts to fail and clean properly
  return $?
}

cleanup() {
  unset DOCKER_REGISTRY_ADDRESS DOCKER_REGISTRY_USER DOCKER_REGISTRY_PASS DOCKER_PRIVATE_IMAGE
  echo "### unsetted env ###"
  for p in $(docker container ls -f 'name=private_registry' -q); do docker stop $p; done
  echo "### stopped private registry ###"
  rm -f "$(pwd)"/scripts/testing/auth/htpasswd
  rm -f "$(pwd)"/scripts/testing/certs/registry_auth.*
  echo "### removed auth and certs ###"
  for resource in "container" "volume"; do
    for r in $(docker $resource ls -f 'name=tftest-' -q); do docker $resource rm -f "$r"; done
    echo "### removed $resource ###"
  done
  for resource in "config" "secret" "network"; do
    for r in $(docker $resource ls -f 'name=tftest-' -q); do docker $resource rm "$r"; done
    echo "### removed $resource ###"
  done
  for i in $(docker images -aq 127.0.0.1:15000/tftest-service); do docker rmi -f "$i"; done
  echo "### removed service images ###"
}

## main
log "setup" && setup 
log "run" && run || (log "cleanup" && cleanup && exit 1)
log "cleanup" && cleanup
