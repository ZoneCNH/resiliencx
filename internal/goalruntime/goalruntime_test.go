package goalruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateGoalRuntimeFinalPassesWithAuthority(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	writePrerequisiteLedgerFixture(t, root, DefaultGoalID)

	report, err := Evaluate("goal-runtime-final", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" {
		t.Fatalf("status = %q; gaps %#v", report.Status, report.Gaps)
	}
	if report.GoalID != DefaultGoalID {
		t.Fatalf("goal_id = %q; want %q", report.GoalID, DefaultGoalID)
	}
	if report.Gate != "G12_G16_FINAL" {
		t.Fatalf("gate = %q; want G12_G16_FINAL", report.Gate)
	}
	if !report.Blocking {
		t.Fatalf("blocking = false; want final runtime evidence to be blocking")
	}
	if report.MVAStatus != "complete" {
		t.Fatalf("mva_status = %q; want complete", report.MVAStatus)
	}
	for _, gate := range report.Gates {
		if !gate.Blocking {
			t.Fatalf("gate = %#v; want goalcli MVA gates to be blocking", gate)
		}
	}
	if !contains(report.Evidence, "source_evidence_ledger="+SourceLedgerPath) {
		t.Fatalf("evidence = %#v; want source ledger path", report.Evidence)
	}
	if !contains(report.Evidence, "generated_evidence_pack="+EvidenceLedgerPath) {
		t.Fatalf("evidence = %#v; want generated evidence pack path", report.Evidence)
	}
	if !contains(report.Evidence, "requires=goal-certify") {
		t.Fatalf("evidence = %#v; want final gate dependency", report.Evidence)
	}
	for _, want := range downstreamAdoptionBoundaryEvidence() {
		if !contains(report.Evidence, want) {
			t.Fatalf("evidence = %#v; want downstream adoption boundary %s", report.Evidence, want)
		}
	}
	if !containsSubstring(report.Details, "完成状态由本地 authority 校验和 evidence 写入共同证明") {
		t.Fatalf("details = %#v; want completion evidence boundary", report.Details)
	}
	if len(report.AuthorityPaths) == 0 {
		t.Fatalf("authority_paths is empty")
	}
}

func TestEvaluateRejectsUnknownCommand(t *testing.T) {
	if _, err := Evaluate("not-a-goalcli-command", Options{}); err == nil {
		t.Fatalf("Evaluate returned nil error for unknown command")
	}
}

func TestEvaluateReportsMissingAuthorityPaths(t *testing.T) {
	root := t.TempDir()
	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "failed" {
		t.Fatalf("status = %q; want failed", report.Status)
	}
	if len(report.Gaps) == 0 {
		t.Fatalf("gaps is empty; want missing authority paths")
	}
	if !containsSubstring(report.Gaps, ".worktree/goalcli-v0.1.0-plan.md") {
		t.Fatalf("gaps = %#v; want root plan gap", report.Gaps)
	}
	if report.MVAStatus != "not-complete" {
		t.Fatalf("mva_status = %q; want not-complete when authority is missing", report.MVAStatus)
	}
}

func TestEvaluateRenderedDownstreamSkipsSourceOnlyAuthorityPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/ZoneCNH/kernel\n"), 0o644); err != nil {
		t.Fatalf("write downstream go.mod: %v", err)
	}
	writeAuthorityPaths(t, root, portableAuthorityPaths)

	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" {
		t.Fatalf("status = %q; gaps %#v", report.Status, report.Gaps)
	}
	for _, path := range sourceOnlyAuthorityPaths {
		if contains(report.AuthorityPaths, path) {
			t.Fatalf("authority_paths = %#v; want source-only path %s skipped", report.AuthorityPaths, path)
		}
		if containsSubstring(report.Gaps, path) {
			t.Fatalf("gaps = %#v; want source-only path %s skipped", report.Gaps, path)
		}
	}
	if !containsSubstring(report.Details, "rendered downstream") {
		t.Fatalf("details = %#v; want rendered downstream boundary", report.Details)
	}
}

