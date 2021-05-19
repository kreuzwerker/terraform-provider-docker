#!/bin/bash
printf '{"a":"b"}' | docker config create foo -
# prints the the long ID 
08c26c477474478d971139f750984775a7f019dbe8a2e7f09d66a187c009e66d

$ terraform import docker_config.foo 08c26c477474478d971139f750984775a7f019dbe8a2e7f09d66a187c009e66d
# or use the short version to retrieve the long ID
$ terraform import docker_config.foo "$(docker config inspect -f {{.ID}} 08c)"