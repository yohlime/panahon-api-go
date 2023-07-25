#!/bin/bash

BASE_TAG="emiliogozo/panahon-api"
DATE_STR="$(date +%Y%m%d)"

# yarn build

docker build -t ${BASE_TAG} -t ${BASE_TAG}:"${DATE_STR}" .