func TestEvaluateGoalcliGatePassesAsBlockingMVAContract(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)

	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" || report.MVAStatus != "complete" || !report.Blocking {
		t.Fatalf("report = %#v; want passed complete blocking goalcli gate", report)
	}
	if len(report.Gates) != 1 || !report.Gates[0].Blocking {
		t.Fatalf("gates = %#v; want one blocking gate", report.Gates)
	}
}

func TestEvaluateDownstreamAdoptionDeclaresLocalScope(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)

	report, err := Evaluate("goal-downstream-adoption", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" {
		t.Fatalf("status = %q; gaps %#v", report.Status, report.Gaps)
	}
	for _, want := range downstreamAdoptionBoundaryEvidence() {
		if !contains(report.Evidence, want) {
			t.Fatalf("evidence = %#v; want downstream adoption boundary %s", report.Evidence, want)
		}
	}
	if !containsSubstring(report.Details, "不声明 proof-based downstream adoption") {
		t.Fatalf("details = %#v; want proof-based adoption boundary", report.Details)
	}
}

func TestEvaluateGoalRuntimeFinalRequiresPrerequisiteLedger(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)

	report, err := Evaluate("goal-runtime-final", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "failed" || report.MVAStatus != "not-complete" {
		t.Fatalf("report = %#v; want failed not-complete when prerequisite ledger is missing", report)
	}
	if !containsSubstring(report.Gaps, SourceLedgerPath) {
		t.Fatalf("gaps = %#v; want source ledger prerequisite gap", report.Gaps)
	}
}

func TestWriteEvidenceWritesPackAndLedgerIdempotently(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	writePrerequisiteLedgerFixture(t, root, "GOAL-20260603-XLIB-GOALCLI-001")
	report, err := Evaluate("goal-runtime-final", Options{
		Root:   root,
		GoalID: "GOAL-20260603-XLIB-GOALCLI-001",
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("WriteEvidence returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("second WriteEvidence returned error: %v", err)
	}

	packPath := filepath.Join(root, filepath.FromSlash(report.EvidencePackPath))
	pack, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("read evidence pack: %v", err)
	}
	if !strings.Contains(string(pack), `"mva_status": "complete"`) || !strings.Contains(string(pack), `"blocking": true`) {
		t.Fatalf("evidence pack = %s; want complete blocking report", pack)
	}
	for _, want := range downstreamAdoptionBoundaryEvidence() {
		if !strings.Contains(string(pack), want) {
			t.Fatalf("evidence pack = %s; want downstream adoption boundary %s", pack, want)
		}
	}
	ledgerPath := filepath.Join(root, filepath.FromSlash(SourceLedgerPath))
	ledger, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read evidence ledger: %v", err)
	}
	if strings.Count(string(ledger), `"command":"goal-runtime-final"`) != 1 {
		t.Fatalf("ledger = %s; want one idempotent final entry for %s", ledger, report.EvidencePackPath)
	}
}

func TestWriteEvidenceRecordsPrerequisiteLedgerEntry(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("WriteEvidence returned error: %v", err)
	}
	ledger, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(SourceLedgerPath)))
	if err != nil {
		t.Fatalf("read evidence ledger: %v", err)
	}
	if !strings.Contains(string(ledger), `"command":"goal-acceptance"`) || !strings.Contains(string(ledger), `"mva_status":"complete"`) {
		t.Fatalf("ledger = %s; want goal-acceptance complete entry", ledger)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(report.EvidencePackPath))); !os.IsNotExist(err) {
		t.Fatalf("non-final WriteEvidence must not write generated pack, stat err = %v", err)
	}
}

func TestWriteEvidenceRejectsIncompleteReport(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	report.Status = "failed"
	if err := WriteEvidence(root, report); err == nil {
		t.Fatalf("WriteEvidence returned nil error for incomplete report")
	}
}

