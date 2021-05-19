#!/bin/bash
docker run --name foo -p8080:80 -d nginx 
# prints the container ID 
9a550c0f0163d39d77222d3efd58701b625d47676c25c686c95b5b92d1cba6fd

$ terraform import docker_container.foo 9a550c0f0163d39d77222d3efd58701b625d47676c25c686c95b5b92d1cba6fd
# or use the name to retrieve the container ID
$ terraform import docker_container.foo "$(docker inspect -f {.ID}} foo)"