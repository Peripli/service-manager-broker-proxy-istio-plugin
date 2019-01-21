#!/bin/bash

set -euox pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
GOPATH=${SCRIPT_DIR}/../../../..

cd ${SCRIPT_DIR}
cd $GOPATH/src/github.com/Peripli/service-broker-proxy-k8s

git checkout -- .

# Add ISTIO environment variables to deployment after "key: sm_password"
sed -i -e "/key: sm_password/r $SCRIPT_DIR/env.yml" charts/service-broker-proxy-k8s/templates/deployment.yaml

helm del --purge service-broker-proxy || true
helm install \
    --name service-broker-proxy \
    --namespace service-broker-proxy \
    --set config.sm.url=${SM_URL} \
    --set sm.user=$SM_USER \
    --set sm.password=$SM_PASSWORD \
    --set image.repository=$HUB/sb-istio-proxy-k8s \
    --set image.tag=$TAG \
    --set istio.consumer_id=${ISTIO_CONSUMER_ID} \
    --set istio.service_name_prefix=${ISTIO_SERVICE_NAME_PREFIX:-istio-} \
    --set istio.network_profile=${ISTIO_NETWORK_PROFILE} \
    charts/service-broker-proxy-k8s
