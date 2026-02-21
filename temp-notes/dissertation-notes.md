## Initial implementation notes (after feature_mapping complete)
observations
- usage of environment variables (influenced by kuberay e2e tests CI setup)
- flaky tests handling (retries) - need more on this
- skipping tests (due to complicated setup (kuberay))
- pre/post test hooks (go coverage flushing/ operator restart (kuberay))
- serial test execution (coverage in isolation)
- coverage (https://go.dev/blog/integration-test-coverage)
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