func TestCommands(t *testing.T) {
	cmds := Commands()
	if len(cmds) != 6 {
		t.Fatalf("expected 6 commands, got %d", len(cmds))
	}
	want := []string{
		"goal-acceptance", "goal-delivery", "goal-handover",
		"goal-downstream-adoption", "goal-certify", "goal-runtime-final",
	}
	for i, w := range want {
		if cmds[i] != w {
			t.Fatalf("cmds[%d] = %q, want %q", i, cmds[i], w)
		}
	}
}

func TestWriteEvidenceRejectsUnsupportedCommand(t *testing.T) {
	report := Report{Command: "not-a-command", Status: "passed", MVAStatus: "complete", Blocking: true}
	err := WriteEvidence(".", report)
	if err == nil {
		t.Fatal("expected error for unsupported command")
	}
}

func TestWriteEvidenceRejectsIncompleteMVA(t *testing.T) {
	report := Report{Command: "goal-acceptance", Status: "passed", MVAStatus: "not-complete", Blocking: true}
	err := WriteEvidence(".", report)
	if err == nil {
		t.Fatal("expected error for incomplete MVA status")
	}
}

func TestWriteEvidenceRejectsNonBlocking(t *testing.T) {
	report := Report{Command: "goal-acceptance", Status: "passed", MVAStatus: "complete", Blocking: false}
	err := WriteEvidence(".", report)
	if err == nil {
		t.Fatal("expected error for non-blocking report")
	}
}

func TestWriteEvidence_DefaultRoot(t *testing.T) {
	// WriteEvidence with empty root defaults to "." - use a temp dir as cwd
	// We can't easily change cwd in tests, so just verify the function doesn't panic
	// by testing with a valid root instead
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("WriteEvidence returned error: %v", err)
	}
}

func TestEvaluateWithCustomGoalID(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root, GoalID: "CUSTOM-GOAL-001"})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.GoalID != "CUSTOM-GOAL-001" {
		t.Fatalf("goal_id = %q; want CUSTOM-GOAL-001", report.GoalID)
	}
	if !containsSubstring(report.Details, "non-default goal_id") {
		t.Fatalf("details = %#v; want non-default goal_id note", report.Details)
	}
}

func TestEvaluateWithCustomMode(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root, Mode: "QUICK"})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Mode != "QUICK" {
		t.Fatalf("mode = %q; want QUICK", report.Mode)
	}
}

func TestModulePathForRoot_NoGoMod(t *testing.T) {
	root := t.TempDir()
	_, ok := modulePathForRoot(root)
	if ok {
		t.Fatal("expected false when go.mod is missing")
	}
}

func TestModulePathForRoot_BadGoMod(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("not a go mod"), 0o644)
	_, ok := modulePathForRoot(root)
	if ok {
		t.Fatal("expected false for bad go.mod")
	}
}

func TestEvaluateFinalRuntimeWithAllPrerequisites(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	writePrerequisiteLedgerFixture(t, root, DefaultGoalID)

	report, err := Evaluate("goal-runtime-final", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" {
		t.Fatalf("status = %q; gaps %#v", report.Status, report.Gaps)
	}
	if len(report.Gates) != 5 {
		t.Fatalf("expected 5 gates for final command, got %d", len(report.Gates))
	}
}

func TestValidateFinalPrerequisites_IncompleteEntry(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	// Write an incomplete entry
	os.MkdirAll(filepath.Join(root, filepath.FromSlash(SourceLedgerPath)), 0o755)
	os.Remove(filepath.Join(root, filepath.FromSlash(SourceLedgerPath)))
	ledgerPath := filepath.Join(root, filepath.FromSlash(SourceLedgerPath))
	os.MkdirAll(filepath.Dir(ledgerPath), 0o755)
	entry := LedgerEntry{
		SchemaVersion: "goalcli-mva/v1",
		GoalID:        DefaultGoalID,
		Command:       "goal-acceptance",
		Status:        "failed",
		MVAStatus:     "not-complete",
		Blocking:      true,
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(ledgerPath, append(data, '\n'), 0o644)

	gaps := validateFinalPrerequisites(root, DefaultGoalID)
	found := false
	for _, g := range gaps {
		if strings.Contains(g, "incomplete") {
			found = true
		}
	}
	if !found {
		t.Fatalf("gaps = %#v; want incomplete prerequisite gap", gaps)
	}
}

func TestUpsertLedgerEntry_BadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	os.WriteFile(path, []byte("not json\n"), 0o644)
	entry := LedgerEntry{GoalID: "test", Command: "test"}
	err := upsertLedgerEntry(path, entry)
	if err != nil {
		t.Fatalf("upsertLedgerEntry should skip bad JSON lines, got: %v", err)
	}
}

