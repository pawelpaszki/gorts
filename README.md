# WiP README for now

## kuberay deployment steps
```
# create kind cluster and set kubectl context
# docker setup skipped for now
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
```

## Raw notes - README to be revised later in the development stage

### experimentation cut-off
last commit used for evaluation of kuberay is: `8fc4e2a0e644db392534927b7c03d15e3ab7bdbc`
