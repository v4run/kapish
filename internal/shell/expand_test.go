package shell

import (
	"os"
	"testing"
)

func TestExpandCwd(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	t.Setenv("HOME", home) // keep os.ExpandEnv($HOME) consistent with UserHomeDir on this platform
	t.Setenv("KAPISH_TEST_DIR", "/tmp/k")

	cases := []struct{ in, want string }{
		{"", ""},
		{"~", home},
		{"~/work", home + "/work"},
		{"$HOME/work", home + "/work"},
		{"${HOME}/work", home + "/work"},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"$KAPISH_TEST_DIR/sub", "/tmp/k/sub"},
		{"~unknownuser/x", "~unknownuser/x"}, // bare ~user form left unchanged
	}
	for _, c := range cases {
		if got := expandCwd(c.in); got != c.want {
			t.Errorf("expandCwd(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCdLine(t *testing.T) {
	if got := cdLine(""); got != "" {
		t.Errorf("cdLine(\"\") = %q, want empty", got)
	}
	home, _ := os.UserHomeDir()
	t.Setenv("HOME", home)
	if got := cdLine("~/with 'quote"); got != "cd "+posixSingleQuote(home+"/with 'quote")+"\n" {
		t.Errorf("cdLine quoting wrong: %q", got)
	}
}
