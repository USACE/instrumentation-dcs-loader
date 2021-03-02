#!/bin/bash

# NOTE: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_DEFAULT_REGION
#       are typically injected by IAM Role. Passed here explicitly for local testing
#       but probably do not need to be passed explicitly when running in AWS

docker run --rm \
  --network="instrumentation-api_default" \
  -v "$(PWD)/bin":/var/task:ro,delegated \
  -e "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE" \
  -e "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" \
  -e "AWS_DEFAULT_REGION=us-east-1" \
  -e "LOADER_POST_URL=http://instrumentation-api_api_1/instrumentation/timeseries_measurements" \
  -e "LOADER_API_KEY=appkey" \
  -e "LOADER_AWS_S3_REGION=us-east-1" \
  -e "LOADER_AWS_S3_ENDPOINT=http://minio:9000" \
  -e "LOADER_AWS_S3_DISABLE_SSL=True" \
  -e "LOADER_AWS_S3_FORCE_PATH_STYLE=True" \
  lambci/lambda:go1.x instrumentation-dcs-loader "$(cat ./test-event.json)"
