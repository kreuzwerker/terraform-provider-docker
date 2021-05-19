#!/bin/bash
terraform import docker_plugin.sample-volume-plugin "$(docker plugin inspect -f {{.ID}} tiborvass/sample-volume-plugin:latest)"