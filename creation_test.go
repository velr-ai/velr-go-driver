package velr

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCreateInlineEdgeProperties(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	options := CreateOptions{InlineEdgeProperties: []string{"payload"}}
	db, err := Create(path, options)
	if err != nil && os.Getenv("VELR_RUNTIME_PATH") == "" && strings.Contains(err.Error(), "does not expose velr_create") {
		t.Skip("bundled runtime predates the creation API")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if names, err := db.InlineEdgeProperties(); err != nil || !reflect.DeepEqual(names, options.InlineEdgeProperties) {
		t.Fatalf("inline properties = %v, %v", names, err)
	}
	if other, err := Create(path, options); err == nil {
		other.Close()
		t.Fatal("Create overwrote an existing database")
	}
	large := strings.Repeat("x", 129)
	if err := db.RunWithParams(
		"CREATE (a)-[:Inline {payload: 'short'}]->(b), (a)-[:Spill {payload: $large}]->(b)",
		Params{"large": large},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	readonly, err := OpenReadonly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer readonly.Close()
	if names, err := readonly.InlineEdgeProperties(); err != nil || !reflect.DeepEqual(names, options.InlineEdgeProperties) {
		t.Fatalf("reopened inline properties = %v, %v", names, err)
	}
	got, err := readonly.Query("MATCH ()-[r:Spill]->() RETURN r.payload AS payload")
	want := []map[string]any{{"payload": large}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("spill result = %v, %v", got, err)
	}
	memory, err := CreateInMemory(CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	if names, err := memory.InlineEdgeProperties(); err != nil || len(names) != 0 {
		t.Fatalf("default inline properties = %v, %v", names, err)
	}
}
