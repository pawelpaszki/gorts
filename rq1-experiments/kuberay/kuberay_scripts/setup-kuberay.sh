#!/bin/bash
if grep -q 'gevent==24.2.1' test/e2erayservice/rayservice_ha_test.go; then
    echo "gevent fix already applied to rayservice_ha_test.go"
else
    sed -i '' 's/"pip", "install", "locust==2.32.10"/"pip", "install", "gevent==24.2.1", "locust==2.32.10"/g' \
        test/e2erayservice/rayservice_ha_test.go
    echo "Applied gevent fix to rayservice_ha_test.go"
fi
# run from kuberay/ray-operator
kind delete clusters --all
kind create cluster --image=kindest/node:v1.26.0
kubectl cluster-info --context kind-kind
# kuberay deployment - kuberay modifications to be documented later
# build images
IMG=kuberay/operator:coverage make docker-build-coverage
IMG=kuberay/submitter:nightly make docker-image-rayjob-submitter
# load images
kind load docker-image kuberay/operator:coverage
kind load docker-image kuberay/submitter:nightly
# deploy operator
IMG=kuberay/operator:coverage make deploy-coverage
kubectl wait --timeout=90s --for=condition=Available deployment -n default kuberay-operator