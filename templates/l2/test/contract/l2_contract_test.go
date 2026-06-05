package contract

import (
	"os"
	"strings"
	"testing"
)

// TestL2ContractPackDeclaration validates the local template manifest shape.
// Downstream L2 repositories replace this guard with testkitx contract-pack execution.
func TestL2ContractPackDeclaration(t *testing.T) {
	manifest, err := os.ReadFile("../../.agent/l2-capabilities.yaml")
	if err != nil {
		t.Fatalf("read template L2 capability manifest: %v", err)
	}

	text := string(manifest)
	requiredSnippets := []string{
		`schema_version: "1.0"`,
		"layer: L2",
		"adapter:",
		"capabilities:",
		"contract_packs:",
		"evidence:",
		"output_dir: .agent/evidence/l2",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("template manifest missing required snippet %q", snippet)
		}
	}
	forbiddenKeys := []string{
		"provider_endpoint",
		"provider_credentials",
		"password",
		"secret",
		"token",
	}
	for _, key := range forbiddenKeys {
		snippet := key + ":"
		if strings.Contains(text, snippet) {
			t.Fatalf("template manifest contains forbidden provider-boundary snippet %q", snippet)
		}
	}
}
