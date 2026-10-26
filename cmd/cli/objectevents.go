package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/describe"
)

// printObjectEvents prints the Events section for the object being described,
// using events found in the dump rather than a live cluster.
func printObjectEvents(out io.Writer) error {
	events := dedupeEvents(collectObjectEvents(resKind, resNamespace, resName))
	s, err := tabbedString(func(w io.Writer) error {
		describe.DescribeEvents(&corev1.EventList{Items: events}, describe.NewPrefixWriter(w))
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(out, s)
	return nil
}

func collectObjectEvents(kind, namespace, name string) []corev1.Event {
	var events []corev1.Event
	process := func(buffer []byte) bool {
		docType := docKind(buffer)
		if docType != "List" && !docMatchesKind(docType, "Event") {
			return true
		}
		var list corev1.EventList
		if err := json.Unmarshal(buffer, &list); err != nil {
			return true
		}
		for _, event := range list.Items {
			if eventMatchesObject(event, kind, namespace, name) {
				events = append(events, event)
			}
		}
		return true
	}

	if len(dumpDir) == 0 {
		readFile(dumpFile, process)
		return events
	}

	if isSosreportDir(dumpDir) {
		if contains(UnnamespacedTypes, kind) {
			entries, err := os.ReadDir(dumpDir)
			if err != nil {
				return events
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				path, err := findSosResourceFile(dumpDir, entry.Name(), DumpFileNames["Event"])
				if err == nil {
					readFile(path, process)
				}
			}
			return events
		}
		path, err := findSosResourceFile(dumpDir, namespace, DumpFileNames["Event"])
		if err == nil {
			readFile(path, process)
		}
		return events
	}

	if contains(UnnamespacedTypes, kind) {
		entries, err := os.ReadDir(dumpDir)
		if err != nil {
			return events
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			readFile(filepath.Join(dumpDir, entry.Name(), DumpFileNames["Event"]+"."+dumpFormat), process)
		}
		return events
	}
	readFile(filepath.Join(dumpDir, namespace, DumpFileNames["Event"]+"."+dumpFormat), process)
	return events
}

func eventMatchesObject(event corev1.Event, kind, namespace, name string) bool {
	obj := event.InvolvedObject
	if !strings.EqualFold(obj.Kind, kind) || obj.Name != name {
		return false
	}
	if contains(UnnamespacedTypes, kind) {
		return true
	}
	if obj.Namespace == namespace {
		return true
	}
	return obj.Namespace == "" && event.Namespace == namespace
}

func dedupeEvents(events []corev1.Event) []corev1.Event {
	if len(events) < 2 {
		return events
	}
	seen := make(map[string]struct{}, len(events))
	result := events[:0]
	for _, event := range events {
		uid := string(event.UID)
		if uid != "" {
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
		}
		result = append(result, event)
	}
	return result
}
