package assets

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestExtractExact(t *testing.T) {
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	f, err := w.Create("wintun/bin/amd64/wintun.dll")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("dll")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := extractExact(archive.Bytes(), "wintun/bin/amd64/wintun.dll")
	if err != nil || string(got) != "dll" {
		t.Fatalf("extractExact() = %q, %v", got, err)
	}
	sum := sha256.Sum256(archive.Bytes())
	if err := verifySHA256(archive.Bytes(), fmt.Sprintf("%x", sum)); err != nil {
		t.Fatal(err)
	}
}

func TestSHA256Mismatch(t *testing.T) {
	if err := verifySHA256([]byte("bad"), fmt.Sprintf("%064x", 1)); err == nil {
		t.Fatal("digest mismatch unexpectedly accepted")
	}
}
