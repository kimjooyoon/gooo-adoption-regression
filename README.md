# gooo-adoption-regression

`gooo-adoption-regression` is a single-record evidence product for observing adoption before and after a change under identical comparison conditions.

It reports exact paired raw integers and semantic equality. It does not infer a generalized result from an integer delta. A pair is eligible only when `scenario_id`, fixture digest, source digest, toolchain digest, evaluator digest, and trial index all match. A semantic `expected`/`actual` exact match is the only route to `NON_REGRESSION_CLOSED`.

The released contract has a fixed denominator of 16 cells: four each for `INSTRUMENT`, `COMPARABILITY`, `SEMANTIC_REGRESSION`, and `PUBLISH`. The canonical corpus contains 12 cases: 3 closed, 4 unknown, and 5 refuted. State precedence is `REFUTED > UNKNOWN > CLOSED`.

All format, build, test, vet, and conformance execution is performed in GitHub Actions. The runner creates two receipts from the same inputs and compares all seven output artifacts byte-for-byte. The final output inventory is:

`comparison-manifest.json`, `before-receipt.json`, `after-receipt.json`, `test-manifest.json`, `semantic-diff.json`, `replay-receipt.json`, and `regression-report.md`.

See [the contract](contracts/adoption-regression-denominator-v1.json), [the source program](examples/adoption-regression.gooo), and [the release notes](docs/release-v0.1.0.md).
