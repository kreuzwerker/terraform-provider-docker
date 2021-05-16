#!/bin/bash

# Docker volume can be imported using the long id, 
# e.g. for a volume with the short id `ecae276c5`:

terraform import docker_volume.foo "$(docker volume inspect -f {{.ID}} eca)"