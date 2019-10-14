#!/bin/bash

set -eu

# trigger docker hub
curl -X "POST" "https://cloud.docker.com/api/build/v1/source/${DOCKER_HUB_TRIGGER_SOURCE}/trigger/${DOCKER_HUB_TRIGGER_ID}/call/" \
     -sS

