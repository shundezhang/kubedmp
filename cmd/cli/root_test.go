package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeDump(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cluster-info.dump")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func documentKinds(t *testing.T, path string, stopAfter int) []string {
	t.Helper()
	var kinds []string
	readFile(path, func(data []byte) bool {
		var doc struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("invalid document passed to callback: %v\n%s", err, data)
		}
		kinds = append(kinds, doc.Kind)
		return stopAfter == 0 || len(kinds) < stopAfter
	})
	return kinds
}

func TestReadFile(t *testing.T) {
	content := "leading non-JSON output\n" +
		"{\n  \"kind\": \"NodeList\",\n  \"items\": []\n}\n" +
		"==== START logs for container c of pod ns/p ====\n" +
		"some log output\n" +
		"{\n  \"kind\": \"PodList\",\n  \"items\": []\n}" // no trailing newline

	path := writeDump(t, content)
	want := []string{"NodeList", "PodList"}
	if got := documentKinds(t, path, 0); !reflect.DeepEqual(got, want) {
		t.Fatalf("readFile() = %v, want %v", got, want)
	}
}

func TestReadFileStopsWhenCallbackReturnsFalse(t *testing.T) {
	content := "{\n  \"kind\": \"NodeList\",\n  \"items\": []\n}\n" +
		"{\n  \"kind\": \"PodList\",\n  \"items\": []\n}\n"

	path := writeDump(t, content)
	if got := documentKinds(t, path, 1); !reflect.DeepEqual(got, []string{"NodeList"}) {
		t.Fatalf("readFile() = %v, want only the first document", got)
	}
}

func TestReadFileCRLF(t *testing.T) {
	content := "{\r\n  \"kind\": \"PodList\",\r\n  \"items\": []\r\n}\r\n"
	path := writeDump(t, content)
	if got := documentKinds(t, path, 0); !reflect.DeepEqual(got, []string{"PodList"}) {
		t.Fatalf("readFile() = %v, want [PodList]", got)
	}
}

func TestReadFileMissingFile(t *testing.T) {
	called := false
	readFile(filepath.Join(t.TempDir(), "does-not-exist"), func([]byte) bool {
		called = true
		return true
	})
	if called {
		t.Fatal("callback was called for a missing file")
	}
}

func TestReadFileIgnoresUnterminatedDocument(t *testing.T) {
	content := "{\n  \"kind\": \"PodList\",\n  \"items\": []\n"
	path := writeDump(t, content)
	if got := documentKinds(t, path, 0); len(got) != 0 {
		t.Fatalf("readFile() = %v, want no documents", got)
	}
}

func TestParseResourceArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantType  string
		wantName  string
		wantError bool
	}{
		{"type only", []string{"pod"}, "pod", "", false},
		{"type and name", []string{"pod", "nginx"}, "pod", "nginx", false},
		{"type slash name", []string{"pod/nginx"}, "pod", "nginx", false},
		{"alias slash name", []string{"deploy/web"}, "deploy", "web", false},
		{"no args", nil, "", "", true},
		{"empty name", []string{"pod/"}, "", "", true},
		{"empty type", []string{"/nginx"}, "", "", true},
		{"extra slash", []string{"pod/nginx/extra"}, "", "", true},
		{"name plus extra arg", []string{"pod/nginx", "extra"}, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotName, err := parseResourceArgs(tt.args)
			if tt.wantError {
				if err == nil {
					t.Fatalf("parseResourceArgs(%v) did not return an error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotType != tt.wantType || gotName != tt.wantName {
				t.Fatalf("parseResourceArgs(%v) = %q, %q; want %q, %q", tt.args, gotType, gotName, tt.wantType, tt.wantName)
			}
		})
	}
}

func TestDocKind(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"kind first", `{"kind":"PodList","apiVersion":"v1"}`, "PodList"},
		{"kind later", `{"apiVersion":"v1","metadata":{"name":"x"},"kind":"NodeList"}`, "NodeList"},
		{"generic list", `{"kind":"List","items":[]}`, "List"},
		{"invalid", `{"kind":`, ""},
		{"not an object", `[]`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := docKind([]byte(tt.data)); got != tt.want {
				t.Fatalf("docKind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocMatchesKind(t *testing.T) {
	tests := []struct {
		kind     string
		resource string
		want     bool
	}{
		{"PodList", "Pod", true},
		{"PodList", "Node", false},
		{"List", "Pod", true},
		{"Pod", "Pod", false},
		{"PodList", "", false},
	}
	for _, tt := range tests {
		if got := docMatchesKind(tt.kind, tt.resource); got != tt.want {
			t.Errorf("docMatchesKind(%q, %q) = %t, want %t", tt.kind, tt.resource, got, tt.want)
		}
	}
}

func TestProcessDocFiltersAndDistinguishesDocuments(t *testing.T) {
	oldKind, oldNamespace, oldName, oldAll := resKind, resNamespace, resName, allNamespaces
	t.Cleanup(func() {
		resKind, resNamespace, resName, allNamespaces = oldKind, oldNamespace, oldName, oldAll
	})

	resKind = "Pod"
	resNamespace = "team-a"
	resName = ""
	allNamespaces = false

	pods := []byte(`{
		"kind": "PodList",
		"items": [
			{"kind": "Pod", "metadata": {"name": "a", "namespace": "team-a"}},
			{"kind": "Pod", "metadata": {"name": "b", "namespace": "team-b"}}
		]
	}`)
	displayItems = nil
	if keepGoing := processDoc(pods); !keepGoing {
		t.Fatal("processDoc() asked to stop reading")
	}
	if len(displayItems) != 1 {
		t.Fatalf("processDoc() collected %d items, want 1", len(displayItems))
	}
	meta := displayItems[0].(map[string]interface{})["metadata"].(map[string]interface{})
	if meta["name"] != "a" {
		t.Fatalf("processDoc() collected pod %v, want a", meta["name"])
	}

	// A document of another kind must be skipped.
	displayItems = nil
	processDoc([]byte(`{"kind": "NodeList", "items": [{"kind": "Node"}]}`))
	if len(displayItems) != 0 {
		t.Fatalf("processDoc() collected %d items from an unrelated document", len(displayItems))
	}
}