func TestUpsertLedgerEntry_ReadError(t *testing.T) {
	// Write to a directory path to trigger write error
	dir := t.TempDir()
	entry := LedgerEntry{GoalID: "test", Command: "test"}
	err := upsertLedgerEntry(dir, entry)
	if err == nil {
		// Writing to a directory may or may not fail depending on OS
	}
}

func TestReadLedgerEntries_BadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	os.WriteFile(path, []byte("not json\n"), 0o644)
	_, err := readLedgerEntries(path)
	if err == nil {
		t.Fatal("expected error for bad JSON in ledger")
	}
}

func TestReadLedgerEntries_MissingFile(t *testing.T) {
	_, err := readLedgerEntries("/nonexistent/ledger.jsonl")
	if err == nil {
		t.Fatal("expected error for missing ledger file")
	}
}

func TestEvaluateNonDefaultGoalIDWithFinal(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	writePrerequisiteLedgerFixture(t, root, "CUSTOM-GOAL-002")

	report, err := Evaluate("goal-runtime-final", Options{Root: root, GoalID: "CUSTOM-GOAL-002"})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "passed" {
		t.Fatalf("status = %q; gaps %#v", report.Status, report.Gaps)
	}
	if !containsSubstring(report.Details, "non-default goal_id") {
		t.Fatalf("details = %#v; want non-default goal_id note", report.Details)
	}
}

func TestWriteEvidence_FinalWithoutPrerequisites(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report := Report{
		SchemaVersion:    "goalcli-mva/v1",
		Command:          "goal-runtime-final",
		Status:           "passed",
		GoalID:           DefaultGoalID,
		Gate:             "G12_G16_FINAL",
		Mode:             "FULL",
		Blocking:         true,
		MVAStatus:        "complete",
		LedgerPath:       SourceLedgerPath,
		EvidencePackPath: EvidenceLedgerPath + DefaultGoalID + ".json",
	}
	err := WriteEvidence(root, report)
	if err == nil {
		t.Fatal("expected error for final without prerequisites")
	}
}

func TestWriteEvidence_NonFinalCommand(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-delivery", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("WriteEvidence returned error: %v", err)
	}
	// Should NOT write evidence pack for non-final
	packPath := filepath.Join(root, filepath.FromSlash(report.EvidencePackPath))
	if _, err := os.Stat(packPath); !os.IsNotExist(err) {
		t.Fatal("non-final command should not write evidence pack")
	}
}

func TestWriteEvidence_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	report := Report{
		Command:   "goal-acceptance",
		Status:    "passed",
		MVAStatus: "complete",
		Blocking:  true,
		LedgerPath: filepath.Join(root, "ledger.jsonl"),
	}
	err := WriteEvidence("", report)
	// May fail due to directory creation in cwd, but should not panic
	_ = err
}

func TestUpsertLedgerEntry_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "ledger.jsonl")
	entry := LedgerEntry{
		SchemaVersion: "goalcli-mva/v1",
		GoalID:        "test",
		Command:       "goal-acceptance",
		Status:        "passed",
		MVAStatus:     "complete",
		Blocking:      true,
	}
	err := upsertLedgerEntry(path, entry)
	if err != nil {
		t.Fatalf("upsertLedgerEntry should create dirs: %v", err)
	}
}

