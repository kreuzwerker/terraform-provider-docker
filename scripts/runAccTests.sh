#!/bin/bash
set -e

log() {
  echo "####################"
  echo "## ->  $1 "
  echo "####################"
}

setup() {
  # Create self signed certs
  mkdir -p "$(pwd)"/scripts/testing/certs
  openssl req \
    -newkey rsa:2048 \
    -nodes \
    -x509 \
    -days 365 \
    -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=127.0.0.1" \
    -keyout "$(pwd)"/scripts/testing/certs/registry_auth.key \
    -out "$(pwd)"/scripts/testing/certs/registry_auth.crt
  # Create auth
  mkdir -p "$(pwd)"/scripts/testing/auth
  # Start registry
  docker run --rm --entrypoint htpasswd registry:2 -Bbn testuser testpwd > "$(pwd)"/scripts/testing/auth/htpasswd
  docker run -d -p 15000:5000 --rm --name private_registry \
    -v "$(pwd)"/scripts/testing/auth:/auth \
    -e "REGISTRY_AUTH=htpasswd" \
    -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
    -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
    -v "$(pwd)"/scripts/testing/certs:/certs \
    -e "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt" \
    -e "REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key" \
    registry:2
  # wait a bit for travis...
  sleep 5
  # Login to private registry
  docker login -u testuser -p testpwd 127.0.0.1:15000
  # Build private images
  for i in $(seq 1 3); do 
    docker build -t tftest-service --build-arg JS_FILE_PATH=server_v${i}.js "$(pwd)"/scripts/testing -f "$(pwd)"/scripts/testing/Dockerfile
    docker tag tftest-service 127.0.0.1:15000/tftest-service:v${i}
    docker push 127.0.0.1:15000/tftest-service:v${i}
    docker tag tftest-service 127.0.0.1:15000/tftest-service
    docker push 127.0.0.1:15000/tftest-service
  done
  # Remove images from host machine before starting the tests
  for i in $(docker images -aq 127.0.0.1:15000/tftest-service); do docker rmi -f "$i"; done
}

run() {
  go clean -testcache
  TF_ACC=1 go test ./docker -v -timeout 120m
  
  # for a single test comment the previous line and uncomment the next line
  #TF_LOG=DEBUG TF_ACC=1 go test -v ./docker -run ^TestAccDockerService_updateMultiplePropertiesConverge$ -timeout 360s
  
  # keep the return value for the scripts to fail and clean properly
  return $?
}

cleanup() {
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
