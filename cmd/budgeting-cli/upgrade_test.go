package main

import (
	"strings"
	"testing"
)

func TestArchiveName(t *testing.T) {
	got := archiveName("budgeting-cli", "0.3.3", "linux", "amd64")
	want := "budgeting-cli_0.3.3_linux_amd64.tar.gz"
	if got != want {
		t.Fatalf("archiveName = %q, want %q", got, want)
	}
}

func TestParseChecksums(t *testing.T) {
	raw := `abc123  budgeting-cli_0.3.3_linux_amd64.tar.gz
def456  budgeting-cli_0.3.3_darwin_arm64.tar.gz

ff00  checksums.txt
`
	m, err := parseChecksums(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parseChecksums: %v", err)
	}
	if m["budgeting-cli_0.3.3_linux_amd64.tar.gz"] != "abc123" {
		t.Errorf("linux amd64 = %q, want abc123", m["budgeting-cli_0.3.3_linux_amd64.tar.gz"])
	}
	if m["budgeting-cli_0.3.3_darwin_arm64.tar.gz"] != "def456" {
		t.Errorf("darwin arm64 = %q, want def456", m["budgeting-cli_0.3.3_darwin_arm64.tar.gz"])
	}
	if m["checksums.txt"] != "ff00" {
		t.Errorf("checksums.txt = %q, want ff00", m["checksums.txt"])
	}
	if len(m) != 3 {
		t.Errorf("got %d entries, want 3 (blanks must be skipped)", len(m))
	}
}
