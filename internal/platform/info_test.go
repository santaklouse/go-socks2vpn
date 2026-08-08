package platform

import "testing"

func TestDescription(t *testing.T) {
	got := (Info{OS: "linux", Arch: "amd64", AMD64V3: true, Musl: true}).Description()
	if got != "linux/amd64 (AVX2) (musl)" {
		t.Fatalf("Description() = %q", got)
	}
}
