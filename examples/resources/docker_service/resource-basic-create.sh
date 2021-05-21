#!/bin/bash
docker service create -d -p 8080 --name foo-service repo.mycompany.com:8080/foo-service:v1