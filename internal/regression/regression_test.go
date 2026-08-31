package regression

import "testing"

func boolPointer(value bool) *bool { return &value }

func receiptForTest() *Receipt {
	return &Receipt{
		ScenarioID: "scenario/test",
		FixtureDigest: "fixture",
		SourceDigest: "source",
		ToolchainDigest: "toolchain",
		EvaluatorDigest: "evaluator",
		TrialIndex: 1,
		Metrics: Metrics{CacheHit: boolPointer(false)},
	}
}

func TestExactPairUsesAllSixIdentityFields(t *testing.T) {
	before := receiptForTest()
	after := receiptForTest()
	if !exactPair(before, after) {
		t.Fatal("identical receipts must form an exact pair")
	}
	after.EvaluatorDigest = "other-evaluator"
	if exactPair(before, after) {
		t.Fatal("evaluator mismatch must not form an exact pair")
	}
}

func TestCacheStateAmbiguityIsNotComparable(t *testing.T) {
	before := receiptForTest()
	after := receiptForTest()
	after.Metrics.CacheHit = nil
	if cacheComparable(before, after) {
		t.Fatal("missing cache state must be ambiguous")
	}
}

func TestSemanticEqualityIsExactAndOrdered(t *testing.T) {
	if !semanticEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("equal semantic arrays must compare equal")
	}
	if semanticEqual([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("semantic array order must be preserved")
	}
}

func TestFixedCaseCardinality(t *testing.T) {
	if len(RequiredCaseIDs) != 12 {
		t.Fatalf("expected 12 canonical case IDs, got %d", len(RequiredCaseIDs))
	}
	if len(ArtifactNames) != 7 || len(RequiredUnknownFields) != 6 {
		t.Fatal("fixed output or unknown cardinality changed")
	}
}
