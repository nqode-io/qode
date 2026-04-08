## Test report

### Coverage gate

CI and release workflows enforce a **minimum 70% code coverage** threshold with race detection enabled. Integration tests (build-tagged `integration`) run separately after unit tests.

### Test taxonomy

| Layer | Scope | Examples |
|---|---|---|
| Pure unit | No I/O, no filesystem | Scoring extraction, semver compat, workflow guards, log init, CLI helpers |
| Filesystem unit | Real temp dirs, no network | Config round-trips, branch context load/save, prompt engine, golden files, scaffold generation |
| Integration | Full Cobra command tree, real git repos | End-to-end command execution via `rootCmd.Execute()`, build-tag separated |
| Fuzz | Panic-safety of parsing | Score parser with adversarial seeds derived from unit tests |

### Continue doing

- **Table-driven tests everywhere** -- every function with more than one interesting input uses `[]struct{name string; ...}` with `t.Run`. This is the default shape for new tests.
- **Build-tag separation** -- `//go:build integration` keeps `go test ./...` fast (<2s) while heavier tests remain invocable via `-tags integration`.
- **Golden files with `-update` flag** -- prompt template output is snapshot-tested; run `go test -update` to regenerate. Prevents prompt drift without brittle string assertions.
- **Sentinel / negative-content assertions** -- inject a unique string into context, assert it does NOT appear in rendered output. This tests contracts ("prompts reference files by path, not inline content") cleanly.
- **Real git repos in git package tests** -- the package wraps `git`; mocking it would give false confidence. Temp repos with real commits are the right boundary.
- **`t.Helper()` on every helper** -- all `setupXxx` and `writeFile` helpers call `t.Helper()` so failures point to the call site.
- **`t.Setenv` / `t.Cleanup` for env mutation** -- auto-restored environment prevents test pollution.
- **`t.Parallel()` where safe** -- parallel subtests everywhere except tests that mutate package-level globals (`flagRoot`, logger). Serial tests are serial for a reason.
- **Idempotency tests for setup operations** -- scaffold and init tests run the operation twice and assert identical results.
- **Negative existence assertions** -- assert that legacy files/directories are NOT created. Guards against regressions.
- **Context cancellation tests at multiple layers** -- git, iokit, session, and integration tests all verify cancellation propagation.
- **Fuzz seed derivation from adversarial unit tests** -- the fuzz corpus starts from all known-interesting inputs, not random noise.
- **Functional options for integration test setup** -- `setupProject(t, "branch", withTicket(...), withRefinedAnalysis(...))` is composable and readable.
- **Shared YAML constants for CLI tests** -- a single place for `testYAMLMinimal`, `testYAMLWithStack`, etc. eliminates scattered inline YAML.
- **Errors-as-values** -- tests use `errors.Is(err, ErrNoAnalysis)` rather than string matching. This is idiomatic and refactor-safe.

### Stop doing

- **Mutating package-level `flagRoot` directly in tests** -- this makes tests order-sensitive and race-prone if `t.Parallel()` is ever added. Thread root through function parameters or a session struct instead.
- **Resetting global Cobra command state in cleanup closures** -- manually nilling `rootCmd` children contexts is fragile and will silently break when new commands are added. Tests should create fresh command instances.
- **Near-duplicate test functions for Claude vs Cursor scaffolding** -- the two IDE paths share identical test structures. Use a parameterized helper (`for _, ide := range ideConfigs`) to halve duplication without reducing coverage.
- **Index-based status line assertions with gaps** -- `TestBuildStatusLines_FullyComplete` checks indices 0, 1, 3, 4 but silently skips index 2. Either assert all indices or document why one is excluded.

### Start doing

- **Assert error message content on all error paths** -- `err != nil` alone is insufficient; use `strings.Contains(err.Error(), "expected substring")` to catch regressions in user-facing messages.
- **Consolidate `buildStatusLines` tests into a single table-driven test** -- the three imperative test functions should become named cases in one table, matching the style of `TestRefineStatus_Table` in the same file.
- **Add `t.Parallel()` to pure-function tests that currently lack it** -- status line tests and rubric tests touch no global state and can safely run in parallel.
- **Test idempotency of `runBranchCreate`** -- scaffold tests cover idempotency; branch context creation does not.
- **Test context cancellation for all CLI commands that accept a context** -- currently only `plan refine` has a cancellation test; `plan spec`, `review code`, and `review security` should follow.

### Proposals for adoption (beneficial but not required)

- **Benchmark tests for hot parsing paths** -- `ParseScore` and `extractScore` run on every judge invocation. `BenchmarkParseScore` would catch performance regressions in regex/YAML parsing and establish a baseline.
- **Property-based testing for config normalization** -- the shorthand-to-multi-layer normalization in config has many valid input shapes. A property test asserting `normalize(normalize(x)) == normalize(x)` (idempotency) would cover edge cases table tests miss.
- **Test fixture builder with `testing/fstest.MapFS`** -- for packages that read directory trees (branch context, knowledge), `fstest.MapFS` would eliminate temp-dir boilerplate and make test intent clearer without sacrificing real-filesystem confidence in integration tests.
- **Coverage delta reporting in CI** -- in addition to the absolute 70% gate, report per-PR coverage delta so reviewers see when a change decreases coverage, even if it stays above threshold. Tools like `go-coverage-report` or GitHub Actions annotations can do this with no new dependencies.

### Tests to reconsider

- **`TestLogFunctions_NoPanic`** -- only asserts log functions do not panic, with no assertion on output content or level filtering. Either add output assertions or remove in favor of tests that exercise logging through real call paths.
- **`TestNewEngine` (prompt engine)** -- asserts the constructor returns non-nil and `ProjectName()` echoes the input. This is trivially covered by every other engine test and adds no incremental value.

### Rules for writing new tests

1. Default shape is table-driven with `t.Run(tc.name, ...)` and `t.Parallel()` on both the parent and subtests, unless the test mutates global state
2. Use `t.Helper()` on every helper function, `t.Cleanup` for teardown, `t.TempDir()` for filesystem tests
3. Never mock what you own -- test real implementations; mock only at system boundaries (network, external processes)
4. Golden files for any template or structured output -- always support `-update` flag
5. Sentinel assertions for prompt content -- inject unique strings and assert presence/absence
6. One assertion theme per test function -- if a test name needs "and" you likely need two tests
7. Error paths must assert the error type (`errors.Is`) or message content, not just `err != nil`
8. Integration tests must be behind `//go:build integration` and create fresh command instances, never reset globals
