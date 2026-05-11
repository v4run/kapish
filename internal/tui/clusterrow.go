package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/v4run/kapish/internal/capi"
)

func clusterKey(c capi.Cluster) string { return c.Namespace + "/" + c.Name }

func sortClusters(in []capi.Cluster) []capi.Cluster {
	out := append([]capi.Cluster(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func filterClusters(in []capi.Cluster, q string) []capi.Cluster {
	if q == "" {
		return append([]capi.Cluster(nil), in...)
	}
	out := make([]capi.Cluster, 0, len(in))
	for _, c := range in {
		if strings.Contains(c.Name, q) || strings.Contains(c.Namespace, q) {
			out = append(out, c)
		}
	}
	return out
}

func ageString(created, now time.Time) string {
	if created.IsZero() {
		return "?"
	}
	d := now.Sub(created)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
