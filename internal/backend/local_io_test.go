package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newLocalBackendIOTest(t *testing.T) *LocalBackend {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "io.db")
	be, err := NewLocalBackend(dbPath, "")
	if err != nil {
		t.Fatalf("NewLocalBackend: %v", err)
	}
	be.now = func() time.Time {
		return time.Date(2100, 1, 2, 10, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { _ = be.Close() })
	return be
}

func TestLocalBackendImportYAMLAndExportCSV(t *testing.T) {
	ctx := context.Background()
	be := newLocalBackendIOTest(t)

	yamlPath := filepath.Join(t.TempDir(), "import.yaml")
	yamlContent := `deck: Imported
cards:
  - front: Hallo
    back: hello
    tags: [greeting]
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	n, err := be.ImportYAML(ctx, yamlPath)
	if err != nil {
		t.Fatalf("ImportYAML: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported=%d want=1", n)
	}

	exportPath := filepath.Join(t.TempDir(), "export.csv")
	exported, err := be.ExportCSV(ctx, exportPath)
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if exported != 1 {
		t.Fatalf("exported=%d want=1", exported)
	}
	b, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !strings.Contains(string(b), "Imported,Hallo,hello") {
		t.Fatalf("unexpected export content: %s", string(b))
	}
}

func TestLocalBackendImportCSV(t *testing.T) {
	ctx := context.Background()
	be := newLocalBackendIOTest(t)

	csvPath := filepath.Join(t.TempDir(), "import.csv")
	csvContent := "Starter,Ich bin hier,I am here,basic,sentence|present\n"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	n, err := be.ImportCSV(ctx, csvPath)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported=%d want=1", n)
	}
	results, err := be.SearchCards(ctx, "Ich bin hier", 10)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(results) != 1 || results[0].Back != "I am here" {
		t.Fatalf("unexpected imported result: %+v", results)
	}
}
