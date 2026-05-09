package config

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateShell_EmptyCommandOK(t *testing.T) {
	// Empty command means "use $SHELL at spawn time" — accepted.
	errs := validateShell(ShellConfig{Command: ""})
	assert.Empty(t, errs)
}

func TestValidateShell_KnownShellInPath(t *testing.T) {
	// Pick a shell that is virtually always installed.
	bash, err := exec.LookPath("bash")
	require.NoError(t, err, "this test assumes bash is on PATH")

	errs := validateShell(ShellConfig{Command: bash})
	assert.Empty(t, errs)
}

func TestValidateShell_UnsupportedBasename(t *testing.T) {
	// /bin/ksh has a basename `ksh` which v1 doesn't support.
	errs := validateShell(ShellConfig{Command: "/usr/bin/ksh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "unsupported shell")
}

func TestValidateShell_NotInPath(t *testing.T) {
	errs := validateShell(ShellConfig{Command: "/totally/not/a/real/zsh"})
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "not found")
}

func TestValidateEnv_KeyRules(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"upper letters", "KUBECONFIG", false},
		{"with underscore", "AWS_REGION", false},
		{"with digits", "FOO123", false},
		{"leading underscore", "_FOO", false},
		{"leading digit", "1FOO", true},
		{"lowercase", "foo", true},
		{"empty", "", true},
		{"spaces", "FOO BAR", true},
		{"hyphen", "FOO-BAR", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{tc.key: "v"}
			errs := validateEnv(env)
			if tc.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateAliases_NameRules(t *testing.T) {
	cases := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"simple", "k", false},
		{"with underscore", "k_get", false},
		{"with digits", "k1", false},
		{"with hyphen", "k-get", false},
		{"leading digit", "1k", true},
		{"with space", "k get", true},
		{"with equals", "k=v", true},
		{"empty", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := map[string]string{tc.alias: "kubectl"}
			errs := validateAliases(a)
			if tc.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidatePrompt_KnownTokens(t *testing.T) {
	cases := []string{
		"",
		"$ ",
		"[{cluster}] ",
		"[{cluster}/{ns}] {ctx} {time} ",
		"({provider}) > ",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			errs := validatePrompt(p)
			assert.Empty(t, errs, "should accept: %q", p)
		})
	}
}

func TestValidatePrompt_UnknownToken(t *testing.T) {
	errs := validatePrompt("[{cluster}] {region} ")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "{region}")
}

func TestValidatePrompt_MalformedToken(t *testing.T) {
	errs := validatePrompt("hello {cluster ")
	require.NotEmpty(t, errs)
}
