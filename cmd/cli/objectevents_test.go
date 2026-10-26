package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const eventListDoc = `{
  "kind": "EventList",
  "items": [
    {
      "metadata": {"name": "e1", "namespace": "ns", "uid": "uid-1"},
      "involvedObject": {"kind": "Pod", "namespace": "ns", "name": "pod-a"},
      "type": "Normal",
      "reason": "Scheduled",
      "message": "Successfully assigned ns/pod-a",
      "source": {"component": "default-scheduler"},
      "lastTimestamp": "2026-10-20T10:00:00Z",
      "count": 1
    },
    {
      "metadata": {"name": "e2", "namespace": "ns", "uid": "uid-2"},
      "involvedObject": {"kind": "Pod", "namespace": "ns", "name": "pod-b"},
      "type": "Warning",
      "reason": "Failed",
      "message": "failed to start pod-b",
      "source": {"component": "kubelet"},
      "lastTimestamp": "2026-10-20T10:01:00Z",
      "count": 1
    },
    {
      "metadata": {"name": "e3", "namespace": "other", "uid": "uid-3"},
      "involvedObject": {"kind": "Pod", "namespace": "other", "name": "pod-a"},
      "type": "Normal",
      "reason": "Scheduled",
      "message": "scheduled in another namespace",
      "source": {"component": "default-scheduler"},
      "lastTimestamp": "2026-10-20T10:02:00Z",
      "count": 1
    },
    {
      "metadata": {"name": "e4", "namespace": "default", "uid": "uid-4"},
      "involvedObject": {"kind": "Node", "name": "node-a"},
      "type": "Normal",
      "reason": "Ready",
      "message": "Node node-a is ready",
      "source": {"component": "kubelet"},
      "lastTimestamp": "2026-10-20T10:03:00Z",
      "count": 1
    },
    {
      "metadata": {"name": "e1", "namespace": "ns", "uid": "uid-1"},
      "involvedObject": {"kind": "Pod", "namespace": "ns", "name": "pod-a"},
      "type": "Normal",
      "reason": "Scheduled",
      "message": "Successfully assigned ns/pod-a",
      "source": {"component": "default-scheduler"},
      "lastTimestamp": "2026-10-20T10:00:00Z",
      "count": 1
    }
  ]
}`

const nodeEventListDoc = `{
  "kind": "EventList",
  "items": [
    {
      "metadata": {"name": "n1", "namespace": "default", "uid": "uid-node"},
      "involvedObject": {"kind": "Node", "name": "node-a"},
      "type": "Normal",
      "reason": "Ready",
      "message": "Node node-a is ready",
      "source": {"component": "kubelet"},
      "lastTimestamp": "2026-10-20T11:00:00Z",
      "count": 1
    }
  ]
}`

func saveDescribeGlobals(t *testing.T) {
	t.Helper()
	saveLogGlobals(t)
	oldKind, oldName := resKind, resName
	t.Cleanup(func() {
		resKind, resName = oldKind, oldName
	})
}

