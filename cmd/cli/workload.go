package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type workloadPod struct {
	name   string
	labels map[string]string
}

type dumpListItem struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Selector *metav1.LabelSelector `json:"selector"`
	} `json:"spec"`
}

type dumpList struct {
	Items []dumpListItem `json:"items"`
}

// workloadPodResolver collects a workload's label selector and the pods in its
// namespace from one or more cluster-info dump files.
type workloadPodResolver struct {
	kind      string
	namespace string
	name      string

	selector labels.Selector
	found    bool
	err      error
	pods     []workloadPod
}

func (r *workloadPodResolver) process(buffer []byte) bool {
	kind := docKind(buffer)
	if kind == "" {
		return true
	}
	// Skip unrelated typed lists without decoding their items.
	if kind != "List" && !docMatchesKind(kind, r.kind) && !docMatchesKind(kind, "Pod") {
		return true
	}

	var doc dumpList
	if err := json.Unmarshal(buffer, &doc); err != nil {
		return true
	}

	switch {
	case kind == "List":
		// sosreport and some dump files use a generic List whose items carry
		// their own kind, so the document kind alone is not enough.
		for _, item := range doc.Items {
			switch item.Kind {
			case r.kind:
				r.addWorkload(item)
			case "Pod":
				r.addPod(item)
			}
		}
	case docMatchesKind(kind, r.kind):
		for _, item := range doc.Items {
			r.addWorkload(item)
		}
	case docMatchesKind(kind, "Pod"):
		for _, item := range doc.Items {
			r.addPod(item)
		}
	}
	return true
}

func (r *workloadPodResolver) addWorkload(item dumpListItem) {
	if item.Metadata.Namespace != r.namespace || item.Metadata.Name != r.name {
		return
	}
	if item.Spec.Selector == nil {
		r.err = fmt.Errorf("%s %q in namespace %q has no selector", r.kind, r.name, r.namespace)
		return
	}
	selector, err := metav1.LabelSelectorAsSelector(item.Spec.Selector)
	if err != nil {
		r.err = fmt.Errorf("invalid selector for %s %q in namespace %q: %w", r.kind, r.name, r.namespace, err)
		return
	}
	r.selector = selector
	r.found = true
}

func (r *workloadPodResolver) addPod(item dumpListItem) {
	if item.Metadata.Namespace != r.namespace {
		return
	}
	r.pods = append(r.pods, workloadPod{name: item.Metadata.Name, labels: item.Metadata.Labels})
}

func (r *workloadPodResolver) matchingPods() []string {
	seen := make(map[string]struct{})
	var pods []string
	for _, pod := range r.pods {
		if _, ok := seen[pod.name]; ok {
			continue
		}
		if r.selector.Matches(labels.Set(pod.labels)) {
			seen[pod.name] = struct{}{}
			pods = append(pods, pod.name)
		}
	}
	sort.Strings(pods)
	return pods
}

// resolveWorkloadPods returns the names of the pods selected by a Deployment,
// DaemonSet or StatefulSet in the dump file or dump directory.
func resolveWorkloadPods(kind, namespace, name string) ([]string, error) {
	resolver := &workloadPodResolver{kind: kind, namespace: namespace, name: name}

	if len(dumpDir) > 0 {
		if isSosreportDir(dumpDir) {
			workloadFile, err := findSosResourceFile(dumpDir, namespace, DumpFileNames[kind])
			if err != nil {
				return nil, err
			}
			podsFile, err := findSosResourceFile(dumpDir, namespace, DumpFileNames["Pod"])
			if err != nil {
				return nil, err
			}
			readFile(workloadFile, resolver.process)
			readFile(podsFile, resolver.process)
		} else {
			readFile(filepath.Join(dumpDir, namespace, DumpFileNames[kind]+"."+dumpFormat), resolver.process)
			readFile(filepath.Join(dumpDir, namespace, DumpFileNames["Pod"]+"."+dumpFormat), resolver.process)
		}
	} else {
		readFile(dumpFile, resolver.process)
	}

	if resolver.err != nil {
		return nil, resolver.err
	}
	if !resolver.found {
		return nil, fmt.Errorf("%s %q not found in namespace %q", kind, name, namespace)
	}
	pods := resolver.matchingPods()
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for %s %q in namespace %q", kind, name, namespace)
	}
	return pods, nil
}

func findSosResourceFile(dumpDir, namespace, resource string) (string, error) {
	namespaceDir := filepath.Join(dumpDir, namespace)
	entries, err := os.ReadDir(namespaceDir)
	if err != nil {
		return "", fmt.Errorf("error reading namespace %q: %w", namespace, err)
	}
	suffix := "_get_-o_json_--namespace_" + namespace + "_" + resource
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			return filepath.Join(namespaceDir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("%s data for namespace %q not found under %s", resource, namespace, dumpDir)
}
