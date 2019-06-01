#!/bin/bash
set -e

log() {	
  echo "####################"	
  echo "## ->  $1 "	
  echo "####################"	
}

setup() {
  sh "$(pwd)"/scripts/testacc_setup.sh
}

run() {
  go clean -testcache
  TF_ACC=1 go test ./docker -v -timeout 120m
  
  # keep the return value for the scripts to fail and clean properly
  return $?
}

cleanup() {
  sh "$(pwd)"/scripts/testacc_cleanup.sh
}

## main
log "setup" && setup 
log "run" && run || (log "cleanup" && cleanup && exit 1)
log "cleanup" && cleanup
