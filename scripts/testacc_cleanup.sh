#!/bin/bash
set -e
  
for p in $(docker container ls -f 'name=private_registry' -q); do docker stop $p; done
for p in $(docker container ls -f 'name=no_auth_registry' -q); do docker stop $p; done
echo "### stopped private registry ###"

rm -f "$(pwd)/scripts/testing/testingFile"
rm -f "$(pwd)/scripts/testing/testingFile.base64"
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
