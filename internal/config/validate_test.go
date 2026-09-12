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

func TestValidateMgmt_EmptyAccepted(t *testing.T) {
	errs := validateMgmt(ManagementClustersConfig{})
	assert.Empty(t, errs)
}

func TestValidateMgmt_DuplicateNames(t *testing.T) {
	m := ManagementClustersConfig{
		Entries: []ManagementClusterEntry{
			{Name: "a"},
			{Name: "a"},
		},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "duplicate")
}

func TestValidateMgmt_EmptyEntryName(t *testing.T) {
	m := ManagementClustersConfig{
		Entries: []ManagementClusterEntry{{Name: ""}},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "name")
}

func TestValidateMgmt_CurrentMustReferenceEntry(t *testing.T) {
	m := ManagementClustersConfig{
		Current: "missing",
		Entries: []ManagementClusterEntry{{Name: "a"}},
	}
	errs := validateMgmt(m)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "current")
	assert.Contains(t, errs[0].Error(), "missing")
}

func TestValidateMgmt_HappyPath(t *testing.T) {
	m := ManagementClustersConfig{
		Current: "a",
		Entries: []ManagementClusterEntry{
			{Name: "a"},
			{Name: "b"},
		},
	}
	errs := validateMgmt(m)
	assert.Empty(t, errs)
}

func TestValidate_HappyDefaults(t *testing.T) {
	require.NoError(t, Validate(Defaults()))
}

func TestValidate_AggregatesErrors(t *testing.T) {
	c := Defaults()
	c.Shell.Env = map[string]string{"bad-key": "x"}        // env error
	c.Shell.Aliases = map[string]string{"1bad": "kubectl"} // alias error
	c.Shell.Prompt = "{nope}"                              // prompt error
	c.ManagementClusters.Current = "nada"                  // mgmt error

	err := Validate(c)
	require.Error(t, err)
	// Validate should surface all errors at once, joined.
	msg := err.Error()
	assert.Contains(t, msg, "env key")
	assert.Contains(t, msg, "alias name")
	assert.Contains(t, msg, "{nope}")
	assert.Contains(t, msg, "current")
}

func validEnrichment() EnrichmentConfig {
	return EnrichmentConfig{
		Providers: []EnrichProvider{{
			Name: "cost", Type: "shell", Command: []string{"kapish-cost", "--all"},
			Format: "json", Key: "clusterId", RefreshIntervalSec: 300, TimeoutSec: 15,
		}},
		Columns: []EnrichColumn{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}},
	}
}

func TestValidate_EnrichmentOK(t *testing.T) {
	c := Defaults()
	c.Enrichment = validEnrichment()
	assert.NoError(t, Validate(c))
}

func TestValidate_EnrichmentRejections(t *testing.T) {
	cases := map[string]func(*EnrichmentConfig){
		"duplicate provider": func(e *EnrichmentConfig) { e.Providers = append(e.Providers, e.Providers[0]) },
		"bad type":           func(e *EnrichmentConfig) { e.Providers[0].Type = "http" },
		"bad format":         func(e *EnrichmentConfig) { e.Providers[0].Format = "tsv" },
		"empty command":      func(e *EnrichmentConfig) { e.Providers[0].Command = nil },
		"empty key":          func(e *EnrichmentConfig) { e.Providers[0].Key = "" },
		"zero interval":      func(e *EnrichmentConfig) { e.Providers[0].RefreshIntervalSec = 0 },
		"zero timeout":       func(e *EnrichmentConfig) { e.Providers[0].TimeoutSec = 0 },
		"unknown col prov":   func(e *EnrichmentConfig) { e.Columns[0].Provider = "nope" },
		"empty field":        func(e *EnrichmentConfig) { e.Columns[0].Field = "" },
		"bad joinOn":         func(e *EnrichmentConfig) { e.Columns[0].JoinOn = "label" },
		"empty header":       func(e *EnrichmentConfig) { e.Columns[0].Header = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := Defaults()
			e := validEnrichment()
			mutate(&e)
			c.Enrichment = e
			assert.Error(t, Validate(c))
		})
	}
}
