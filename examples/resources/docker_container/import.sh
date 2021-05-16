#!/bin/bash
$ terraform import docker_container.foo "$(docker inspect -f {.ID}} foo)"