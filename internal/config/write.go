package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

// WriteToFile writes c back to path. If the file already exists, the existing
// YAML's comments and key ordering are preserved by mutating a yaml.Node tree
// in place rather than re-marshaling from the Go struct. If the file does not
// exist, a fresh YAML document is generated from c.
//
// File mode is 0600; parent directories are created with 0700 if missing.
func WriteToFile(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}

	lockPath := path + ".lock"
	lk := flock.New(lockPath)
	if err := lk.Lock(); err != nil {
		return fmt.Errorf("config: lock %s: %w", lockPath, err)
	}
	defer func() {
		_ = lk.Unlock()
		_ = os.Remove(lockPath)
	}()

	out, err := renderYAML(path, c)
	if err != nil {
		return err
	}

	// Atomic write via temp file + rename.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kapish-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: write %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("config: chmod %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("config: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// renderYAML produces the bytes to write. If path exists, it loads the
// existing YAML as a yaml.Node tree, applies the values from c onto the
// tree (preserving comments and ordering for keys that already exist),
// and serializes. If path does not exist, it marshals c directly.
func renderYAML(path string, c Config) ([]byte, error) {
	existing, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// No existing file: marshal directly.
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(c); err != nil {
			return nil, fmt.Errorf("config: encode: %w", err)
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(existing, &root); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	// Re-marshal c into a yaml.Node so we can patch values back into root.
	var fresh yaml.Node
	if err := fresh.Encode(c); err != nil {
		return nil, fmt.Errorf("config: encode patch: %w", err)
	}

	// root is DocumentNode with .Content[0] = mapping; same for fresh.
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		patchMappingValues(root.Content[0], &fresh)
	} else {
		// Empty file or weird shape — replace wholesale.
		root = fresh
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, fmt.Errorf("config: encode: %w", err)
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

// patchMappingValues walks the existing mapping node and replaces values
// found in fresh, preserving comments on the existing node. Keys present
// only in fresh are appended. Keys present only in existing are kept
// (so user-only keys aren't deleted).
func patchMappingValues(existing, fresh *yaml.Node) {
	if existing.Kind != yaml.MappingNode || fresh.Kind != yaml.MappingNode {
		// types diverged — replace existing's content with fresh's
		existing.Content = fresh.Content
		return
	}

	idx := indexMapping(existing)
	for i := 0; i < len(fresh.Content); i += 2 {
		key := fresh.Content[i].Value
		newVal := fresh.Content[i+1]
		if pos, ok := idx[key]; ok {
			existingVal := existing.Content[pos+1]
			switch {
			case existingVal.Kind == yaml.MappingNode && newVal.Kind == yaml.MappingNode:
				patchMappingValues(existingVal, newVal)
			case existingVal.Kind == yaml.SequenceNode && newVal.Kind == yaml.SequenceNode:
				patchSequenceValues(existingVal, newVal)
			case existingVal.Kind == yaml.ScalarNode && newVal.Kind == yaml.ScalarNode:
				// Update value in place, preserving comments on the existing node.
				existingVal.Value = newVal.Value
				existingVal.Tag = newVal.Tag
				existingVal.Style = newVal.Style
			default:
				// Types diverged — replace the value node wholesale.
				existing.Content[pos+1] = newVal
			}
		} else {
			existing.Content = append(existing.Content, fresh.Content[i], newVal)
		}
	}
}

// patchSequenceValues patches a sequence node in place. For sequences of
// mappings that have a "name" key, items are matched by name so that
// per-entry comments are preserved. For all other sequences (scalars, etc.)
// items are matched positionally; extra existing items are kept, extra fresh
// items are appended.
func patchSequenceValues(existing, fresh *yaml.Node) {
	if existing.Kind != yaml.SequenceNode || fresh.Kind != yaml.SequenceNode {
		existing.Content = fresh.Content
		return
	}

	// Check if items are named mappings (have a "name" key).
	if len(fresh.Content) > 0 && fresh.Content[0].Kind == yaml.MappingNode {
		freshByName := make(map[string]*yaml.Node)
		for _, item := range fresh.Content {
			if item.Kind == yaml.MappingNode {
				if n := mappingScalarValue(item, "name"); n != "" {
					freshByName[n] = item
				}
			}
		}
		if len(freshByName) > 0 {
			// Named mapping sequence — patch by name.
			matched := make(map[string]bool)
			for _, existingItem := range existing.Content {
				if existingItem.Kind == yaml.MappingNode {
					n := mappingScalarValue(existingItem, "name")
					if freshItem, ok := freshByName[n]; ok {
						patchMappingValues(existingItem, freshItem)
						matched[n] = true
					}
				}
			}
			// Append new items not found in existing.
			for _, freshItem := range fresh.Content {
				if freshItem.Kind == yaml.MappingNode {
					n := mappingScalarValue(freshItem, "name")
					if !matched[n] {
						existing.Content = append(existing.Content, freshItem)
					}
				}
			}
			return
		}
	}

	// Positional patch for non-named sequences.
	for i, freshItem := range fresh.Content {
		if i < len(existing.Content) {
			existingItem := existing.Content[i]
			switch {
			case existingItem.Kind == yaml.MappingNode && freshItem.Kind == yaml.MappingNode:
				patchMappingValues(existingItem, freshItem)
			case existingItem.Kind == yaml.ScalarNode && freshItem.Kind == yaml.ScalarNode:
				existingItem.Value = freshItem.Value
				existingItem.Tag = freshItem.Tag
				existingItem.Style = freshItem.Style
			default:
				existing.Content[i] = freshItem
			}
		} else {
			existing.Content = append(existing.Content, freshItem)
		}
	}
}

// mappingScalarValue returns the scalar value of the given key in a mapping
// node, or "" if not found.
func mappingScalarValue(n *yaml.Node, key string) string {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key && n.Content[i+1].Kind == yaml.ScalarNode {
			return n.Content[i+1].Value
		}
	}
	return ""
}

func indexMapping(n *yaml.Node) map[string]int {
	out := make(map[string]int, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		out[n.Content[i].Value] = i
	}
	return out
}
