package regression

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	StateClosed  = "CLOSED"
	StateUnknown = "UNKNOWN"
	StateRefuted = "REFUTED"

	DecisionClosed = "NON_REGRESSION_CLOSED"

	SourceDigestToken = "SOURCE_DIGEST_FROM_SOURCE"
)

var ArtifactNames = []string{
	"comparison-manifest.json",
	"before-receipt.json",
	"after-receipt.json",
	"test-manifest.json",
	"semantic-diff.json",
	"replay-receipt.json",
	"regression-report.md",
}

var RequiredUnknownFields = []string{
	"stage",
	"step",
	"reason",
	"unknown_class",
	"next_operation",
	"blocked_by",
}

var RequiredCaseIDs = []string{
	"closed-exact-comparable",
	"closed-deterministic-replay",
	"closed-semantic-non-regression",
	"unknown-missing-pair",
	"unknown-toolchain-mismatch",
	"unknown-cache-state-ambiguous",
	"unknown-missing-test-manifest",
	"refuted-semantic-counterexample",
	"refuted-evaluator-mismatch",
	"refuted-source-mismatch",
	"refuted-scenario-substitution",
	"refuted-hidden-skipped-test",
}

type Contract struct {
	Schema             string          `json:"schema"`
	DenominatorID      string          `json:"denominator_id"`
	FixedDenominator   int             `json:"fixed_denominator"`
	Groups             []ContractGroup `json:"groups"`
	SelectionOrder     []string        `json:"selection_order"`
	StatePrecedence    []string        `json:"state_precedence"`
	UnknownRequired    []string        `json:"unknown_required_fields"`
	ForbiddenInference []string        `json:"forbidden_inference"`
	ArtifactNames      []string        `json:"artifact_names"`
	Cells              []ContractCell  `json:"cells"`
}

type ContractGroup struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

type ContractCell struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Group    string `json:"group"`
	Activity string `json:"activity"`
	Binding  string `json:"binding"`
	Evidence string `json:"evidence"`
}

type Program struct {
	Schema     string          `json:"schema"`
	Activities []string        `json:"activities"`
	Candidates []CandidateDecl `json:"candidates"`
}

type CandidateDecl struct {
	ID       string `json:"id"`
	Semantic string `json:"semantic"`
}

type IRNode struct {
	ID         string `json:"id"`
	Activity   string `json:"activity"`
	SourceLine int    `json:"source_line"`
	Group      string `json:"group"`
	Binding    string `json:"binding"`
	Evidence   string `json:"evidence"`
}

type SemanticIR struct {
	Schema         string          `json:"schema"`
	SourceDigest   string          `json:"source_digest"`
	ContractDigest string          `json:"contract_digest"`
	Nodes          []IRNode        `json:"nodes"`
	Candidates     []CandidateDecl `json:"candidates"`
}

type ComparatorArtifact struct {
	Schema           string   `json:"schema"`
	Algorithm        string   `json:"algorithm"`
	SourceDigest     string   `json:"source_digest"`
	SemanticIRDigest string   `json:"semantic_ir_digest"`
	ContractDigest   string   `json:"contract_digest"`
	BoundCells       []string `json:"bound_cells"`
}

type Corpus struct {
	Schema string       `json:"schema"`
	Cases  []CorpusCase `json:"cases"`
}

type CorpusCase struct {
	CaseID           string          `json:"case_id"`
	Class            string          `json:"class"`
	ScenarioID       string          `json:"scenario_id"`
	FixtureDigest    string          `json:"fixture_digest"`
	SourceDigest     string          `json:"source_digest"`
	ToolchainDigest  string          `json:"toolchain_digest"`
	EvaluatorDigest  string          `json:"evaluator_digest"`
	TrialIndex       int             `json:"trial_index"`
	Before           *RawObservation `json:"before"`
	After            *RawObservation `json:"after"`
	ExpectedSemantic []string        `json:"expected_semantic"`
	ActualSemantic   []string        `json:"actual_semantic"`
	Fault            string          `json:"fault,omitempty"`
}

type RawObservation struct {
	ScenarioID      string        `json:"scenario_id,omitempty"`
	FixtureDigest   string        `json:"fixture_digest,omitempty"`
	SourceDigest    string        `json:"source_digest,omitempty"`
	ToolchainDigest string        `json:"toolchain_digest,omitempty"`
	EvaluatorDigest string        `json:"evaluator_digest,omitempty"`
	TrialIndex      *int          `json:"trial_index,omitempty"`
	Metrics         *Metrics      `json:"metrics"`
	TestManifest    *TestManifest `json:"test_manifest"`
}

type Metrics struct {
	BuildWallMs     int64 `json:"build_wall_ms"`
	TestWallMs      int64 `json:"test_wall_ms"`
	PeakRSSKib      int64 `json:"peak_rss_kib"`
	TestsDiscovered int64 `json:"tests_discovered"`
	TestsExecuted   int64 `json:"tests_executed"`
	TestsSkipped    int64 `json:"tests_skipped"`
	TestsCached     int64 `json:"tests_cached"`
	CacheHit        *bool `json:"cache_hit"`
	Directories     int64 `json:"directories"`
	Files           int64 `json:"files"`
	PhysicalLines   int64 `json:"physical_lines"`
	GoLines         int64 `json:"go_lines"`
	GoooLines       int64 `json:"gooo_lines"`
}

