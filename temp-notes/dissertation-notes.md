## Initial implementation notes (after feature_mapping complete)
observations
- usage of environment variables (influenced by kuberay e2e tests CI setup)
- flaky tests handling (retries) - need more on this
- skipping tests (due to complicated setup (kuberay))
- pre/post test hooks (go coverage flushing/ operator restart (kuberay))
- serial test execution (coverage in isolation)
- coverage (https://go.dev/blog/integration-test-coverage) and more coverage (https://pkg.go.dev/cmd/covdata)
- path normalization

## improvements phase
- possible test name collisions (tests appearing twice in mapping/ select)
- allow to check with forked repo changes (for CI/ PRs)
https://stackoverflow.com/questions/20808892/git-diff-between-current-branch-and-master-but-not-including-unmerged-master-com
this is actually not needed and will be skipped - it should work without any modifications for checking current PR against baseline commitSHA
- force retest all (mention Ekstazi and probably fine more (istqb - environment/ impact analysis?)) - there potentially is a value in allowing to run all tests for changes in particular files (safety)
- allow discovery of new tests (NOTE - the discovery relies on baseline directories - this needs to be stated in the final readme)
- better handling of go coverage portability (related to local or containerised deployments and their coverage analysis)
- only handling the tests from original baseline
- an idea - run baseline regeneration nightly so it does not become stale

## safety/ recall
re-read rustyRTS (and/ or similar) with their approach for artificial breaking test changes

## notes to self - function level
* use checksums for functions and the following to get coverage data for functions:
```
go tool covdata func -i=.cov/coverage/test_e2e/TestGcsFaultToleranceAnnotations
```

### go AST references:
- [go/ast package](https://pkg.go.dev/go/ast)
- [go/parser package](https://pkg.go.dev/go/parser)
- [go/printer package](https://pkg.go.dev/go/printer)
- [go/token package](https://pkg.go.dev/go/token)
- [ASTs in Go](https://blog.bradleygore.com/2022/04/18/ast-in-go-p1/)

function level decisions:
* changed function (checksum) - select all tests that cover that function
* new function - ?select all tests covering a file containing the function?

## pre-release testing
mention running all tests in a prod-like environment

## tagging tests
https://mickey.dev/posts/go-build-tags-testing/

## writing unit tests
https://betterstack.com/community/guides/scaling-go/golang-testify/


mention the diff between these two commits:

git diff --name-only 476c9021..b30436ba --

run all might be required for test changes like these

## commits 506bea38..d41d70f3
added new fields to baseline to track the total execution time

## commits b7a69ebf..b80c782c
spotted double output of similar meaning:
```
[Info] Saved test manifest to /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/b7a69ebf_b80c782c/tests.json
[Info] Saved manifest to /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/b7a69ebf_b80c782c/tests.json
```

subsequent fix was added

discovered retry failed tests calculates two attempts - subsequently fixed

## commits 392429f6..646ef143

A path normalisation issue was identified where baseline directories stored with relative paths from the working directory (e.g., ../../repo/test/e2e) did not match git diff output paths (e.g., test/e2e). This caused in-scope test files to be incorrectly reported as out-of-scope. The issue was resolved by implementing suffix-based directory matching, demonstrating the iterative refinement characteristic of DSR.

## commits 9db19f18..b7a69ebf
skipped tests were changed between revisions. The output was misleading:
```
[Info] Saved selection to /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/9db19f18_b7a69ebf/select_file.json
==================================================
Selection Complete!
  From commit:    9db19f180e19
  To commit:      b7a69ebf91c8
  Changed files:  4
  Test files:     1 (in scope, all tests in affected packages selected)
  Selected tests: 0/50 (100.0% reduction)
  Output:         /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/9db19f18_b7a69ebf/select_file.json
==================================================
```

better handling of skipped tests was needed. subsequently fixed. new output is:

```
[Info] Saved selection to /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/9db19f18_b7a69ebf/select_file.json
==================================================
Selection Complete!
  From commit:    9db19f180e19
  To commit:      b7a69ebf91c8
  Changed files:  4
  Test files:     1 (in scope, all tests in affected packages selected)
  [Warn] 1 package(s) with no coverage data (tests were likely skipped during baseline):
         - test/e2eincrementalupgrade
  Selected tests: 0/50 (100.0% reduction)
  Output:         /Users/pawelpaszki/masters/gorts/rq2-experiments/kuberay/.cov/9db19f18_b7a69ebf/select_file.json
==================================================
```


## commits 646ef143..9db19f18
after several attempts of running the baseline an issue was discovered with arm package (kuberay e2e uses amd arch). it was debugged and a temporary fix was applied for subsequent runs:

```
if grep -q 'gevent==24.2.1' test/e2erayservice/rayservice_ha_test.go; then
    echo "gevent fix already applied to rayservice_ha_test.go"
else
    sed -i '' 's/"pip", "install", "locust==2.32.10"/"pip", "install", "gevent==24.2.1", "locust==2.32.10"/g' \
        test/e2erayservice/rayservice_ha_test.go
    echo "Applied gevent fix to rayservice_ha_test.go"
fi
```

observed failure

```
=== RUN   TestAutoscalingRayService
    rayservice_ha_test.go:74: [2026-04-09T18:12:08+01:00] Created ConfigMap test-ns-f2bf6/locust-runner-script successfully
    rayservice_ha_test.go:78: [2026-04-09T18:12:08+01:00] Successfully applied testdata/rayservice.autoscaling.yaml to namespace test-ns-f2bf6
    rayservice_ha_test.go:78: [2026-04-09T18:12:08+01:00] Created RayService test-ns-f2bf6/test-rayservice successfully
    rayservice_ha_test.go:78: [2026-04-09T18:12:08+01:00] Waiting for RayService test-ns-f2bf6/test-rayservice to be ready
    rayservice_ha_test.go:94: [2026-04-09T18:14:04+01:00] Successfully applied testdata/locust-cluster.burst.yaml to namespace test-ns-f2bf6
    rayservice_ha_test.go:97: [2026-04-09T18:14:04+01:00] Created Locust RayCluster test-ns-f2bf6/locust-cluster successfully
    rayservice_ha_test.go:105: [2026-04-09T18:14:25+01:00] Found head pod test-ns-f2bf6/locust-cluster-head-rd8lx
    core.go:89: [2026-04-09T18:14:25+01:00] Executing command: [pip install locust==2.32.10]
    core.go:102: [2026-04-09T18:14:40+01:00] Command stdout: Collecting locust==2.32.10
          Downloading locust-2.32.10-py3-none-any.whl.metadata (9.6 kB)
        Collecting configargparse>=1.5.5 (from locust==2.32.10)
          Downloading configargparse-1.7.5-py3-none-any.whl.metadata (23 kB)
        Collecting flask-cors>=3.0.10 (from locust==2.32.10)
          Downloading flask_cors-6.0.2-py3-none-any.whl.metadata (5.3 kB)
        Collecting flask-login>=0.6.3 (from locust==2.32.10)
          Downloading Flask_Login-0.6.3-py3-none-any.whl.metadata (5.8 kB)
        Collecting flask>=2.0.0 (from locust==2.32.10)
          Downloading flask-3.1.3-py3-none-any.whl.metadata (3.2 kB)
        Collecting gevent>=22.10.2 (from locust==2.32.10)
          Downloading gevent-26.4.0.tar.gz (6.2 MB)
             ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 6.2/6.2 MB 14.8 MB/s eta 0:00:00
          Installing build dependencies: started
          Installing build dependencies: finished with status 'done'
          Getting requirements to build wheel: started
          Getting requirements to build wheel: finished with status 'done'
          Preparing metadata (pyproject.toml): started
          Preparing metadata (pyproject.toml): finished with status 'done'
        Collecting geventhttpclient>=2.3.1 (from locust==2.32.10)
          Downloading geventhttpclient-2.3.9-cp310-cp310-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl.metadata (8.5 kB)
        Requirement already satisfied: msgpack>=1.0.0 in ./anaconda3/lib/python3.10/site-packages (from locust==2.32.10) (1.0.7)
        Requirement already satisfied: psutil>=5.9.1 in ./anaconda3/lib/python3.10/site-packages (from locust==2.32.10) (5.9.6)
        Collecting pyzmq>=25.0.0 (from locust==2.32.10)
          Downloading pyzmq-27.1.0-cp310-cp310-manylinux_2_26_aarch64.manylinux_2_28_aarch64.whl.metadata (6.0 kB)
        Requirement already satisfied: requests>=2.26.0 in ./anaconda3/lib/python3.10/site-packages (from locust==2.32.10) (2.32.3)
        Requirement already satisfied: setuptools>=70.0.0 in ./anaconda3/lib/python3.10/site-packages (from locust==2.32.10) (80.9.0)
        Collecting tomli>=1.1.0 (from locust==2.32.10)
          Downloading tomli-2.4.1-py3-none-any.whl.metadata (10 kB)
        Requirement already satisfied: typing-extensions>=4.6.0 in ./anaconda3/lib/python3.10/site-packages (from locust==2.32.10) (4.12.2)
        Collecting werkzeug>=2.0.0 (from locust==2.32.10)
          Downloading werkzeug-3.1.8-py3-none-any.whl.metadata (4.0 kB)
        Collecting blinker>=1.9.0 (from flask>=2.0.0->locust==2.32.10)
          Downloading blinker-1.9.0-py3-none-any.whl.metadata (1.6 kB)
        Requirement already satisfied: click>=8.1.3 in ./anaconda3/lib/python3.10/site-packages (from flask>=2.0.0->locust==2.32.10) (8.1.7)
        Collecting itsdangerous>=2.2.0 (from flask>=2.0.0->locust==2.32.10)
          Downloading itsdangerous-2.2.0-py3-none-any.whl.metadata (1.9 kB)
        Requirement already satisfied: jinja2>=3.1.2 in ./anaconda3/lib/python3.10/site-packages (from flask>=2.0.0->locust==2.32.10) (3.1.6)
        Requirement already satisfied: markupsafe>=2.1.1 in ./anaconda3/lib/python3.10/site-packages (from flask>=2.0.0->locust==2.32.10) (2.1.3)
        Collecting greenlet>=3.2.2 (from gevent>=22.10.2->locust==2.32.10)
          Using cached greenlet-3.4.0-cp310-cp310-manylinux_2_24_aarch64.manylinux_2_28_aarch64.whl.metadata (3.7 kB)
        Collecting zope.event (from gevent>=22.10.2->locust==2.32.10)
          Downloading zope_event-6.1-py3-none-any.whl.metadata (5.1 kB)
        Collecting zope.interface (from gevent>=22.10.2->locust==2.32.10)
          Downloading zope_interface-8.2-cp310-cp310-manylinux2014_aarch64.manylinux_2_17_aarch64.whl.metadata (45 kB)
        Requirement already satisfied: certifi in ./anaconda3/lib/python3.10/site-packages (from geventhttpclient>=2.3.1->locust==2.32.10) (2025.1.31)
        Requirement already satisfied: brotli in ./anaconda3/lib/python3.10/site-packages (from geventhttpclient>=2.3.1->locust==2.32.10) (1.2.0)
        Requirement already satisfied: urllib3 in ./anaconda3/lib/python3.10/site-packages (from geventhttpclient>=2.3.1->locust==2.32.10) (1.26.19)
        Requirement already satisfied: charset_normalizer<4,>=2 in ./anaconda3/lib/python3.10/site-packages (from requests>=2.26.0->locust==2.32.10) (3.3.2)
        Requirement already satisfied: idna<4,>=2.5 in ./anaconda3/lib/python3.10/site-packages (from requests>=2.26.0->locust==2.32.10) (3.7)
        Downloading locust-2.32.10-py3-none-any.whl (2.4 MB)
           ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 2.4/2.4 MB 11.4 MB/s eta 0:00:00
        Downloading configargparse-1.7.5-py3-none-any.whl (27 kB)
        Downloading flask-3.1.3-py3-none-any.whl (103 kB)
        Downloading flask_cors-6.0.2-py3-none-any.whl (13 kB)
        Downloading Flask_Login-0.6.3-py3-none-any.whl (17 kB)
        Downloading geventhttpclient-2.3.9-cp310-cp310-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl (115 kB)
        Downloading pyzmq-27.1.0-cp310-cp310-manylinux_2_26_aarch64.manylinux_2_28_aarch64.whl (666 kB)
           ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 666.4/666.4 kB 4.6 MB/s eta 0:00:00
        Downloading tomli-2.4.1-py3-none-any.whl (14 kB)
        Downloading werkzeug-3.1.8-py3-none-any.whl (226 kB)
        Downloading blinker-1.9.0-py3-none-any.whl (8.5 kB)
        Using cached greenlet-3.4.0-cp310-cp310-manylinux_2_24_aarch64.manylinux_2_28_aarch64.whl (601 kB)
        Downloading itsdangerous-2.2.0-py3-none-any.whl (16 kB)
        Downloading zope_event-6.1-py3-none-any.whl (6.4 kB)
        Downloading zope_interface-8.2-cp310-cp310-manylinux2014_aarch64.manylinux_2_17_aarch64.whl (255 kB)
        Building wheels for collected packages: gevent
          Building wheel for gevent (pyproject.toml): started
          Building wheel for gevent (pyproject.toml): finished with status 'error'
        Failed to build gevent
    core.go:103: [2026-04-09T18:14:40+01:00] Command stderr:   error: subprocess-exited-with-error
          
          × Building wheel for gevent (pyproject.toml) did not run successfully.
          │ exit code: 1
          ....
          
          note: This error originates from a subprocess, and is likely not a problem with pip.
          ERROR: Failed building wheel for gevent
        ERROR: ERROR: Failed to build installable wheels for some pyproject.toml based projects (gevent)
    core.go:106: 
        	Error Trace:	/Users/pawelpaszki/rhoai/upstream/kuberay/ray-operator/test/support/core.go:106
        	            				/Users/pawelpaszki/rhoai/upstream/kuberay/ray-operator/test/e2erayservice/rayservice_ha_test.go:108
        	Error:      	Received unexpected error:
        	            	command terminated with exit code 1
        	Test:       	TestAutoscalingRayService
        	Messages:   	Command failed unexpectedly
    test.go:114: [2026-04-09T18:14:40+01:00] Retrieving Pod Container test-ns-f2bf6/locust-cluster-head-rd8lx/ray-head logs
    test.go:102: [2026-04-09T18:14:40+01:00] Creating ephemeral output directory as KUBERAY_TEST_OUTPUT_DIR env variable is unset
    test.go:105: [2026-04-09T18:14:40+01:00] Output directory has been created at: /var/folders/7_/d9ngwy550tnd097vfkrqqjrh0000gn/T/TestAutoscalingRayService1581778617/001
    test.go:114: [2026-04-09T18:14:40+01:00] Retrieving Pod Container test-ns-f2bf6/test-rayservice-8dfwm-head-9rssx/ray-head logs
    test.go:114: [2026-04-09T18:14:40+01:00] Retrieving Pod Container test-ns-f2bf6/test-rayservice-8dfwm-head-9rssx/autoscaler logs
    test.go:114: [2026-04-09T18:14:40+01:00] Retrieving Pod Container test-ns-f2bf6/test-rayservice-8dfwm-small-group-worker-kn77x/ray-worker logs
--- FAIL: TestAutoscalingRayService (152.42s)
FAIL
FAIL	github.com/ray-project/kuberay/ray-operator/test/e2erayservice	153.142s
FAIL
```
