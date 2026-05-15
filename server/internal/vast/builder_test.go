package vast

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func sampleInput() InlineInput {
	return InlineInput{
		AdID:          "00000000-0000-0000-0000-000000000003",
		Title:         "Demo Pre-roll",
		ImpressionURL: "http://localhost:8088/track?event=impression&ad_id=00000000-0000-0000-0000-000000000003",
		MediaURL:      "http://localhost:9000/openadsource/seed/big_buck_bunny_720p_1mb.mp4",
		MediaMime:     "video/mp4",
		MediaWidth:    1280,
		MediaHeight:   720,
		MediaBitrate:  1500,
		MediaDuration: "00:00:10",
		LandingURL:    "https://example.com/landing",
	}
}

func TestBuildInline_Golden(t *testing.T) {
	got, err := BuildInline(sampleInput())
	if err != nil {
		t.Fatalf("BuildInline: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "golden_inline.xml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(normalizeNewlines(got), normalizeNewlines(want)) {
		t.Errorf("inline output drifted from golden.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestBuildEmpty_Golden(t *testing.T) {
	got, err := BuildEmpty()
	if err != nil {
		t.Fatalf("BuildEmpty: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "golden_empty.xml"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(normalizeNewlines(got), normalizeNewlines(want)) {
		t.Errorf("empty output drifted from golden.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestBuildEmpty_XSD validates that the no-fill VAST passes the vendored IAB
// VAST 4.2 XSD via xmllint. Strict full-document XSD compliance (AdServingId
// requirement, alphabetical ordering inside InLine, IAB namespace) is a
// Phase 5 hardening task; for Phase 1 we exercise xmllint on the empty
// response only — that's enough to prove the validator wiring works.
// Skips cleanly when xmllint isn't available.
func TestBuildEmpty_XSD(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not on PATH; skipping XSD validation (install libxml2-utils to enable)")
	}

	xsd, err := os.ReadFile(filepath.Join("testdata", "vast_4.2.xsd"))
	if err != nil {
		t.Skip("vendored XSD missing:", err)
	}
	relaxed := relaxXSD(xsd)

	tmp := t.TempDir()
	xsdPath := filepath.Join(tmp, "vast.xsd")
	if err := os.WriteFile(xsdPath, relaxed, 0o644); err != nil {
		t.Fatalf("write relaxed xsd: %v", err)
	}

	out, err := BuildEmpty()
	if err != nil {
		t.Fatalf("BuildEmpty: %v", err)
	}
	xmlPath := filepath.Join(tmp, "out.xml")
	if err := os.WriteFile(xmlPath, out, 0o644); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	cmd := exec.Command(xmllint, "--noout", "--nonet", "--schema", xsdPath, xmlPath)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("xmllint validation failed: %v\noutput:\n%s\n--- xml ---\n%s", err, combined, out)
	}
}

// relaxXSD makes the upstream IAB schema accept namespace-less VAST documents
// like the ones this package emits. It removes the IAB `targetNamespace` plus
// the matching prefix on every internal type reference (`type="vast:Foo"` →
// `type="Foo"`), and drops `elementFormDefault="qualified"`.
func relaxXSD(xsd []byte) []byte {
	out := bytes.ReplaceAll(xsd, []byte(`targetNamespace="http://www.iab.com/VAST"`), nil)
	out = bytes.ReplaceAll(out, []byte(`xmlns:vast="http://www.iab.com/VAST"`), nil)
	out = bytes.ReplaceAll(out, []byte(`elementFormDefault="qualified"`), nil)
	out = bytes.ReplaceAll(out, []byte(`vast:`), nil)
	return out
}

func normalizeNewlines(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
