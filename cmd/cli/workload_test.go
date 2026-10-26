package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const deploymentListDoc = `{
  "kind": "DeploymentList",
  "items": [
    {
      "metadata": {"name": "web", "namespace": "prod"},
      "spec": {"selector": {"matchLabels": {"app": "web"}}}
    },
    {
      "metadata": {"name": "empty", "namespace": "prod"},
      "spec": {"selector": {"matchLabels": {"app": "empty"}}}
    }
  ]
}`

const daemonSetListDoc = `{
  "kind": "DaemonSetList",
  "items": [
    {
      "metadata": {"name": "agent", "namespace": "prod"},
      "spec": {
        "selector": {
          "matchExpressions": [
            {"key": "app", "operator": "In", "values": ["agent"]}
          ]
        }
      }
    }
  ]
}`

const statefulSetListDoc = `{
  "kind": "StatefulSetList",
  "items": [
    {
      "metadata": {"name": "db", "namespace": "prod"},
      "spec": {"selector": {"matchLabels": {"app": "db"}}}
    }
  ]
}`

const podListDoc = `{
  "kind": "PodList",
  "items": [
    {"metadata": {"name": "web-1", "namespace": "prod", "labels": {"app": "web"}}},
    {"metadata": {"name": "web-2", "namespace": "prod", "labels": {"app": "web"}}},
    {"metadata": {"name": "agent-1", "namespace": "prod", "labels": {"app": "agent"}}},
    {"metadata": {"name": "db-0", "namespace": "prod", "labels": {"app": "db"}}},
    {"metadata": {"name": "db-1", "namespace": "prod", "labels": {"app": "db"}}},
    {"metadata": {"name": "web-staging", "namespace": "staging", "labels": {"app": "web"}}},
    {"metadata": {"name": "web-unrelated", "namespace": "prod", "labels": {"app": "other"}}}
  ]
}`

const genericListDoc = `{
  "kind": "List",
  "items": [
    {
      "kind": "Deployment",
      "metadata": {"name": "web", "namespace": "prod"},
      "spec": {"selector": {"matchLabels": {"app": "web"}}}
    },
    {
      "kind": "Pod",
      "metadata": {"name": "web-1", "namespace": "prod", "labels": {"app": "web"}}
    },
    {
      "kind": "Pod",
      "metadata": {"name": "web-2", "namespace": "prod", "labels": {"app": "web"}}
    }
  ]
}`

func joinDocs(docs ...string) string {
	return strings.Join(docs, "\n") + "\n"
}

func saveLogGlobals(t *testing.T) {
	t.Helper()
	oldDumpFile, oldDumpDir, oldNamespace, oldContainer := dumpFile, dumpDir, resNamespace, resContainer
	t.Cleanup(func() {
		dumpFile, dumpDir, resNamespace, resContainer = oldDumpFile, oldDumpDir, oldNamespace, oldContainer
	})
}

func TestResolveWorkloadPodsFromDumpFile(t *testing.T) {
	saveLogGlobals(t)
	dumpFile = writeDump(t, joinDocs(deploymentListDoc, daemonSetListDoc, statefulSetListDoc, podListDoc))
	dumpDir = ""
	resNamespace = "prod"

	tests := []struct {
		kind string
		name string
		want []string
	}{
		{"Deployment", "web", []string{"web-1", "web-2"}},
		{"DaemonSet", "agent", []string{"agent-1"}},
		{"StatefulSet", "db", []string{"db-0", "db-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got, err := resolveWorkloadPods(tt.kind, resNamespace, tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveWorkloadPods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveWorkloadPodsFromGenericList(t *testing.T) {
	saveLogGlobals(t)
	dumpFile = writeDump(t, genericListDoc)
	dumpDir = ""

	got, err := resolveWorkloadPods("Deployment", "prod", "web")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"web-1", "web-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveWorkloadPods() = %v, want %v", got, want)
	}
}

func TestResolveWorkloadPodsFromDumpDir(t *testing.T) {
	saveLogGlobals(t)
	dumpDir = t.TempDir()
	dumpFile = ""
	namespaceDir := filepath.Join(dumpDir, "prod")
	if err := os.MkdirAll(namespaceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(namespaceDir, "deployments.json"), []byte(deploymentListDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(namespaceDir, "pods.json"), []byte(podListDoc), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveWorkloadPods("Deployment", "prod", "web")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"web-1", "web-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveWorkloadPods() = %v, want %v", got, want)
	}
}

func TestResolveWorkloadPodsFromSosDir(t *testing.T) {
	saveLogGlobals(t)
	root := t.TempDir()
	dumpDir = filepath.Join(root, "sos_commands", "kubernetes", "cluster-info")
	dumpFile = ""
	namespaceDir := filepath.Join(dumpDir, "prod")
	if err := os.MkdirAll(namespaceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(namespaceDir, "kubectl_get_-o_json_--namespace_prod_deployments"), []byte(genericListDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	// The generic list doubles as the pod list here; it contains both kinds.
	if err := os.WriteFile(filepath.Join(namespaceDir, "kubectl_get_-o_json_--namespace_prod_pods"), []byte(genericListDoc), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveWorkloadPods("Deployment", "prod", "web")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"web-1", "web-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveWorkloadPods() = %v, want %v", got, want)
	}
}

func TestResolveWorkloadPodsErrors(t *testing.T) {
	saveLogGlobals(t)
	dumpFile = writeDump(t, joinDocs(deploymentListDoc, podListDoc))
	dumpDir = ""
	resNamespace = "prod"

	if _, err := resolveWorkloadPods("Deployment", "prod", "missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
	if _, err := resolveWorkloadPods("Deployment", "prod", "empty"); err == nil || !strings.Contains(err.Error(), "no pods found") {
		t.Fatalf("expected no-pods error, got %v", err)
	}
}

func TestResolveLogTarget(t *testing.T) {
	saveLogGlobals(t)
	dumpFile = writeDump(t, joinDocs(deploymentListDoc, daemonSetListDoc, statefulSetListDoc, podListDoc))
	dumpDir = ""
	resNamespace = "prod"

	tests := []struct {
		target       string
		wantPods     []string
		wantWorkload bool
	}{
		{"web-1", []string{"web-1"}, false},
		{"pod/web-1", []string{"web-1"}, false},
		{"deploy/web", []string{"web-1", "web-2"}, true},
		{"daemonset/agent", []string{"agent-1"}, true},
		{"sts/db", []string{"db-0", "db-1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			pods, workload, err := resolveLogTarget(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			if workload != tt.wantWorkload {
				t.Errorf("workload = %t, want %t", workload, tt.wantWorkload)
			}
			if !reflect.DeepEqual(pods, tt.wantPods) {
				t.Errorf("pods = %v, want %v", pods, tt.wantPods)
			}
		})
	}

	for _, target := range []string{"svc/web", "deploy/", "/web"} {
		if _, _, err := resolveLogTarget(target); err == nil {
			t.Errorf("resolveLogTarget(%q) did not return an error", target)
		}
	}
}