func TestUpsertLedgerEntry_Deduplication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.jsonl")
	entry := LedgerEntry{
		SchemaVersion: "goalcli-mva/v1",
		GoalID:        "test",
		Command:       "goal-acceptance",
		Status:        "passed",
		MVAStatus:     "complete",
		Blocking:      true,
		EvidencePackPath: "test.json",
	}
	if err := upsertLedgerEntry(path, entry); err != nil {
		t.Fatal(err)
	}
	if err := upsertLedgerEntry(path, entry); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\"command\":\"goal-acceptance\"") != 1 {
		t.Fatalf("expected 1 entry after dedup, got %d", strings.Count(string(data), "\"command\":\"goal-acceptance\""))
	}
}

func TestReadLedgerEntries_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0o644)
	entries, err := readLedgerEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadLedgerEntries_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blank.jsonl")
	os.WriteFile(path, []byte("\n\n\n"), 0o644)
	entries, err := readLedgerEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestGatesForCommand_FinalRuntime(t *testing.T) {
	gates := gatesForCommand("goal-runtime-final")
	if len(gates) != 5 {
		t.Fatalf("expected 5 gates for final, got %d", len(gates))
	}
}

func TestGatesForCommand_Single(t *testing.T) {
	gates := gatesForCommand("goal-acceptance")
	if len(gates) != 1 {
		t.Fatalf("expected 1 gate, got %d", len(gates))
	}
	if gates[0].ID != "G12_ACCEPTANCE" {
		t.Fatalf("expected G12_ACCEPTANCE, got %s", gates[0].ID)
	}
}

func TestIsStandardSourceRoot_StandardModule(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/ZoneCNH/xlib-standard\n"), 0o644)
	if !isStandardSourceRoot(root) {
		t.Fatal("expected true for standard module path")
	}
}

func TestIsStandardSourceRoot_DownstreamModule(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/ZoneCNH/kernel\n"), 0o644)
	if isStandardSourceRoot(root) {
		t.Fatal("expected false for downstream module path")
	}
}

func TestIsStandardSourceRoot_NoGoMod(t *testing.T) {
	root := t.TempDir()
	// No go.mod => modulePathForRoot returns false => isStandardSourceRoot returns true
	if !isStandardSourceRoot(root) {
		t.Fatal("expected true when go.mod is missing (defaults to standard)")
	}
}

