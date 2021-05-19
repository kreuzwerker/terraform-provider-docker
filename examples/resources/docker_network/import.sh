#!/bin/bash
docker network create foo
# prints the long ID
87b57a9b91ecab2db2a6dbf38df74c67d7c7108cbe479d6576574ec2cd8c2d73

$ terraform import docker_network.foo 87b57a9b91ecab2db2a6dbf38df74c67d7c7108cbe479d6576574ec2cd8c2d73
# or use the short version to retrieve the long ID
$ terraform import docker_network.foo "$(docker network inspect -f {{.ID}} 87b)"