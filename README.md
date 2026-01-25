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

### Get all tests:

Notes for the future:
* gorts gets built and put to path - so it can be executed from any directory by using `gorts` command
* the ideal place to collect and execute the tests is to run them from the directory in which they are normally executed, e.g. `<root_of_kuberay_repo>/ray-operator` - the tests' directories will also be simplified

```
./gorts tests --directories ../../rhoai/upstream/kuberay/ray-operator/test/e2e,../../rhoai/upstream/kuberay/ray-operator/test/e2eautoscaler,../../rhoai/upstream/kuberay/ray-operator/test/e2eincrementalupgrade,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjob,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjobsubmitter,../../rhoai/upstream/kuberay/ray-operator/test/e2erayservice,../../rhoai/upstream/kuberay/ray-operator/test/e2eupgrade,../../rhoai/upstream/kuberay/ray-operator/test/sampleyaml --output ~/masters/gorts/.cov/tests.json
```

### Run baseline (wip)
```
./gorts baseline --manifest .cov/tests.json --output .cov/baseline.json \
  --env KUBERAY_TEST_TIMEOUT_SHORT=5m,KUBERAY_TEST_TIMEOUT_MEDIUM=12m,KUBERAY_TEST_TIMEOUT_LONG=15m,KUBERAY_TEST_RAY_IMAGE=rayproject/ray:2.52.1
```

### git flow to be followed
https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow

### experimentation cut-off
first commit used: `d26dbfa9ed1f2f9832981ba7c43304e54c3ee1f1`
last commit used for evaluation of kuberay is: `8fc4e2a0e644db392534927b7c03d15e3ab7bdbc`
