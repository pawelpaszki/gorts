### gorts-demo
first commit: `c0291f34a918dd57f9d58dc216a2858a1881676a`
last commit: `579a4e856c9aa4752637ffad15b4b7a296927f22`

## 1 Build `gorts` if needed from gorts root

```bash
go build -o gorts
```

---

## 2 Enumerate E2E tests from gorts root

```bash
./gorts tests --directories ~/masters/gorts-demo/test/e2e \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/tests.json
```

---

## 3 Build E2E test binary

```bash
go test -c -cover -covermode=atomic \
  -coverpkg=github.com/pawelpaszki/gorts-demo/... \
  -o gorts-demo-e2e.test \
  ./test/e2e/...
```

---

## 4 Baseline (`--test-binary`)

```bash
./gorts baseline \
  --manifest ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/tests.json \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/baseline.json \
  --coverage-dir ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/coverage \
  --retry 1 \
  --test-binary ~/masters/gorts-demo/gorts-demo-e2e.test
```

---

## 5 Mapping (repo clean, still at **baseline** commit)

```bash
./gorts mapping \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/baseline.json \
  --module "github.com/pawelpaszki/gorts-demo" \
  --repo ~/masters/gorts-demo \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/mapping.json
```

---

## 6 Checkout next commit (from gorts-demo root)

```bash
git checkout daf334ac114f3bfe8ee98a10e871e868986ecfd7
```

## 7 select (from gorts)

```bash
./gorts select \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/baseline.json \
  --mapping ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/mapping.json \
  --repo ~/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/select_file.json
```

```bash
./gorts select \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/baseline.json \
  --mapping ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/mapping.json \
  --repo ~/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/569f4bb1_daf334ac/select_func.json
```
