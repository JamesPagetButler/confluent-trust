package validate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:generate cp ../../schema/inventory.schema.json schema.json

//go:embed schema.json
var rawSchema []byte

const schemaURI = "https://github.com/JamesPagetButler/confluent-trust/schema/inventory.schema.json"

var (
	compileOnce sync.Once
	compiled    *jsonschema.Schema
	compileErr  error
)

func loadSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		var doc any
		if err := json.Unmarshal(rawSchema, &doc); err != nil {
			compileErr = fmt.Errorf("validate: parse embedded schema: %w", err)
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource(schemaURI, doc); err != nil {
			compileErr = fmt.Errorf("validate: register schema: %w", err)
			return
		}
		s, err := c.Compile(schemaURI)
		if err != nil {
			compileErr = fmt.Errorf("validate: compile schema: %w", err)
			return
		}
		compiled = s
	})
	return compiled, compileErr
}

// Inventory validates the given JSON document against the embedded
// inventory schema. It returns nil on success or a wrapped
// *jsonschema.ValidationError on failure.
func Inventory(data []byte) error {
	s, err := loadSchema()
	if err != nil {
		return err
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("validate: parse document: %w", err)
	}
	if err := s.Validate(doc); err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	return nil
}

// SchemaVersion peeks at the top-level schema_version field without
// requiring the full document to be valid. Returns "v0.1" if absent
// (legacy default).
func SchemaVersion(data []byte) (string, error) {
	var head struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return "", fmt.Errorf("validate: peek schema_version: %w", err)
	}
	v := strings.TrimSpace(head.SchemaVersion)
	if v == "" {
		return "v0.1", nil
	}
	return v, nil
}