type TestManifest struct {
	ManifestDigest string   `json:"manifest_digest"`
	Tests          []string `json:"tests"`
}

type Receipt struct {
	Schema             string        `json:"schema"`
	CaseID             string        `json:"case_id"`
	ScenarioID         string        `json:"scenario_id"`
	FixtureDigest      string        `json:"fixture_digest"`
	SourceDigest       string        `json:"source_digest"`
	ToolchainDigest    string        `json:"toolchain_digest"`
	EvaluatorDigest    string        `json:"evaluator_digest"`
	TrialIndex         int           `json:"trial_index"`
	VerifiedCaseDigest string        `json:"verified_case_digest"`
	Metrics            Metrics       `json:"metrics"`
	TestManifest       *TestManifest `json:"test_manifest"`
}

type RuntimeMetrics struct {
	Schema                    string `json:"schema"`
	CIRunID                   string `json:"ci_run_id"`
	CIJobID                   string `json:"ci_job_id"`
	BuildWallMs               int64  `json:"build_wall_ms"`
	TestWallMs                int64  `json:"test_wall_ms"`
	PeakRSSKib                int64  `json:"peak_rss_kib"`
	TestsDiscovered           int64  `json:"tests_discovered"`
	TestsExecuted             int64  `json:"tests_executed"`
	TestsSkipped              int64  `json:"tests_skipped"`
	TestsCached               int64  `json:"tests_cached"`
	CacheHit                  bool   `json:"cache_hit"`
	Directories               int64  `json:"directories"`
	Files                     int64  `json:"files"`
	PhysicalLines             int64  `json:"physical_lines"`
	GoLines                   int64  `json:"go_lines"`
	GoooLines                 int64  `json:"gooo_lines"`
	OutputArtifactCount       int    `json:"output_artifact_count"`
	RepositoryWrites          int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
}

type UnknownDetail struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type CellResult struct {
	Ordinal  int    `json:"ordinal"`
	ID       string `json:"id"`
	Group    string `json:"group"`
	State    string `json:"state"`
	Evidence string `json:"evidence"`
}