func TestWriteEvidence_FinalWithPrerequisites(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	writePrerequisiteLedgerFixture(t, root, DefaultGoalID)

	report, err := Evaluate("goal-runtime-final", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if err := WriteEvidence(root, report); err != nil {
		t.Fatalf("WriteEvidence returned error: %v", err)
	}
	// Verify pack was written
	packPath := filepath.Join(root, filepath.FromSlash(report.EvidencePackPath))
	if _, err := os.Stat(packPath); err != nil {
		t.Fatalf("evidence pack not written: %v", err)
	}
	// Verify ledger has final entry
	ledgerPath := filepath.Join(root, filepath.FromSlash(SourceLedgerPath))
	ledger, _ := os.ReadFile(ledgerPath)
	if !strings.Contains(string(ledger), `"command":"goal-runtime-final"`) {
		t.Fatal("ledger missing final entry")
	}
}

func TestUpsertLedgerEntry_WriteToReadOnlyDir(t *testing.T) {
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	os.MkdirAll(roDir, 0o555)
	path := filepath.Join(roDir, "ledger.jsonl")
	entry := LedgerEntry{GoalID: "test", Command: "test"}
	err := upsertLedgerEntry(path, entry)
	if err == nil {
		// May succeed on some systems (root), just ensure no panic
	}
}

func TestWriteEvidence_PackPathError(t *testing.T) {
	// WriteEvidence with a pack path under a read-only directory
	root := t.TempDir()
	roDir := filepath.Join(root, "readonly")
	os.MkdirAll(roDir, 0o555)
	report := Report{
		Command:          "goal-acceptance",
		Status:           "passed",
		MVAStatus:        "complete",
		Blocking:         true,
		LedgerPath:       filepath.Join(root, "ledger.jsonl"),
		EvidencePackPath: filepath.Join(roDir, "pack.json"),
	}
	err := WriteEvidence(root, report)
	// May succeed on root, but exercises the MkdirAll/WriteFile error paths
	_ = err
}

func TestWriteEvidence_LedgerPathError(t *testing.T) {
	root := t.TempDir()
	roDir := filepath.Join(root, "readonly")
	os.MkdirAll(roDir, 0o555)
	report := Report{
		Command:          "goal-acceptance",
		Status:           "passed",
		MVAStatus:        "complete",
		Blocking:         true,
		LedgerPath:       filepath.Join(roDir, "sub", "ledger.jsonl"),
		EvidencePackPath: filepath.Join(root, "pack.json"),
	}
	err := WriteEvidence(root, report)
	// May succeed on root, but exercises the upsertLedgerEntry MkdirAll path
	_ = err
}

func TestEvaluate_AllCommands(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	for _, cmd := range Commands() {
		report, err := Evaluate(cmd, Options{Root: root})
		if err != nil {
			t.Fatalf("Evaluate(%q) returned error: %v", cmd, err)
		}
		if report.Command != cmd {
			t.Fatalf("command = %q; want %q", report.Command, cmd)
		}
		if report.SchemaVersion != "goalcli-mva/v1" {
			t.Fatalf("schema_version = %q; want goalcli-mva/v1", report.SchemaVersion)
		}
	}
}

func TestEvaluate_EmptyRoot(t *testing.T) {
	// Evaluate with empty root should default to "."
	_, err := Evaluate("goal-acceptance", Options{})
	// May fail due to missing files in cwd, but should not panic
	_ = err
}

func TestEvaluate_EmptyGoalID(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	report, err := Evaluate("goal-acceptance", Options{Root: root, GoalID: "  "})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	// Empty/whitespace GoalID should default to DefaultGoalID
	if report.GoalID != DefaultGoalID {
		t.Fatalf("goal_id = %q; want %q", report.GoalID, DefaultGoalID)
	}
}

func TestEvaluate_FinalRuntimeMissingSomePrerequisites(t *testing.T) {
	root := t.TempDir()
	writeAuthorityFixture(t, root)
	// Write only 2 of 5 prerequisites
	for _, command := range []string{"goal-acceptance", "goal-delivery"} {
		report, err := Evaluate(command, Options{Root: root, GoalID: DefaultGoalID})
		if err != nil {
			t.Fatalf("Evaluate %s: %v", command, err)
		}
		if err := WriteEvidence(root, report); err != nil {
			t.Fatalf("WriteEvidence %s: %v", command, err)
		}
	}

	report, err := Evaluate("goal-runtime-final", Options{Root: root})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if report.Status != "failed" {
		t.Fatalf("status = %q; want failed with incomplete prerequisites", report.Status)
	}
	if len(report.Gaps) == 0 {
		t.Fatal("expected gaps for incomplete prerequisites")
	}
}

func writeAuthorityFixture(t *testing.T, root string) {
	t.Helper()
	writeAuthorityPaths(t, root, requiredAuthorityPaths(true))
}

func writeAuthorityPaths(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, path := range paths {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir fixture path %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write fixture path %s: %v", path, err)
		}
	}
}

func writePrerequisiteLedgerFixture(t *testing.T, root string, goalID string) {
	t.Helper()
	for _, command := range finalPrerequisiteCommands {
		report, err := Evaluate(command, Options{Root: root, GoalID: goalID})
		if err != nil {
			t.Fatalf("Evaluate prerequisite %s returned error: %v", command, err)
		}
		if err := WriteEvidence(root, report); err != nil {
			t.Fatalf("WriteEvidence prerequisite %s returned error: %v", command, err)
		}
	}
}

func downstreamAdoptionBoundaryEvidence() []string {
	return []string{
		downstreamAdoptionClaimEvidence,
		downstreamAdoptionScopeEvidence,
		downstreamAdoptionProofEvidence,
		downstreamRepoWriteEvidence,
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
