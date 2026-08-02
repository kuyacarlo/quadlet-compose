package unit

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Opts holds all configuration needed to render a systemd unit file.
type Opts struct {
	Name             string
	ComposePath      string // absolute path to compose file
	ComposeFilename  string
	WorkingDir       string
	ExternalNetworks []string
	After            []string
	EnvFiles         []string
	Timeout          int
	ComposeBin       string
	GeneratedAt      time.Time
}

// UnitFileName returns the systemd unit file name for a given service name.
func UnitFileName(name string) string {
	return name + ".service"
}

// podmanBin derives the podman binary path from ComposeBin.
// ComposeBin is expected to be something like "/usr/bin/podman compose"
// and we want the first word: "/usr/bin/podman".
func podmanBin(composeBin string) string {
	parts := strings.Fields(composeBin)
	if len(parts) == 0 {
		return "/usr/bin/podman"
	}
	return parts[0]
}

// templateData is the internal struct passed to the template.
type templateData struct {
	Name             string
	ComposePath      string
	GeneratedAt      string
	Description      string
	AfterList        string
	RequiresList     string
	WorkingDir       string
	ExternalNetworks []string
	PodmanBin        string
	EnvFiles         []string
	ComposeBin       string
	ComposeFilename  string
	Timeout          int
}

const unitTemplate = `# Managed by quadlet-compose
# Source: {{.ComposePath}}
# Generated: {{.GeneratedAt}}

[Unit]
Description={{.Description}}
After={{.AfterList}}
Requires={{.RequiresList}}

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory={{.WorkingDir}}
{{- range .ExternalNetworks}}
ExecStartPre=/bin/sh -c '{{$.PodmanBin}} network exists {{.}} || {{$.PodmanBin}} network create {{.}}'
{{- end}}
{{- range .EnvFiles}}
EnvironmentFile={{.}}
{{- end}}
ExecStart={{.ComposeBin}} -f {{.ComposeFilename}} up -d
ExecStop={{.ComposeBin}} -f {{.ComposeFilename}} down
ExecReload={{.ComposeBin}} -f {{.ComposeFilename}} up -d --force-recreate
Restart=on-failure
TimeoutStartSec={{.Timeout}}

[Install]
WantedBy=default.target
`

// Generate renders a systemd unit file from the given options.
func Generate(opts Opts) string {
	// Build After and Requires lists with deduplication of network-online.target.
	afterSet := make(map[string]struct{})
	afterSet["network-online.target"] = struct{}{}
	for _, a := range opts.After {
		afterSet[a] = struct{}{}
	}

	// Build ordered after list: network-online.target first, then others sorted.
	afterItems := []string{"network-online.target"}
	for a := range afterSet {
		if a != "network-online.target" {
			afterItems = append(afterItems, a)
		}
	}

	afterList := strings.Join(afterItems, " ")
	requiresList := afterList

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120
	}

	composeBin := opts.ComposeBin
	if composeBin == "" {
		composeBin = "/usr/bin/podman compose"
	}

	data := templateData{
		Name:             opts.Name,
		ComposePath:      opts.ComposePath,
		GeneratedAt:      opts.GeneratedAt.Format(time.RFC3339),
		Description:      fmt.Sprintf("quadlet-compose: %s", opts.Name),
		AfterList:        afterList,
		RequiresList:     requiresList,
		WorkingDir:       opts.WorkingDir,
		ExternalNetworks: opts.ExternalNetworks,
		PodmanBin:        podmanBin(composeBin),
		EnvFiles:         opts.EnvFiles,
		ComposeBin:       composeBin,
		ComposeFilename:  opts.ComposeFilename,
		Timeout:          timeout,
	}

	tmpl := template.Must(template.New("unit").Parse(unitTemplate))
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("unit template execution failed: %v", err))
	}
	return buf.String()
}
