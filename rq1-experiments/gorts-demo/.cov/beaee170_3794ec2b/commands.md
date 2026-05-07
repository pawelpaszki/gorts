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
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/tests.json
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
  --manifest ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/tests.json \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/baseline.json \
  --coverage-dir ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/coverage \
  --retry 1 \
  --test-binary ~/masters/gorts-demo/gorts-demo-e2e.test
```

---

## 5 Mapping (repo clean, still at **baseline** commit)

```bash
./gorts mapping \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/baseline.json \
  --module "github.com/pawelpaszki/gorts-demo" \
  --repo ~/masters/gorts-demo \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/mapping.json
```

---

## 6 Checkout next commit (from gorts-demo root)

```bash
git checkout 3794ec2bd500490c263c4194e77240de406c0485
```

## 7 select (from gorts)

```bash
./gorts select \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/baseline.json \
  --mapping ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/mapping.json \
  --repo ~/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/select_file.json
```

```bash
./gorts select \
  --baseline ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/baseline.json \
  --mapping ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/mapping.json \
  --repo ~/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output ~/masters/gorts/rq2-experiments/gorts-demo/.cov/beaee170_3794ec2b/select_func.json
```
