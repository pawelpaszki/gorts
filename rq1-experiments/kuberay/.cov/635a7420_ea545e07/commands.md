### kuberay commands
To be executed from <kuberay_repo>/ray-operator
```
./patch-deployment.sh
./setup.sh
```

To be executed from gorts root:
```
go build -o gorts # build the binary
```

## tests command
```
./gorts tests --directories ../../rhoai/upstream/kuberay/ray-operator/test/e2e,../../rhoai/upstream/kuberay/ray-operator/test/e2eautoscaler,../../rhoai/upstream/kuberay/ray-operator/test/e2eincrementalupgrade,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjob,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjobsubmitter,../../rhoai/upstream/kuberay/ray-operator/test/e2erayservice,../../rhoai/upstream/kuberay/ray-operator/test/e2eupgrade,../../rhoai/upstream/kuberay/ray-operator/test/sampleyaml --output ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/tests.json
```

## baseline command
```
./gorts baseline \
  --manifest ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/tests.json \
  --output ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/baseline.json \
  --coverage-dir ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/coverage \
  --retry 1 \
  --skip TestZeroDowntimeUpgradeAfterOperatorUpgrade --skip TestRayServiceIncrementalUpgradeRollback \
  --skip TestRayServiceIncrementalUpgrade --skip TestRayServiceIncrementalUpgradeWithLocust \
  --skip TestDeletionStrategy \
  --skip TestRayClusterUpgradeStrategy \
  --pre-test "/Users/pawelpaszki/masters/gorts/scripts/cleanup_namespace.sh && kubectl exec -n default \$(kubectl get pod -n default -l app.kubernetes.io/name=kuberay -o jsonpath='{.items[0].metadata.name}') -- sh -c 'rm -rf /coverage/*'" \
  --post-test "kubectl rollout restart deployment/kuberay-operator -n default && kubectl rollout status deployment/kuberay-operator -n default --timeout=120s && kubectl cp default/\$(kubectl get pod -n default -l app.kubernetes.io/name=kuberay -o jsonpath='{.items[0].metadata.name}'):/coverage/. {{COVERAGE_PATH}}" \
  --env KUBERAY_TEST_TIMEOUT_SHORT=5m,KUBERAY_TEST_TIMEOUT_MEDIUM=12m,KUBERAY_TEST_TIMEOUT_LONG=15m,KUBERAY_TEST_RAY_IMAGE=rayproject/ray:2.52.1
```

## mapping command
```
./gorts mapping \
  --baseline ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/baseline.json \
  --module "github.com/ray-project/kuberay/ray-operator" \
  --repo ~/rhoai/upstream/kuberay/ray-operator \
  --output ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/mapping.json
```

## select command (previous commit +1)
To be executed from <kuberay_repo>
```
git stash --include-untracked # uncommitted setup files
git checkout ea545e079343ca6b4d1595923fcbd7c24007e79a
```

to be executed from gorts root

### file level
```
./gorts select \
  --baseline ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/baseline.json \
  --mapping ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/mapping.json \
  --repo ~/rhoai/upstream/kuberay/ray-operator \
  --strip-prefix ray-operator/ \
  --granularity file \
  --output ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/select_file.json
```

### func level
```
./gorts select \
  --baseline ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/baseline.json \
  --mapping ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/mapping.json \
  --repo ~/rhoai/upstream/kuberay/ray-operator \
  --strip-prefix ray-operator/ \
  --granularity function \
  --output ~/masters/gorts/rq1-experiments/kuberay/.cov/635a7420_ea545e07/select_func.json
```



##############

./gorts tests --directories ../../rhoai/upstream/kuberay/ray-operator/test/e2e,../../rhoai/upstream/kuberay/ray-operator/test/e2eautoscaler,../../rhoai/upstream/kuberay/ray-operator/test/e2eincrementalupgrade,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjob,../../rhoai/upstream/kuberay/ray-operator/test/e2erayjobsubmitter,../../rhoai/upstream/kuberay/ray-operator/test/e2erayservice,../../rhoai/upstream/kuberay/ray-operator/test/e2eupgrade,../../rhoai/upstream/kuberay/ray-operator/test/sampleyaml --output ~/masters/gorts/rq1-experiments/kuberay/.cov/ea545e07_88045fe2/tests.json