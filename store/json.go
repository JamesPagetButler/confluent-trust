package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JamesPagetButler/confluent-trust/internal/validate"
	"github.com/JamesPagetButler/confluent-trust/model"
)

// supported schema versions for the JSON inventory format.
const (
	SchemaV01 = "v0.1"
	SchemaV02 = "v0.2"
)

// LoadInventory reads, schema-validates, and unmarshals an inventory file.
// Binary path_a / path_b confluences (legacy v0.1 shape) are normalized into
// the N-ary Paths form on load. The returned Inventory.SchemaVersion is set
// to the dispatched version.
func LoadInventory(path string) (model.Inventory, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return model.Inventory{}, fmt.Errorf("store/json: read %s: %w", path, err)
	}

	version, err := validate.SchemaVersion(data)
	if err != nil {
		return model.Inventory{}, fmt.Errorf("store/json: %s: %w", path, err)
	}
	switch version {
	case SchemaV01, SchemaV02:
		// supported
	default:
		return model.Inventory{}, fmt.Errorf("store/json: %s: unsupported schema version %q", path, version)
	}

	if err := validate.Inventory(data); err != nil {
		return model.Inventory{}, fmt.Errorf("store/json: %s: %w", path, err)
	}

	var inv model.Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return model.Inventory{}, fmt.Errorf("store/json: %s: unmarshal: %w", path, err)
	}

	if inv.SchemaVersion == "" {
		inv.SchemaVersion = version
	}
	inv.NormalizeConfluences()

	if err := inv.Validate(); err != nil {
		return model.Inventory{}, fmt.Errorf("store/json: %s: %w", path, err)
	}

	return inv, nil
}

// SaveInventory writes an inventory to disk as indented JSON. The write is
// atomic: data is staged at <path>.tmp and then renamed.
func SaveInventory(inv model.Inventory, path string) error {
	if inv.SchemaVersion == "" {
		inv.SchemaVersion = SchemaV02
	}
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("store/json: marshal: %w", err)
	}

	cleaned := filepath.Clean(path)
	tmp := cleaned + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil { // #nosec G306 -- inventory files are non-sensitive
		return fmt.Errorf("store/json: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, cleaned); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("store/json: rename %s -> %s: %w", tmp, cleaned, err)
	}
	return nil
}

// LoadMultiple loads each path with LoadInventory. On any error the slice
// returned is empty and the wrapped error includes every failure.
func LoadMultiple(paths []string) ([]model.Inventory, error) {
	out := make([]model.Inventory, 0, len(paths))
	var errs []error
	for _, p := range paths {
		inv, err := LoadInventory(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, inv)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}
