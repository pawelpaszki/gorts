#!/usr/bin/env bash
# Regenerate tests.json, baseline.json, mapping.json, select_*.json (+ *_output.log)
# for every adjacent commit pair from Commit 17 -> Commit 40 on gorts-demo main.
# Instrumented binary: ${DEMO}/gorts-demo-e2e.test (NOT /gorts-demo-e2e.test).
set -euo pipefail

GORTS_ROOT="${GORTS_ROOT:-/Users/pawelpaszki/masters/gorts}"
DEMO="${DEMO:-/Users/pawelpaszki/masters/gorts-demo}"
COV_ROOT="${COV_ROOT:-/Users/pawelpaszki/masters/gorts/rq1-experiments/gorts-demo/.cov}"
GORTS_BIN="$GORTS_ROOT/gorts"
TEST_BIN="$DEMO/gorts-demo-e2e.test"
MODULE="github.com/pawelpaszki/gorts-demo"

# Linear history Commit 17 (inclusive) .. Commit 40 (inclusive)
COMMITS=(
  c0291f34a918dd57f9d58dc216a2858a1881676a
  1fc163ced9d709b56245c5f97e6c2a3405e421df
  67fd722898c0be3769005ab8d7506f9f912bbb40
  1f982418f2d2f8b6489ec3bb00d9be116e5b8208
  8b8b5e253effddd79f8f244bb748fd0802be2994
  9c9aa6a819a2765a51c1ab648970211413785e25
  de2bd093dab85eb6bfda3dbcd10f8545dd70f41a
  c6c460a2911f3c1192c49d4ee279a37dafe06436
  7575ecf5f0b2cc9b536e3a7eb3e73844d864f1f9
  569f4bb183b9e08d8c217a11d8931a59cfa76464
  daf334ac114f3bfe8ee98a10e871e868986ecfd7
  1e1246d38f7205b35d9277c15cf85e60ef068107
  aba6afcacdf09d62cddcf04eda8bd9debcb13939
  beaee1709c419ddfe1215474ca4681f1ee52e779
  3794ec2bd500490c263c4194e77240de406c0485
  dc764e515e6a4dde6acb5c3c356d365b47087de4
  49b5912f3ef136134d0067810cc33bc7602ae794
  9cee31e6f72cb9442df39f80a14d4271fcb0677f
  c01eb940724741d581deceb12fd408348c5cabd4
  389b536a16ca59b5b8c3df3a52b40d61fa7e26ff
  854c42058f0a84e1cf0502c42c397b4d8d5707a1
  b923c3c60a2e1c3ee1dd396f0be9f4174d62a57f
  579a4e856c9aa4752637ffad15b4b7a296927f22
)

pair_dir() {
  local a="$1" b="$2"
  echo "${a:0:8}_${b:0:8}"
}

run_pair() {
  local old_sha="$1" new_sha="$2"
  local dir name
  name="$(pair_dir "$old_sha" "$new_sha")"
  dir="$COV_ROOT/$name"
  mkdir -p "$dir/coverage"

  echo ""
  echo "========== Pair $name =========="
  echo "  baseline (tests/mapping): $old_sha"
  echo "  select checkout:          $new_sha"

  cd "$DEMO"
  git checkout -q "$old_sha"

  echo "[tests]"
  if "$GORTS_BIN" tests \
    --directories "$DEMO/test/e2e" \
    --output "$dir/tests.json" \
    >"$dir/tests_output.log" 2>&1
  then
    echo "  OK tests -> $dir/tests.json"
  else
    echo "  FAIL tests (see $dir/tests_output.log)" >&2
    return 1
  fi

  echo "[build binary]"
  rm -f "$TEST_BIN"
  (
    cd "$DEMO"
    go test -c -cover -covermode=atomic \
      -coverpkg="$MODULE/..." \
      -o "$TEST_BIN" \
      ./test/e2e/...
  ) >"$dir/build_output.log" 2>&1 || { echo "  FAIL build — $dir/build_output.log"; return 1; }

  echo "[baseline]"
  if "$GORTS_BIN" baseline \
    --manifest "$dir/tests.json" \
    --output "$dir/baseline.json" \
    --coverage-dir "$dir/coverage" \
    --retry 1 \
    --test-binary "$TEST_BIN" \
    >"$dir/baseline_output.log" 2>&1
  then
    echo "  OK baseline"
  else
    echo "  FAIL baseline — $dir/baseline_output.log" >&2
    return 1
  fi

  echo "[mapping]"
  if "$GORTS_BIN" mapping \
    --baseline "$dir/baseline.json" \
    --module "$MODULE" \
    --repo "$DEMO" \
    --output "$dir/mapping.json" \
    >"$dir/mapping_output.log" 2>&1
  then
    echo "  OK mapping"
  else
    echo "  FAIL mapping — $dir/mapping_output.log" >&2
    return 1
  fi

  git checkout -q "$new_sha"

  echo "[select file]"
  "$GORTS_BIN" select \
    --baseline "$dir/baseline.json" \
    --mapping "$dir/mapping.json" \
    --repo "$DEMO" \
    --strip-prefix "" \
    --granularity file \
    --output "$dir/select_file.json" \
    >"$dir/select_file_output.log" 2>&1 || return 1

  echo "[select func]"
  "$GORTS_BIN" select \
    --baseline "$dir/baseline.json" \
    --mapping "$dir/mapping.json" \
    --repo "$DEMO" \
    --strip-prefix "" \
    --granularity function \
    --output "$dir/select_func.json" \
    >"$dir/select_func_output.log" 2>&1 || return 1

  echo "  Done pair $name"
}

main() {
  cd "$GORTS_ROOT"
  go build -o "$GORTS_BIN" .

  local i n
  n=$((${#COMMITS[@]} - 1))
  echo "Building gorts OK. Running $n adjacent commit pairs."

  for ((i = 0; i < n; i++)); do
    run_pair "${COMMITS[i]}" "${COMMITS[i + 1]}"
  done

  cd "$DEMO"
  git checkout -q main
  echo ""
  echo "All pairs complete. gorts-demo reset to main."
}

main "$@"
