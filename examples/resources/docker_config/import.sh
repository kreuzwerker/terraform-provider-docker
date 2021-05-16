#!/bin/bash
$ terraform import docker_config.foo "$(docker config inspect -f {{.ID}} p73)"