//go:build genvast

// Run with: go test -tags genvast ./internal/vast -run TestGenerateGolden
// One-off helper to (re)create the golden files in testdata/. Excluded from
// the normal test build by the genvast build tag.
package vast

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateGolden(t *testing.T) {
	inline, err := BuildInline(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("testdata", "golden_inline.xml"), inline, 0o644); err != nil {
		t.Fatal(err)
	}

	empty, err := BuildEmpty()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("testdata", "golden_empty.xml"), empty, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Logf("regenerated golden_inline.xml (%d bytes) and golden_empty.xml (%d bytes)", len(inline), len(empty))
}
