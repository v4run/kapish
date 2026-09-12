// Package config defines kapish's configuration types, defaults, and the
// load/validate/persist pipeline. The merge order is:
//
//	built-in Defaults() < config file < env vars < command-line flags
package config

// Config is the top-level kapish configuration.
type Config struct {
	ManagementClusters ManagementClustersConfig `yaml:"managementClusters" json:"managementClusters"`
	Shell              ShellConfig              `yaml:"shell"              json:"shell"`
	UI                 UIConfig                 `yaml:"ui"                 json:"ui"`
	Web                WebConfig                `yaml:"web"                json:"web"`
	Enrichment         EnrichmentConfig         `yaml:"enrichment,omitempty" json:"enrichment,omitempty"`
}

type ManagementClustersConfig struct {
	Current string                   `yaml:"current,omitempty" json:"current,omitempty"`
	Entries []ManagementClusterEntry `yaml:"entries,omitempty" json:"entries,omitempty"`
}

type ManagementClusterEntry struct {
	Name       string `yaml:"name"                json:"name"`
	Kubeconfig string `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty"`
	Context    string `yaml:"context,omitempty"    json:"context,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"  json:"namespace,omitempty"`
}

type ShellConfig struct {
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Cwd     string            `yaml:"cwd,omitempty"     json:"cwd,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Prompt  string            `yaml:"prompt,omitempty"  json:"prompt,omitempty"`
}

type UIConfig struct {
	Theme              string `yaml:"theme"              json:"theme"`
	RefreshIntervalSec int    `yaml:"refreshIntervalSec" json:"refreshIntervalSec"`
	OneShot            bool   `yaml:"oneShot"            json:"oneShot"`
}

type WebConfig struct {
	DefaultPort int    `yaml:"defaultPort"  json:"defaultPort"`
	OpenBrowser bool   `yaml:"openBrowser"  json:"openBrowser"`
	BindAddr    string `yaml:"bindAddr"     json:"bindAddr"`
}

// EnrichmentConfig configures extra cluster-list columns joined from providers.
type EnrichmentConfig struct {
	Providers []EnrichProvider `yaml:"providers,omitempty" json:"providers,omitempty"`
	Columns   []EnrichColumn   `yaml:"columns,omitempty"   json:"columns,omitempty"`
}

// EnrichProvider is one enrichment data source. Only type "shell" is supported.
type EnrichProvider struct {
	Name               string   `yaml:"name"               json:"name"`
	Type               string   `yaml:"type"               json:"type"`
	Command            []string `yaml:"command,omitempty"  json:"command,omitempty"`
	Format             string   `yaml:"format,omitempty"   json:"format,omitempty"`
	Key                string   `yaml:"key,omitempty"      json:"key,omitempty"`
	RefreshIntervalSec int      `yaml:"refreshIntervalSec,omitempty" json:"refreshIntervalSec,omitempty"`
	TimeoutSec         int      `yaml:"timeoutSec,omitempty" json:"timeoutSec,omitempty"`
}

// EnrichColumn is one configured column reading a field from a provider row.
type EnrichColumn struct {
	Header   string `yaml:"header"          json:"header"`
	Provider string `yaml:"provider"        json:"provider"`
	Field    string `yaml:"field"           json:"field"`
	JoinOn   string `yaml:"joinOn"          json:"joinOn"`
	Width    int    `yaml:"width,omitempty" json:"width,omitempty"`
}

// Defaults returns a fresh built-in Config. Each call returns an
// independent copy — callers can mutate the result freely.
func Defaults() Config {
	return Config{
		ManagementClusters: ManagementClustersConfig{
			Current: "",
			Entries: nil,
		},
		Shell: ShellConfig{
			Command: "",
			Cwd:     "",
			Env:     map[string]string{},
			Aliases: map[string]string{},
			Prompt:  "[{cluster}] ",
		},
		UI: UIConfig{
			Theme:              "dark",
			RefreshIntervalSec: 30,
			OneShot:            false,
		},
		Web: WebConfig{
			DefaultPort: 0,
			OpenBrowser: true,
			BindAddr:    "127.0.0.1",
		},
	}
}
