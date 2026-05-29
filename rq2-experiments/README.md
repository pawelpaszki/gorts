# RQ2 experiments
This document describes the setup and results for the dissertation RQ2 experiments (manufactured fault injections against [gorts-demo](https://github.com/pawelpaszki/gorts-demo) repository)

## Setup
This section describes the setup carried out for the RQ2 experiments.

### Starting point
The starting point for the experimentation is the `main` branch of gorts-demo repository at `commit 40` (latest commit into the `main` branch)

```
git rev-parse origin/main
579a4e856c9aa4752637ffad15b4b7a296927f22
```

### gorts setup

#### checkout the main branch
```
git checkout main # from the root of the gorts repo
git pull origin main
```
#### build gorts binary (at revision: c6057f7606f2c3165b0dceec197aedd47c66aaeb)
```
go build -o gorts .
```

### gorts-demo setup

#### checkout the main branch
```
git checkout main # from the root of the gorts repo
git pull origin main
```

#### build gorts-demo e2e test binary with coverage
```
go test -c -cover -covermode=atomic \
  -coverpkg=github.com/pawelpaszki/gorts-demo/... \
  -o gorts-demo-e2e.test \
  ./test/e2e/...
```

### Generate tests, baseline and mapping json structures for the main branch of gorts-demo repo

#### tests.json
```
./gorts tests \
  --directories /Users/pawelpaszki/masters/gorts-demo/test/e2e \
  --output /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/tests.json
```

#### baseline.json
```
./gorts baseline \
  --manifest /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/tests.json \
  --test-binary /Users/pawelpaszki/masters/gorts-demo/gorts-demo-e2e.test \
  --coverage-dir /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/coverage \
  --output /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/baseline.json
```

#### mapping.json
```
./gorts mapping \
  --baseline /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/baseline.json \
  --module github.com/pawelpaszki/gorts-demo \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --output /Users/pawelpaszki/masters/gorts/rq2-experiments/.cov/mapping.json
```

The outputs (json and log) are stored inside `rq2-experiments/.cov`

### Results summary

Baseline manifest: **31** E2E tests. **F** = failing tests; **S** = selected tests. Safety = S (failing) / F (1.0 = safe). Precision = F / S. Details and commands for each scenario are in the sections below.

| Sc | Category | Branch | F | S file | S func | Precision (file) | Precision (func) | Safety |
|----|----------|--------|------|------------|------------|------------------|------------------|--------|
| [1](#scenario-1-single-fault-in-a-single-go-file-a) | single code func (a) | `rq2_scenario_1` | 7 | 9 | 9 | 0.78 | 0.78 | 1.0 |
| [2](#scenario-2-single-fault-in-a-single-go-file-b) | single code func (b) | `rq2_scenario_2` | 4 | 5 | 5 | 0.80 | 0.80 | 1.0 |
| [3](#scenario-3-single-fault-in-a-single-test-file-a) | single test func (a) | `rq2_scenario_3` | 1 | 31 | 31 | 0.03 | 0.03 | 1.0 |
| [4](#scenario-4-single-fault-in-a-single-test-file-b) | single test func (b)  | `rq2_scenario_4` | 1 | 31 | 31 | 0.03 | 0.03 | 1.0 |
| [5](#scenario-5-two-faults-in-a-single-go-file-a) | two code funcs, one file (a) | `rq2_scenario_5` | 4 | 25 | 9 | 0.16 | 0.44 | 1.0 |
| [6](#scenario-6-two-faults-in-a-single-go-file-b) | two code funcs, one file (b) | `rq2_scenario_6` | 4 | 6 | 5 | 0.67 | 0.80 | 1.0 |
| [7](#scenario-7-two-faults-in-a-single-test-file-a) | two test funcs, one file (a) | `rq2_scenario_7` | 2 | 31 | 31 | 0.06 | 0.06 | 1.0 |
| [8](#scenario-8-two-faults-in-a-single-test-file-b) | two test funcs, one file (b) | `rq2_scenario_8` | 2 | 31 | 31 | 0.06 | 0.06 | 1.0 |
| [9](#scenario-9-two-faults-in-a-two-go-files-a) | two code files (a) | `rq2_scenario_9` | 11 | 14 | 14 | 0.79 | 0.79 | 1.0 |
| [10](#scenario-10-two-faults-in-two-single-go-files-b) | two code files (b) | `rq2_scenario_10` | 5 | 11 | 7 | 0.45 | 0.71 | 1.0 |

### Fault injection scenarios
10 different branches (`rq2_scenario_<1-10>`) are created within gorts-demo repository and RQ2 evaluation results are presented below in the scenario-specific sections

---

#### Scenario 1: single fault in a single .go file (a)

Fault injection branch: `rq2_scenario_1`

Change: [internal/model/book.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/model/book.go#L38) 

from `nil` to `return errors.New("rq2-s1: Book.Validate fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-01/select_file.json
```

Selected tests: 9/31 (71.0% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-01/select_func.json
```

Selected tests: 9/31 (71.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Auth_CRUD_WithAuth (0.00s)
--- FAIL: TestE2E_BookCRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_BookCRUD_MultipleBooks (0.00s)
--- FAIL: TestE2E_BookCRUD_UpdateNonExistent (0.00s)
--- FAIL: TestE2E_CreateAndGetBook (0.00s)
--- FAIL: TestE2E_CreateBook_DuplicateISBN (0.00s)
--- FAIL: TestE2E_ReadingList_AddRemoveBooks (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.252s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 7/9 (0.78)   | 7 failing / 9 selected
Safety 7/7 (1)          | 7 selected tests that failed / 7 total failed (safe)

---

#### Scenario 2: single fault in a single .go file (b)

Fault injection branch: `rq2_scenario_2`

Change: [internal/model/author.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/model/author.go#L30) 

from `nil` to `return errors.New("rq2-s2: Author.Validate fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-02/select_file.json
```

Selected tests: 5/31 (83.9% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-02/select_func.json
```

Selected tests: 5/31 (83.9% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )' 
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
--- FAIL: TestE2E_Author_CRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_Author_ListAll (0.00s)
--- FAIL: TestE2E_Author_FilterByCountry (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.490s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 4/5 (0.8)    | 4 failing / 5 selected
Safety 4/4 (1)          | 4 selected tests that failed / 4 total failed (safe)

---

#### Scenario 3: single fault in a single test file (a)
Fault injection branch: `rq2_scenario_3`

Change: [test/e2e/book_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/book_e2e_test.go#L52) 

from `if resp.StatusCode != http.StatusOK {` to `if resp.StatusCode != http.StatusAccepted {`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-03/select_file.json
```

Selected tests: 31/31 (0.0% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-03/select_func.json
```

Selected tests: 31/31 (0.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_CreateAndGetBook (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.524s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 1/31 (0.03)    | 1 failing / 31 selected
Safety 1/1 (1)            | 1 selected tests that failed / 1 total failed (safe)

---

#### Scenario 4: single fault in a single test file (b)
Fault injection branch: `rq2_scenario_4`

Change: [test/e2e/author_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/author_e2e_test.go#L83) 

from `if resp.StatusCode != http.StatusCreated {` to `if resp.StatusCode != http.StatusOK {`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-04/select_file.json
```

Selected tests: 31/31 (0.0% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-04/select_func.json
```

Selected tests: 31/31 (0.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.504s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 1/31 (0.03)    | 1 failing / 31 selected
Safety 1/1 (1)            | 1 selected tests that failed / 1 total failed (safe)

---

#### Scenario 5: two faults in a single .go file (a)
Fault injection branch: `rq2_scenario_5`

Change: [internal/service/book_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/book_service.go#L44) 

from `return nil` to `errors.New("rq2-s5: CreateBook fault")`

and change: [internal/service/book_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/book_service.go#L90)

from `return nil` to `errors.New("rq2-s5: DeleteBook fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-05/select_file.json
```

Selected tests: 25/31 (19.4% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-05/select_func.json
```

Selected tests: 9/31 (71.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Auth_CRUD_WithAuth (0.00s)
--- FAIL: TestE2E_BookCRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_BookCRUD_MultipleBooks (0.00s)
--- FAIL: TestE2E_CreateAndGetBook (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.557s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

File:
Precision: 4/25 (0.16)    | 4 failing / 25 selected
Safety 4/4 (1)            | 4 selected tests that failed / 4 total failed (safe)

Function:
Precision: 4/9 (0.44)     | 4 failing / 9 selected
Safety 4/4 (1)            | 4 selected tests that failed / 4 total failed (safe)

---

#### Scenario 6: two faults in a single .go file (b)
Fault injection branch: `rq2_scenario_6`

Change: [internal/service/author_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/author_service.go#L33) 

from `return s.repo.Create(author)` to `return errors.New("rq2-s6: CreateAuthor fault")`

and change: [internal/service/author_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/author_service.go#L71)

from `return nil` to `return errors.New("rq2-s6: DeleteAuthor fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-06/select_file.json
```

Selected tests: 6/31 (80.6% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-06/select_func.json
```

Selected tests: 5/31 (83.9% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
--- FAIL: TestE2E_Author_CRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_Author_ListAll (0.00s)
--- FAIL: TestE2E_Author_FilterByCountry (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.468s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

File:
Precision: 4/6 (0.67)     | 4 failing / 6 selected
Safety 4/4 (1)            | 4 selected tests that failed / 4 total failed (safe)

Function:
Precision: 4/5 (0.8)      | 4 failing / 5 selected
Safety 4/4 (1)            | 4 selected tests that failed / 4 total failed (safe)

---

#### Scenario 7: two faults in a single test file (a)
Fault injection branch: `rq2_scenario_7`

Change: [test/e2e/author_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/author_e2e_test.go#L94) 

from `if resp.StatusCode != http.StatusOK {` to `if resp.StatusCode != http.StatusAccepted {`

and change: [test/e2e/author_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/author_e2e_test.go#L150) 

from `if resp.StatusCode != http.StatusOK {` to `if resp.StatusCode != http.StatusAccepted {`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-07/select_file.json
```

Selected tests: 31/31 (0.0% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-07/select_func.json
```

Selected tests: 31/31 (0.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
--- FAIL: TestE2E_Author_CRUD_FullLifecycle (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.554s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 2/31 (0.06)    | 2 failing / 31 selected
Safety 2/2 (1)            | 2 selected tests that failed / 2 total failed (safe)

---

#### Scenario 8: two faults in a single test file (b)
Fault injection branch: `rq2_scenario_8`

Change: [test/e2e/book_crud_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/book_crud_e2e_test.go#L59) 

from `if resp.StatusCode != http.StatusOK {` to `if resp.StatusCode != http.StatusAccepted {`

and change: [test/e2e/book_crud_e2e_test.go](https://github.com/pawelpaszki/gorts-demo/blob/main/test/e2e/book_crud_e2e_test.go#L190) 

from `if resp.StatusCode != http.StatusCreated {` to `if resp.StatusCode != http.StatusAccepted {`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-08/select_file.json
```

Selected tests: 31/31 (0.0% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-08/select_func.json
```

Selected tests: 31/31 (0.0% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_BookCRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_BookCRUD_MultipleBooks (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	2.894s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 2/31 (0.06)    | 2 failing / 31 selected
Safety 2/2 (1)            | 2 selected tests that failed / 2 total failed (safe)

---

#### Scenario 9: two faults in a two .go files (a)
Fault injection branch: `rq2_scenario_9`

Change: [internal/model/book.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/model/book.go#L38) 

from `return nil` to `return errors.New("rq2-s9: Book.Validate fault")`

and change: [internal/model/author.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/model/author.go#L30)

from `return nil` to `return errors.New("rq2-s9: Author.Validate fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-09/select_file.json
```

Selected tests: 14/31 (54.8% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-09/select_func.json
```

Selected tests: 14/31 (54.8% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Auth_CRUD_WithAuth (0.00s)
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
--- FAIL: TestE2E_Author_CRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_Author_ListAll (0.00s)
--- FAIL: TestE2E_Author_FilterByCountry (0.00s)
--- FAIL: TestE2E_BookCRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_BookCRUD_MultipleBooks (0.00s)
--- FAIL: TestE2E_BookCRUD_UpdateNonExistent (0.00s)
--- FAIL: TestE2E_CreateAndGetBook (0.00s)
--- FAIL: TestE2E_CreateBook_DuplicateISBN (0.00s)
--- FAIL: TestE2E_ReadingList_AddRemoveBooks (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	0.533s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

Precision: 11/14 (0.79)      | 11 failing / 14 selected
Safety 11/11 (1)             | 11 selected tests that failed / 11 total failed (safe)

---

#### Scenario 10: two faults in two single .go files (b)
Fault injection branch: `rq2_scenario_10`

Change: [internal/service/author_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/author_service.go#L33) 

from `return s.repo.Create(author)` to `return errors.New("rq2-s10: CreateAuthor fault")`

and change: [internal/service/reading_list_service.go](https://github.com/pawelpaszki/gorts-demo/blob/main/internal/service/reading_list_service.go#L79)

from `return nil` to `return errors.New("rq2-s10: DeleteReadingList fault")`

##### select file (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity file \
  --output rq2-experiments/.cov/scenario-10/select_file.json
```

Selected tests: 11/31 (64.5% reduction)

##### select func (from gorts root)
```
./gorts select \
  --baseline rq2-experiments/.cov/baseline.json \
  --mapping rq2-experiments/.cov/mapping.json \
  --repo /Users/pawelpaszki/masters/gorts-demo \
  --strip-prefix "" \
  --granularity function \
  --output rq2-experiments/.cov/scenario-10/select_func.json
```

Selected tests: 7/31 (77.4% reduction)

##### run all and get failed tests names
```
go test -v ./test/e2e/... 2>&1 | grep -E '^(--- FAIL:|FAIL\t|ok  )'
--- FAIL: TestE2E_Author_CreateAndGet (0.00s)
--- FAIL: TestE2E_Author_CRUD_FullLifecycle (0.00s)
--- FAIL: TestE2E_Author_ListAll (0.00s)
--- FAIL: TestE2E_Author_FilterByCountry (0.00s)
--- FAIL: TestE2E_ReadingList_CRUD (0.00s)
FAIL	github.com/pawelpaszki/gorts-demo/test/e2e	1.085s
```

##### Metrics (both function and file)
The failed tests from previous section were visually compared against the selected tests. The metrics were calculated as follows:

File:
Precision: 5/11 (0.45)     | 5 failing / 11 selected
Safety 5/5 (1)             | 5 selected tests that failed / 5 total failed (safe)

Function:
Precision: 5/7 (0.71)      | 5 failing / 7 selected
Safety 5/5 (1)             | 5 selected tests that failed / 5 total failed (safe)