func TestCollectObjectEventsFromDumpFile(t *testing.T) {
	saveDescribeGlobals(t)
	dumpFile = writeDump(t, joinDocs(eventListDoc, nodeEventListDoc))
	dumpDir = ""

	podEvents := collectObjectEvents("Pod", "ns", "pod-a")
	if len(podEvents) != 2 {
		t.Fatalf("collectObjectEvents(Pod) returned %d events, want 2 (including duplicate)", len(podEvents))
	}
	for _, event := range podEvents {
		if event.InvolvedObject.Name != "pod-a" || event.InvolvedObject.Namespace != "ns" {
			t.Fatalf("unexpected event for %s/%s", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
		}
	}
	if got := dedupeEvents(podEvents); len(got) != 1 {
		t.Fatalf("dedupeEvents() returned %d events, want 1", len(got))
	}

	nodeEvents := collectObjectEvents("Node", "", "node-a")
	if len(nodeEvents) != 2 {
		t.Fatalf("collectObjectEvents(Node) returned %d events, want 2", len(nodeEvents))
	}
	for _, event := range nodeEvents {
		if event.Reason != "Ready" {
			t.Fatalf("unexpected node event reason %q", event.Reason)
		}
	}
}

func TestPrintObjectEvents(t *testing.T) {
	saveDescribeGlobals(t)
	dumpFile = writeDump(t, joinDocs(eventListDoc, nodeEventListDoc))
	dumpDir = ""
	resKind = "Pod"
	resNamespace = "ns"
	resName = "pod-a"

	var out bytes.Buffer
	if err := printObjectEvents(&out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"Events:", "Scheduled", "default-scheduler", "Successfully assigned ns/pod-a"} {
		if !strings.Contains(s, want) {
			t.Fatalf("printObjectEvents() output missing %q:\n%s", want, s)
		}
	}
	if strings.Count(s, "Scheduled") != 1 {
		t.Fatalf("printObjectEvents() printed duplicate events:\n%s", s)
	}

	resName = "missing"
	out.Reset()
	if err := printObjectEvents(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Events:") || !strings.Contains(out.String(), "<none>") {
		t.Fatalf("printObjectEvents() output = %q, want Events: <none>", out.String())
	}
}

func TestCollectObjectEventsFromDumpDir(t *testing.T) {
	saveDescribeGlobals(t)
	dumpDir = t.TempDir()
	dumpFile = ""

	for namespace, content := range map[string]string{
		"ns":      eventListDoc,
		"default": nodeEventListDoc,
	} {
		dir := filepath.Join(dumpDir, namespace)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "events.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if got := collectObjectEvents("Pod", "ns", "pod-a"); len(got) != 2 {
		t.Fatalf("collectObjectEvents(Pod) returned %d events, want 2", len(got))
	}
	if got := collectObjectEvents("Node", "", "node-a"); len(got) != 2 {
		t.Fatalf("collectObjectEvents(Node) returned %d events, want 2", len(got))
	}
}

func corev1Event(kind, namespace, name, eventNamespace string) corev1.Event {
	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: eventNamespace},
		InvolvedObject: corev1.ObjectReference{
			Kind:      kind,
			Namespace: namespace,
			Name:      name,
		},
	}
}

func TestCollectObjectEventsFromSosDir(t *testing.T) {
	saveDescribeGlobals(t)
	root := t.TempDir()
	dumpDir = filepath.Join(root, "sos_commands", "kubernetes", "cluster-info")
	dumpFile = ""

	for namespace, content := range map[string]string{
		"ns":      eventListDoc,
		"default": nodeEventListDoc,
	} {
		dir := filepath.Join(dumpDir, namespace)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		name := "kubectl_--namespace_" + namespace + "_get_-o_json_--namespace_" + namespace + "_events"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if got := collectObjectEvents("Pod", "ns", "pod-a"); len(got) != 2 {
		t.Fatalf("collectObjectEvents(Pod) returned %d events, want 2", len(got))
	}
	if got := collectObjectEvents("Node", "", "node-a"); len(got) != 2 {
		t.Fatalf("collectObjectEvents(Node) returned %d events, want 2", len(got))
	}
}

func TestEventMatchesObject(t *testing.T) {
	event := corev1Event("Pod", "ns", "pod-a", "ns")
	if !eventMatchesObject(event, "Pod", "ns", "pod-a") {
		t.Fatal("eventMatchesObject() did not match a namespaced pod event")
	}
	if eventMatchesObject(event, "Pod", "other", "pod-a") {
		t.Fatal("eventMatchesObject() matched an event in another namespace")
	}
	if eventMatchesObject(event, "Pod", "ns", "pod-b") {
		t.Fatal("eventMatchesObject() matched an event for another pod")
	}
	if eventMatchesObject(event, "Deployment", "ns", "pod-a") {
		t.Fatal("eventMatchesObject() matched an event for another kind")
	}

	event = corev1Event("Node", "", "node-a", "longhorn-system")
	if !eventMatchesObject(event, "Node", "", "node-a") {
		t.Fatal("eventMatchesObject() did not match a cluster-scoped event")
	}
	if !eventMatchesObject(event, "node", "", "node-a") {
		t.Fatal("eventMatchesObject() did not match the kind case-insensitively")
	}
}
