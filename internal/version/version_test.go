package version

import (
	"strings"
	"testing"
)

func TestStringFallsBackToDev(t *testing.T) {
	// In a unit-test build, BuildInfo's main module version is usually "(devel)"
	// and there's no VCS info, so String() should fall back to ldflags Version
	// (default "dev").
	got := String()
	if got == "" {
		t.Fatalf("String() returned empty")
	}
	// dev or a 7-char rev or a semver — all acceptable. We just verify it's not empty
	// and doesn't contain whitespace.
	if strings.ContainsAny(got, " \t\n") {
		t.Fatalf("String() = %q, must not contain whitespace", got)
	}
}

func TestLongIncludesString(t *testing.T) {
	long := Long()
	if !strings.Contains(long, String()) {
		t.Fatalf("Long() = %q, expected to contain String() = %q", long, String())
	}
	if !strings.HasPrefix(long, "kapish ") {
		t.Fatalf("Long() = %q, expected to start with 'kapish '", long)
	}
}
