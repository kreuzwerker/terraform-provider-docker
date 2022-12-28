#!/bin/bash
set -e

echo -n "foo" > "$(pwd)/scripts/testing/testingFile"
echo -n `base64 $(pwd)/scripts/testing/testingFile` > "$(pwd)/scripts/testing/testingFile.base64"

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
# pinned to 2.7.0 due to https://github.com/docker/docker.github.io/issues/11060
docker run --rm --entrypoint htpasswd registry:2.7.0 -Bbn testuser testpwd > "$(pwd)"/scripts/testing/auth/htpasswd
docker run -d -p 15000:5000 --rm --name private_registry \
  -v "$(pwd)"/scripts/testing/auth:/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
  -v "$(pwd)"/scripts/testing/certs:/certs \
  -e "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt" \
  -e "REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key" \
  -e "REGISTRY_STORAGE_DELETE_ENABLED=true" \
  registry:2.7.0

docker run -d -p 15001:5000 --rm --name http_private_registry \
  -v "$(pwd)"/scripts/testing/auth:/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
  -e "REGISTRY_STORAGE_DELETE_ENABLED=true" \
  registry:2.7.0

docker run -d -p 15002:5000 --rm --name no_auth_registry \
   -v "$(pwd)"/scripts/testing/certs:/certs \
  -e "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt" \
  -e "REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key" \
  registry:2.7.0

# wait a bit for travis...
sleep 5
# Login to private registry
docker login -u testuser -p testpwd 127.0.0.1:15000
docker login -u testuser -p testpwd 127.0.0.1:15001
# Build private images
for i in $(seq 1 3); do 
  docker build -t tftest-service --build-arg MAIN_FILE_PATH=v${i}/main.go "$(pwd)"/scripts/testing -f "$(pwd)"/scripts/testing/Dockerfile
  docker tag tftest-service 127.0.0.1:15000/tftest-service:v${i}
  docker push 127.0.0.1:15000/tftest-service:v${i}
  docker tag tftest-service 127.0.0.1:15000/tftest-service
  docker push 127.0.0.1:15000/tftest-service

  docker tag tftest-service 127.0.0.1:15001/tftest-service:v${i}
  docker push 127.0.0.1:15001/tftest-service:v${i}
  docker tag tftest-service 127.0.0.1:15001/tftest-service
  docker push 127.0.0.1:15001/tftest-service

  docker tag tftest-service 127.0.0.1:15002/tftest-service:v${i}
  docker push 127.0.0.1:15002/tftest-service:v${i}
  docker tag tftest-service 127.0.0.1:15002/tftest-service
  docker push 127.0.0.1:15002/tftest-service
done
# Remove images from host machine before starting the tests
for i in $(docker images -aq 127.0.0.1:15000/tftest-service); do docker rmi -f "$i"; done
