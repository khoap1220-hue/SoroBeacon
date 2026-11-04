package dashboards_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDashboardJsonValidity(t *testing.T) {
	files, err := filepath.Glob("*.json")
	if err != nil {
		t.Fatalf("failed finding json dashboards: %v", err)
	}

	if len(files) == 0 {
		// Look in current or parent directory
		files, _ = filepath.Glob("deploy/dashboards/*.json")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("failed reading %s: %v", file, err)
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("file %s is not valid JSON: %v", file, err)
			continue
		}

		if title, ok := parsed["title"].(string); !ok || title == "" {
			t.Errorf("file %s missing title property", file)
		}
	}
}
