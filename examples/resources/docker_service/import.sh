#!/bin/bash

docker service create --name foo -p 8080:80 nginx
4pcphbxkfn2rffhbhe6czytgi

## A Docker service can be imported using the long id, 
## e.g. for a service with the short id `55ba873dd`:

$ terraform import docker_service.foo "$(docker service inspect -f {{.ID}} 55b)"