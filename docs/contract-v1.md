# Contract v1

The denominator is fixed at 16 cells, grouped in four groups of four. Each source activity binds to exactly one contract cell through the compiler's semantic IR and generated comparator.

The six pair identity fields are `scenario_id`, `fixture_digest`, `source_digest`, `toolchain_digest`, `evaluator_digest`, and `trial_index`. A before/after pair exists only when both receipts are present and every field is exactly equal. Pair absence or identity mismatch is `UNKNOWN`.

`NON_REGRESSION_CLOSED` requires exact equality of the semantic `expected` and `actual` arrays, a comparable pair, explicit cache state, test-manifest coverage, and no hidden skipped test. Semantic counterexamples and declared identity/evaluator faults are `REFUTED`. State precedence is `REFUTED > UNKNOWN > CLOSED`.

The only numeric comparison is a raw integer `OBSERVED_DELTA = after - before` for each preserved metric. The receipt retains the unmodified before and after integers and cache boolean. CI also records run/job IDs, build/test wall time, peak RSS, test discovery/execution/skip/cache counts, repository inventory lines, and output artifact count.
