#!/bin/bash

BASE_TAG="emiliogozo/panahon-api"
DATE_STR="$(date +%Y%m%d)"

podman build -t ${BASE_TAG} -t ${BASE_TAG}:"${DATE_STR}" .
