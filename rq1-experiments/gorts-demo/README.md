# gorts-demo experiments

RQ1 evaluation artefacts for [gorts-demo](https://github.com/pawelpaszki/gorts-demo): adjacent commit pairs run through the full GoRTS pipeline (`tests` -> `baseline` -> `mapping` -> `select`).

**Baseline mode:** instrumented test binary (`--test-binary`), module `github.com/pawelpaszki/gorts-demo`.

## Contents

| Path | Description |
|------|-------------|
| `.cov/<old8>_<new8>/` | One folder per commit pair (baseline commit -> next commit) |
| `run_all_pairs.sh` | Regenerates all pair artefacts (override paths via env vars below) |

## Per-pair artefacts (`.cov/<pair>/`)

- `tests.json`, `baseline.json`, `mapping.json`
- `select_file.json`, `select_func.json` — file- vs function-level selection
- `coverage/` — raw per-test Go coverage data
- `*_output.log`, `commands.md` — run logs and command templates

## Commit range

24 commits, 22 adjacent pairs — from `c0291f34` through `579a4e85`.

## Regenerating

```bash
export GORTS_ROOT=/path/to/gorts
export DEMO=/path/to/gorts-demo
export COV_ROOT=/path/to/gorts/rq1-experiments/gorts-demo/.cov
./run_all_pairs.sh
```
