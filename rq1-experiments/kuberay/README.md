# KubeRay experiments

RQ1 evaluation artefacts for [KubeRay ray-operator](https://github.com/ray-project/kuberay): adjacent commit pairs run through the full GoRTS pipeline (`tests` -> `baseline` -> `mapping` -> `select`).

**Baseline mode:** hook-based coverage (`--pre-test` / `--post-test`, `kubectl cp` from operator pod), module `github.com/ray-project/kuberay/ray-operator`.

## Contents

| Path | Description |
|------|-------------|
| `.cov/<old>_<new>/` | One folder per commit pair (baseline commit -> next commit) |
| `kuberay_scripts/` | Cluster setup helpers (`setup-kuberay.sh`, `patch-deployment.sh`, `cleanup_namespace.sh`) |
| `init_baseline_results/` | Early baseline JSON snapshots from initial runs |

## Per-pair artefacts (`.cov/<pair>/`)

- `tests.json`, `baseline.json`, `mapping.json`
- `select_file.json`, `select_func.json` — file- vs function-level selection
- `coverage/` — raw per-test Go coverage data
- `*_output.log`, `commands.md` — run logs and command templates used for that pair

## Commit range

25 commits, 24 adjacent pairs — from `fea763aa` through `3636b4e1`.

## Environment notes

Runs used a local Kubernetes cluster (e.g. Colima/kind) with a coverage-instrumented KubeRay operator deployment. Paths in `commands.md` and JSON files reflect the machine where experiments were executed; adjust when re-running locally.
