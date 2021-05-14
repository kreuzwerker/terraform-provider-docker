#!/bin/bash
terraform import docker_network.foo "$(docker network inspect -f {{.ID}} p73)"