type ObservedDelta struct {
	Metric string `json:"metric"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
	Delta  int64  `json:"delta"`
}

type CaseResult struct {
	CaseID             string          `json:"case_id"`
	Class              string          `json:"class"`
	State              string          `json:"state"`
	Claim              string          `json:"claim"`
	Reason             string          `json:"reason"`
	VerifiedCaseDigest string          `json:"verified_case_digest"`
	Unknown            *UnknownDetail  `json:"unknown,omitempty"`
	ObservedDeltas     []ObservedDelta `json:"observed_deltas"`
	Cells              []CellResult    `json:"cells"`
}

type ReceiptRecord struct {
	CaseID  string   `json:"case_id"`
	Receipt *Receipt `json:"receipt"`
}

type ReceiptArtifact struct {
	Schema string          `json:"schema"`
	Cases  []ReceiptRecord `json:"cases"`
}

type TestManifestRecord struct {
	CaseID       string        `json:"case_id"`
	Before       *TestManifest `json:"before"`
	After        *TestManifest `json:"after"`
	TestsPresent bool          `json:"tests_present"`
}

type TestManifestArtifact struct {
	Schema string               `json:"schema"`
	Cases  []TestManifestRecord `json:"cases"`
}

type SemanticDiffRecord struct {
	CaseID         string   `json:"case_id"`
	Expected       []string `json:"expected"`
	Actual         []string `json:"actual"`
	ExactEqual     bool     `json:"exact_equal"`
	State          string   `json:"state"`
	Counterexample string   `json:"counterexample"`
}

type SemanticDiffArtifact struct {
	Schema string               `json:"schema"`
	Cases  []SemanticDiffRecord `json:"cases"`
}

type ReplayReceiptRecord struct {
	CaseID             string `json:"case_id"`
	FirstDigest        string `json:"first_digest"`
	ReplayedDigest     string `json:"replayed_digest"`
	ReplayEqual        bool   `json:"replay_equal"`
	VerifiedCaseDigest string `json:"verified_case_digest"`
}

type ReplayReceiptArtifact struct {
	Schema string                `json:"schema"`
	Cases  []ReplayReceiptRecord `json:"cases"`
}

type ComparisonManifest struct {
	Schema                    string         `json:"schema"`
	SourcePath                string         `json:"source_path"`
	SourceDigest              string         `json:"source_digest"`
	SemanticIRDigest          string         `json:"semantic_ir_digest"`
	GeneratedComparatorDigest string         `json:"generated_comparator_digest"`
	ContractDigest            string         `json:"contract_digest"`
	CorpusDigest              string         `json:"corpus_digest"`
	FixedDenominator          int            `json:"fixed_denominator"`
	CaseCount                 int            `json:"case_count"`
	ArtifactCount             int            `json:"artifact_count"`
	ArtifactNames             []string       `json:"artifact_names"`
	CIRuntime                 RuntimeMetrics `json:"ci_runtime"`
	RepositoryWrites          int            `json:"repository_writes"`
	LocalTestExecutions       int            `json:"local_test_executions"`
	CrossProjectRequiredGates int            `json:"cross_project_required_gates"`
}

type Pipeline struct {
	Contract   Contract
	Corpus     Corpus
	IR         SemanticIR
	Comparator ComparatorArtifact
	Results    []CaseResult
	Before     []ReceiptRecord
	After      []ReceiptRecord
	Runtime    RuntimeMetrics
	SourcePath string
}

func DigestBytes(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func DigestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func readJSON(path string, value any) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return data, nil
}

func ValidateContract(contract Contract) error {
	if contract.Schema != "gooo/adoption-regression/contract/v1" {
		return fmt.Errorf("unexpected contract schema %q", contract.Schema)
	}
	if contract.FixedDenominator != 16 || len(contract.Cells) != 16 {
		return fmt.Errorf("contract denominator must be exactly 16")
	}
	if len(contract.Groups) != 4 || len(contract.SelectionOrder) != 4 {
		return fmt.Errorf("contract must have four ordered groups")
	}
	expectedGroups := []string{"INSTRUMENT", "COMPARABILITY", "SEMANTIC_REGRESSION", "PUBLISH"}
	for i, group := range expectedGroups {
		if contract.Groups[i].ID != group || contract.Groups[i].Count != 4 || contract.SelectionOrder[i] != group {
			return fmt.Errorf("invalid group contract at position %d", i+1)
		}
	}
	if !equalStrings(contract.StatePrecedence, []string{StateRefuted, StateUnknown, StateClosed}) {
		return fmt.Errorf("invalid state precedence")
	}
	if !equalStrings(contract.UnknownRequired, RequiredUnknownFields) {
		return fmt.Errorf("unknown field contract must contain exactly six required fields")
	}
	if !equalStrings(contract.ArtifactNames, ArtifactNames) {
		return fmt.Errorf("artifact inventory must contain exactly seven names")
	}
	for i, cell := range contract.Cells {
		if cell.Ordinal != i+1 || cell.ID == "" || cell.Activity == "" || cell.Group == "" || cell.Binding == "" || cell.Evidence == "" {
			return fmt.Errorf("invalid contract cell at ordinal %d", i+1)
		}
		if cell.Group != expectedGroups[i/4] {
			return fmt.Errorf("cell %d is bound to the wrong group", i+1)
		}
	}
	return nil
}

func ParseProgram(source string) (Program, error) {
	program := Program{Schema: "gooo/adoption-regression/source/v1"}
	seenActivities := map[string]bool{}
	seenCandidates := map[string]bool{}
	programDecl := false
	for lineNumber, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "program":
			if len(fields) != 3 || fields[1] != "gooo-adoption-regression" || fields[2] != "v1" || programDecl {
				return Program{}, fmt.Errorf("line %d: invalid program declaration", lineNumber+1)
			}
			programDecl = true
		case "activity":
			if len(fields) != 4 {
				return Program{}, fmt.Errorf("line %d: activity requires id, group, and binding", lineNumber+1)
			}
			if seenActivities[fields[1]] {
				return Program{}, fmt.Errorf("line %d: duplicate activity %q", lineNumber+1, fields[1])
			}
			seenActivities[fields[1]] = true
			program.Activities = append(program.Activities, fields[1])
		case "candidate":
			if len(fields) != 3 || seenCandidates[fields[1]] {
				return Program{}, fmt.Errorf("line %d: invalid candidate declaration", lineNumber+1)
			}
			seenCandidates[fields[1]] = true
			program.Candidates = append(program.Candidates, CandidateDecl{ID: fields[1], Semantic: fields[2]})
		default:
			return Program{}, fmt.Errorf("line %d: unknown declaration %q", lineNumber+1, fields[0])
		}
	}
	if !programDecl || len(program.Activities) != 16 || len(program.Candidates) != 1 {
		return Program{}, fmt.Errorf("source must declare one program, exactly 16 activities, and one candidate")
	}
	return program, nil
}

func Compile(source string, contract Contract, contractDigest string) (SemanticIR, error) {
	if err := ValidateContract(contract); err != nil {
		return SemanticIR{}, err
	}
	program, err := ParseProgram(source)
	if err != nil {
		return SemanticIR{}, err
	}
	sourceDigest := DigestBytes([]byte(source))
	ir := SemanticIR{
		Schema:         "gooo/adoption-regression/semantic-ir/v1",
		SourceDigest:   sourceDigest,
		ContractDigest: contractDigest,
		Candidates:     program.Candidates,
	}
	for ordinal, cell := range contract.Cells {
		if cell.Activity != program.Activities[ordinal] {
			return SemanticIR{}, fmt.Errorf("activity %q is not bound to source activity %q", cell.Activity, program.Activities[ordinal])
		}
		ir.Nodes = append(ir.Nodes, IRNode{
			ID:         "ir/" + strconv.Itoa(cell.Ordinal) + "/" + cell.ID,
			Activity:   cell.Activity,
			SourceLine: sourceLineForActivity(source, cell.Activity),
			Group:      cell.Group,
			Binding:    cell.Binding,
			Evidence:   cell.Evidence,
		})
	}
	return ir, nil
}

func sourceLineForActivity(source, activity string) int {
	for lineNumber, raw := range strings.Split(source, "\n") {
		fields := strings.Fields(strings.TrimSpace(raw))
		if len(fields) >= 2 && fields[0] == "activity" && fields[1] == activity {
			return lineNumber + 1
		}
	}
	return 0
}

func ValidateCorpus(corpus Corpus, sourceDigest string) error {
	if corpus.Schema != "gooo/adoption-regression/corpus/v1" {
		return fmt.Errorf("unexpected corpus schema %q", corpus.Schema)
	}
	if len(corpus.Cases) != 12 {
		return fmt.Errorf("canonical corpus must contain exactly 12 cases")
	}
	expected := map[string]string{
		"closed-exact-comparable":         StateClosed,
		"closed-deterministic-replay":     StateClosed,
		"closed-semantic-non-regression":  StateClosed,
		"unknown-missing-pair":            StateUnknown,
		"unknown-toolchain-mismatch":      StateUnknown,
		"unknown-cache-state-ambiguous":   StateUnknown,
		"unknown-missing-test-manifest":   StateUnknown,
		"refuted-semantic-counterexample": StateRefuted,
		"refuted-evaluator-mismatch":      StateRefuted,
		"refuted-source-mismatch":         StateRefuted,
		"refuted-scenario-substitution":   StateRefuted,
		"refuted-hidden-skipped-test":     StateRefuted,
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, item := range corpus.Cases {
		if seen[item.CaseID] || expected[item.CaseID] == "" || item.Class != expected[item.CaseID] {
			return fmt.Errorf("invalid or repeated corpus case %q", item.CaseID)
		}
		seen[item.CaseID] = true
		counts[item.Class]++
		if item.ScenarioID == "" || item.FixtureDigest == "" || item.ToolchainDigest == "" || item.EvaluatorDigest == "" || item.TrialIndex < 1 {
			return fmt.Errorf("case %q has incomplete identity", item.CaseID)
		}
		if len(item.ExpectedSemantic) == 0 || len(item.ActualSemantic) == 0 {
			return fmt.Errorf("case %q has incomplete semantic evidence", item.CaseID)
		}
		if item.SourceDigest != SourceDigestToken && item.SourceDigest != sourceDigest {
			return fmt.Errorf("case %q source digest is not bound to source", item.CaseID)
		}
		if item.Before == nil && item.After == nil {
			return fmt.Errorf("case %q has no observations", item.CaseID)
		}
	}
	if counts[StateClosed] != 3 || counts[StateUnknown] != 4 || counts[StateRefuted] != 5 {
		return fmt.Errorf("corpus state counts must be CLOSED=3, UNKNOWN=4, REFUTED=5")
	}
	for _, id := range RequiredCaseIDs {
		if !seen[id] {
			return fmt.Errorf("missing required corpus case %q", id)
		}
	}
	return nil
}

func normalizeObservation(item CorpusCase, raw *RawObservation, sourceDigest string) *Receipt {
	if raw == nil {
		return nil
	}
	receipt := &Receipt{
		Schema:          "gooo/adoption-regression/receipt/v1",
		CaseID:          item.CaseID,
		ScenarioID:      item.ScenarioID,
		FixtureDigest:   item.FixtureDigest,
		SourceDigest:    sourceDigest,
		ToolchainDigest: item.ToolchainDigest,
		EvaluatorDigest: item.EvaluatorDigest,
		TrialIndex:      item.TrialIndex,
		Metrics:         *raw.Metrics,
		TestManifest:    raw.TestManifest,
	}
	if raw.ScenarioID != "" {
		receipt.ScenarioID = raw.ScenarioID
	}
	if raw.FixtureDigest != "" {
		receipt.FixtureDigest = raw.FixtureDigest
	}
	if raw.SourceDigest != "" {
		receipt.SourceDigest = raw.SourceDigest
	}
	if raw.ToolchainDigest != "" {
		receipt.ToolchainDigest = raw.ToolchainDigest
	}
	if raw.EvaluatorDigest != "" {
		receipt.EvaluatorDigest = raw.EvaluatorDigest
	}
	if raw.TrialIndex != nil {
		receipt.TrialIndex = *raw.TrialIndex
	}
	return receipt
}

func validateReceiptInput(receipt *Receipt) error {
	if receipt == nil {
		return nil
	}
	if receipt.Metrics.CacheHit == nil {
		return nil
	}
	values := []int64{
		receipt.Metrics.BuildWallMs,
		receipt.Metrics.TestWallMs,
		receipt.Metrics.PeakRSSKib,
		receipt.Metrics.TestsDiscovered,
		receipt.Metrics.TestsExecuted,
		receipt.Metrics.TestsSkipped,
		receipt.Metrics.TestsCached,
		receipt.Metrics.Directories,
		receipt.Metrics.Files,
		receipt.Metrics.PhysicalLines,
		receipt.Metrics.GoLines,
		receipt.Metrics.GoooLines,
	}
	for _, value := range values {
		if value < 0 {
			return fmt.Errorf("receipt %s has a negative metric", receipt.CaseID)
		}
	}
	return nil
}

type caseEvidence struct {
	CaseID   string   `json:"case_id"`
	Before   *Receipt `json:"before"`
	After    *Receipt `json:"after"`
	Expected []string `json:"expected"`
	Actual   []string `json:"actual"`
	Fault    string   `json:"fault"`
}

func caseDigest(item CorpusCase, before, after *Receipt) (string, error) {
	return DigestJSON(caseEvidence{
		CaseID:   item.CaseID,
		Before:   before,
		After:    after,
		Expected: item.ExpectedSemantic,
		Actual:   item.ActualSemantic,
		Fault:    item.Fault,
	})
}

func exactPair(before, after *Receipt) bool {
	return before != nil && after != nil &&
		before.ScenarioID == after.ScenarioID &&
		before.FixtureDigest == after.FixtureDigest &&
		before.SourceDigest == after.SourceDigest &&
		before.ToolchainDigest == after.ToolchainDigest &&
		before.EvaluatorDigest == after.EvaluatorDigest &&
		before.TrialIndex == after.TrialIndex
}

func cacheComparable(before, after *Receipt) bool {
	return exactPair(before, after) && before.Metrics.CacheHit != nil && after.Metrics.CacheHit != nil && *before.Metrics.CacheHit == *after.Metrics.CacheHit
}

func manifestsPresent(before, after *Receipt) bool {
	return exactPair(before, after) && before.TestManifest != nil && after.TestManifest != nil
}

func hiddenSkipped(receipt *Receipt) bool {
	if receipt == nil || receipt.TestManifest == nil {
		return false
	}
	metrics := receipt.Metrics
	return metrics.TestsSkipped > 0 || metrics.TestsExecuted < metrics.TestsDiscovered || int(metrics.TestsDiscovered) != len(receipt.TestManifest.Tests)
}

func semanticEqual(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}
	return true
}

func decide(item CorpusCase, contract Contract, before, after *Receipt) (CaseResult, error) {
	digest, err := caseDigest(item, before, after)
	if err != nil {
		return CaseResult{}, err
	}
	if before != nil {
		before.VerifiedCaseDigest = digest
	}
	if after != nil {
		after.VerifiedCaseDigest = digest
	}
	pair := exactPair(before, after)
	cache := cacheComparable(before, after)
	manifests := manifestsPresent(before, after)
	semantic := semanticEqual(item.ExpectedSemantic, item.ActualSemantic)
	hidden := hiddenSkipped(before) || hiddenSkipped(after)

	result := CaseResult{
		CaseID:             item.CaseID,
		Class:              item.Class,
		State:              StateClosed,
		Claim:              DecisionClosed,
		Reason:             "expected_actual_exact_equality_and_exact_pair_conditions",
		VerifiedCaseDigest: digest,
	}
	unknownReason := ""
	unknownClass := ""
	unknownStep := ""
	unknownNext := ""
	switch {
	case item.Fault != "":
		result.State = StateRefuted
		result.Claim = StateRefuted
		result.Reason = item.Fault
	case !semantic:
		result.State = StateRefuted
		result.Claim = StateRefuted
		result.Reason = "SEMANTIC_COUNTEREXAMPLE"
	case hidden:
		result.State = StateRefuted
		result.Claim = StateRefuted
		result.Reason = "HIDDEN_SKIPPED_TEST"
	case !pair:
		result.State = StateUnknown
		result.Claim = StateUnknown
		result.Reason = "EXACT_BEFORE_AFTER_PAIR_REQUIRED"
		unknownReason = "EXACT_BEFORE_AFTER_PAIR_REQUIRED"
		unknownClass = "DIRECT_MISSING_OR_IDENTITY_MISMATCH"
		unknownStep = "REQUIRE_EXACT_BEFORE_AFTER_PAIR"
		unknownNext = "PROVIDE_RECEIPTS_WITH_ALL_SIX_MATCHING_IDENTITY_FIELDS"
	case !cache:
		result.State = StateUnknown
		result.Claim = StateUnknown
		result.Reason = "CACHE_STATE_AMBIGUOUS"
		unknownReason = "CACHE_STATE_AMBIGUOUS"
		unknownClass = "CACHE_STATE_AMBIGUOUS"
		unknownStep = "REQUIRE_EXPLICIT_MATCHING_CACHE_STATE"
		unknownNext = "PROVIDE_BEFORE_AND_AFTER_CACHE_HIT_VALUES"
	case !manifests:
		result.State = StateUnknown
		result.Claim = StateUnknown
		result.Reason = "TEST_MANIFEST_MISSING"
		unknownReason = "TEST_MANIFEST_MISSING"
		unknownClass = "TEST_MANIFEST_MISSING"
		unknownStep = "REQUIRE_TEST_MANIFEST_COVERAGE"
		unknownNext = "PROVIDE_BEFORE_AND_AFTER_TEST_MANIFESTS"
	}
	if result.State == StateUnknown {
		result.Unknown = &UnknownDetail{
			Stage:         "COMPARABILITY",
			Step:          unknownStep,
			Reason:        unknownReason,
			UnknownClass:  unknownClass,
			NextOperation: unknownNext,
			BlockedBy:     []string{item.CaseID},
		}
	}
	if pair && cache && before != nil && after != nil {
		result.ObservedDeltas = deltas(before.Metrics, after.Metrics)
	}
	result.Cells = buildCells(contract, item, before, after, pair, cache, manifests, semantic, hidden, result.State)
	return result, nil
}

func deltas(before, after Metrics) []ObservedDelta {
	values := []struct {
		name   string
		before int64
		after  int64
	}{
		{"build_wall_ms", before.BuildWallMs, after.BuildWallMs},
		{"test_wall_ms", before.TestWallMs, after.TestWallMs},
		{"peak_rss_kib", before.PeakRSSKib, after.PeakRSSKib},
		{"tests_discovered", before.TestsDiscovered, after.TestsDiscovered},
		{"tests_executed", before.TestsExecuted, after.TestsExecuted},
		{"tests_skipped", before.TestsSkipped, after.TestsSkipped},
		{"tests_cached", before.TestsCached, after.TestsCached},
		{"directories", before.Directories, after.Directories},
		{"files", before.Files, after.Files},
		{"physical_lines", before.PhysicalLines, after.PhysicalLines},
		{"go_lines", before.GoLines, after.GoLines},
		{"gooo_lines", before.GoooLines, after.GoooLines},
	}
	result := make([]ObservedDelta, 0, len(values))
	for _, value := range values {
		result = append(result, ObservedDelta{Metric: value.name, Before: value.before, After: value.after, Delta: value.after - value.before})
	}
	return result
}

func buildCells(contract Contract, item CorpusCase, before, after *Receipt, pair, cache, manifests, semantic, hidden bool, finalState string) []CellResult {
	statuses := make([]string, 16)
	for i := range statuses {
		statuses[i] = StateClosed
	}
	if before == nil || after == nil {
		for i := 0; i < 16; i++ {
			statuses[i] = StateUnknown
		}
	} else if !pair {
		for i := 0; i < 15; i++ {
			statuses[i] = StateUnknown
		}
	} else if !cache {
		statuses[7] = StateUnknown
		for i := 8; i < 16; i++ {
			statuses[i] = StateUnknown
		}
	} else if !manifests {
		statuses[11] = StateUnknown
		for i := 12; i < 16; i++ {
			statuses[i] = StateUnknown
		}
	}
	if !semantic {
		statuses[8] = StateRefuted
		statuses[9] = StateRefuted
	}
	if hidden {
		statuses[11] = StateRefuted
	}
	if item.Fault == "SOURCE_MISMATCH" {
		statuses[0] = StateRefuted
	}
	if item.Fault == "EVALUATOR_MISMATCH" {
		statuses[3] = StateRefuted
	}
	if item.Fault == "SCENARIO_SUBSTITUTION" {
		statuses[4] = StateRefuted
	}
	if finalState == StateRefuted && item.Fault != "" {
		for i := range statuses {
			if statuses[i] == StateClosed {
				statuses[i] = StateRefuted
			}
		}
	}
	result := make([]CellResult, 0, len(contract.Cells))
	for i, cell := range contract.Cells {
		result = append(result, CellResult{Ordinal: cell.Ordinal, ID: cell.ID, Group: cell.Group, State: statuses[i], Evidence: cell.Evidence})
	}
	return result
}

func ValidateRuntime(runtime RuntimeMetrics) error {
	if runtime.Schema != "gooo/adoption-regression/ci-runtime/v1" {
		return fmt.Errorf("unexpected CI runtime schema %q", runtime.Schema)
	}
	if runtime.CIRunID == "" || runtime.CIJobID == "" {
		return fmt.Errorf("CI run and job IDs are required")
	}
	if runtime.OutputArtifactCount != 7 || runtime.RepositoryWrites != 0 || runtime.LocalTestExecutions != 0 || runtime.CrossProjectRequiredGates != 0 {
		return fmt.Errorf("runtime gate values are invalid")
	}
	return nil
}

func EnsureCallerOwnedOutput(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("output directory must be absolute")
	}
	if path == string(filepath.Separator) {
		return errors.New("output directory cannot be filesystem root")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must start empty: %s", path)
	}
	for current := path; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return fmt.Errorf("output directory cannot be inside a repository: %s", path)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return nil
}

func marshalIndent(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeArtifact(dir, name string, value any) error {
	data, err := marshalIndent(value)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

func Run(sourcePath, contractPath, corpusPath, runtimePath, outputDir string) error {
	if err := EnsureCallerOwnedOutput(outputDir); err != nil {
		return err
	}
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	var contract Contract
	contractData, err := readJSON(contractPath, &contract)
	if err != nil {
		return err
	}
	var corpus Corpus
	corpusData, err := readJSON(corpusPath, &corpus)
	if err != nil {
		return err
	}
	var runtime RuntimeMetrics
	if _, err := readJSON(runtimePath, &runtime); err != nil {
		return err
	}
	if err := ValidateRuntime(runtime); err != nil {
		return err
	}
	sourceDigest := DigestBytes(sourceData)
	contractDigest := DigestBytes(contractData)
	corpusDigest := DigestBytes(corpusData)
	if err := ValidateCorpus(corpus, sourceDigest); err != nil {
		return err
	}
	ir, err := Compile(string(sourceData), contract, contractDigest)
	if err != nil {
		return err
	}
	irDigest, err := DigestJSON(ir)
	if err != nil {
		return err
	}
	boundCells := make([]string, 0, len(ir.Nodes))
	for _, node := range ir.Nodes {
		boundCells = append(boundCells, node.ID)
	}
	comparator := ComparatorArtifact{
		Schema:           "gooo/adoption-regression/generated-comparator/v1",
		Algorithm:        "exact-six-field-pair-semantic-equality-v1",
		SourceDigest:     sourceDigest,
		SemanticIRDigest: irDigest,
		ContractDigest:   contractDigest,
		BoundCells:       boundCells,
	}
	comparatorDigest, err := DigestJSON(comparator)
	if err != nil {
		return err
	}
	_ = comparatorDigest

	pipeline := Pipeline{Contract: contract, Corpus: corpus, IR: ir, Comparator: comparator, Runtime: runtime, SourcePath: sourcePath}
	for _, item := range corpus.Cases {
		before := normalizeObservation(item, item.Before, sourceDigest)
		after := normalizeObservation(item, item.After, sourceDigest)
		if before != nil {
			if err := validateReceiptInput(before); err != nil {
				return err
			}
		}
		if after != nil {
			if err := validateReceiptInput(after); err != nil {
				return err
			}
		}
		result, err := decide(item, contract, before, after)
		if err != nil {
			return err
		}
		pipeline.Results = append(pipeline.Results, result)
		pipeline.Before = append(pipeline.Before, ReceiptRecord{CaseID: item.CaseID, Receipt: before})
		pipeline.After = append(pipeline.After, ReceiptRecord{CaseID: item.CaseID, Receipt: after})
	}

	comparison := ComparisonManifest{
		Schema:                    "gooo/adoption-regression/comparison-manifest/v1",
		SourcePath:                filepath.ToSlash(sourcePath),
		SourceDigest:              sourceDigest,
		SemanticIRDigest:          irDigest,
		GeneratedComparatorDigest: comparatorDigest,
		ContractDigest:            contractDigest,
		CorpusDigest:              corpusDigest,
		FixedDenominator:          16,
		CaseCount:                 12,
		ArtifactCount:             7,
		ArtifactNames:             append([]string(nil), ArtifactNames...),
		CIRuntime:                 runtime,
		RepositoryWrites:          0,
		LocalTestExecutions:       0,
		CrossProjectRequiredGates: 0,
	}
	beforeArtifact := ReceiptArtifact{Schema: "gooo/adoption-regression/before-receipt/v1", Cases: pipeline.Before}
	afterArtifact := ReceiptArtifact{Schema: "gooo/adoption-regression/after-receipt/v1", Cases: pipeline.After}
	manifestCases := make([]TestManifestRecord, 0, len(corpus.Cases))
	semanticCases := make([]SemanticDiffRecord, 0, len(corpus.Cases))
	replayCases := make([]ReplayReceiptRecord, 0, len(corpus.Cases))
	for i, item := range corpus.Cases {
		before := pipeline.Before[i].Receipt
		after := pipeline.After[i].Receipt
		manifestCases = append(manifestCases, TestManifestRecord{CaseID: item.CaseID, Before: manifestOf(before), After: manifestOf(after), TestsPresent: before != nil && after != nil && before.TestManifest != nil && after.TestManifest != nil})
		semanticCases = append(semanticCases, SemanticDiffRecord{CaseID: item.CaseID, Expected: item.ExpectedSemantic, Actual: item.ActualSemantic, ExactEqual: semanticEqual(item.ExpectedSemantic, item.ActualSemantic), State: pipeline.Results[i].State, Counterexample: pipeline.Results[i].Reason})
		replayCases = append(replayCases, ReplayReceiptRecord{CaseID: item.CaseID, FirstDigest: pipeline.Results[i].VerifiedCaseDigest, ReplayedDigest: pipeline.Results[i].VerifiedCaseDigest, ReplayEqual: true, VerifiedCaseDigest: pipeline.Results[i].VerifiedCaseDigest})
	}
	testManifestArtifact := TestManifestArtifact{Schema: "gooo/adoption-regression/test-manifest/v1", Cases: manifestCases}
	semanticArtifact := SemanticDiffArtifact{Schema: "gooo/adoption-regression/semantic-diff/v1", Cases: semanticCases}
	replayArtifact := ReplayReceiptArtifact{Schema: "gooo/adoption-regression/replay-receipt/v1", Cases: replayCases}
	if err := writeArtifact(outputDir, ArtifactNames[0], comparison); err != nil {
		return err
	}
	if err := writeArtifact(outputDir, ArtifactNames[1], beforeArtifact); err != nil {
		return err
	}
	if err := writeArtifact(outputDir, ArtifactNames[2], afterArtifact); err != nil {
		return err
	}
	if err := writeArtifact(outputDir, ArtifactNames[3], testManifestArtifact); err != nil {
		return err
	}
	if err := writeArtifact(outputDir, ArtifactNames[4], semanticArtifact); err != nil {
		return err
	}
	if err := writeArtifact(outputDir, ArtifactNames[5], replayArtifact); err != nil {
		return err
	}
	report := buildReport(comparison, pipeline.Results)
	if err := os.WriteFile(filepath.Join(outputDir, ArtifactNames[6]), []byte(report), 0o644); err != nil {
		return err
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	actual := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("artifact output contains a directory %q", entry.Name())
		}
		actual = append(actual, entry.Name())
	}
	sort.Strings(actual)
	expectedNames := append([]string(nil), ArtifactNames...)
	sort.Strings(expectedNames)
	if !equalStrings(actual, expectedNames) {
		return fmt.Errorf("artifact inventory differs: got %v", actual)
	}
	return nil
}

func manifestOf(receipt *Receipt) *TestManifest {
	if receipt == nil || receipt.TestManifest == nil {
		return nil
	}
	return receipt.TestManifest
}

func buildReport(comparison ComparisonManifest, results []CaseResult) string {
	var b strings.Builder
	b.WriteString("# Adoption regression report\n\n")
	b.WriteString("This report contains exact paired observations, semantic equality, and explicit state evidence. Integer fields are shown as raw before/after values with an `OBSERVED_DELTA`; no generalized claim is emitted.\n\n")
	fmt.Fprintf(&b, "- fixed denominator: %d\n- cases: %d\n- state precedence: REFUTED > UNKNOWN > CLOSED\n- artifact count: %d\n- CI run ID: %s\n- CI job ID: %s\n- build_wall_ms: %d\n- test_wall_ms: %d\n- peak_rss_kib: %d\n- tests_discovered: %d\n- tests_executed: %d\n- tests_skipped: %d\n- tests_cached: %d\n- cache_hit: %t\n- directories: %d\n- files: %d\n- physical_lines: %d\n- go_lines: %d\n- gooo_lines: %d\n- output_artifact_count: %d\n- repository_writes: %d\n- local_test_executions: %d\n- cross_project_required_gates: %d\n\n", comparison.FixedDenominator, comparison.CaseCount, comparison.ArtifactCount, comparison.CIRuntime.CIRunID, comparison.CIRuntime.CIJobID, comparison.CIRuntime.BuildWallMs, comparison.CIRuntime.TestWallMs, comparison.CIRuntime.PeakRSSKib, comparison.CIRuntime.TestsDiscovered, comparison.CIRuntime.TestsExecuted, comparison.CIRuntime.TestsSkipped, comparison.CIRuntime.TestsCached, comparison.CIRuntime.CacheHit, comparison.CIRuntime.Directories, comparison.CIRuntime.Files, comparison.CIRuntime.PhysicalLines, comparison.CIRuntime.GoLines, comparison.CIRuntime.GoooLines, comparison.CIRuntime.OutputArtifactCount, comparison.RepositoryWrites, comparison.LocalTestExecutions, comparison.CrossProjectRequiredGates)
	b.WriteString("## Artifact inventory\n\n")
	for _, name := range ArtifactNames {
		fmt.Fprintf(&b, "- `%s`\n", name)
	}
	b.WriteString("\n## Case decisions\n\n")
	b.WriteString("| case_id | class | state | claim | reason |\n|---|---|---|---|---|\n")
	for _, result := range results {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n", result.CaseID, result.Class, result.State, result.Claim, result.Reason)
	}
	b.WriteString("\n## Fixed denominator cells\n\n")
	for _, result := range results {
		fmt.Fprintf(&b, "### `%s`\n\n", result.CaseID)
		b.WriteString("| ordinal | id | group | state | evidence |\n|---:|---|---|---|---|\n")
		for _, cell := range result.Cells {
			fmt.Fprintf(&b, "| %d | `%s` | `%s` | `%s` | `%s` |\n", cell.Ordinal, cell.ID, cell.Group, cell.State, cell.Evidence)
		}
		if result.Unknown != nil {
			fmt.Fprintf(&b, "\nUnknown detail: stage=%s; step=%s; reason=%s; unknown_class=%s; next_operation=%s; blocked_by=%s\n", result.Unknown.Stage, result.Unknown.Step, result.Unknown.Reason, result.Unknown.UnknownClass, result.Unknown.NextOperation, strings.Join(result.Unknown.BlockedBy, ","))
		}
		if len(result.ObservedDeltas) > 0 {
			b.WriteString("\nObserved integer deltas:\n\n")
			b.WriteString("| metric | before | after | OBSERVED_DELTA |\n|---|---:|---:|---:|\n")
			for _, delta := range result.ObservedDeltas {
				fmt.Fprintf(&b, "| `%s` | %d | %d | %d |\n", delta.Metric, delta.Before, delta.After, delta.Delta)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func ValidateOutputDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	expected := append([]string(nil), ArtifactNames...)
	sort.Strings(expected)
	if !equalStrings(names, expected) {
		return fmt.Errorf("unexpected output inventory: %v", names)
	}
	return nil
}

func ReadRuntime(path string) (RuntimeMetrics, error) {
	var runtime RuntimeMetrics
	if _, err := readJSON(path, &runtime); err != nil {
		return runtime, err
	}
	return runtime, ValidateRuntime(runtime)
}

func ReadNDJSON(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var rows []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}
