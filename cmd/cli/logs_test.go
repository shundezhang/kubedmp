package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const logSample = `==== START logs for container c of pod ns/foo ====
foo line
==== END logs for container c of pod ns/foo ====
==== START logs for container c of pod ns/foobar ====
foobar line
==== END logs for container c of pod ns/foobar ====
==== START logs for container c of pod ns/bar ====
bar line
==== END logs for container c of pod ns/bar ====
`

const fooLogSection = `==== START logs for container c of pod ns/foo ====
foo line
==== END logs for container c of pod ns/foo ====
`

func TestPrintLogsMatchesExactPod(t *testing.T) {
	saveLogGlobals(t)
	resContainer = ""

	var out bytes.Buffer
	if err := printLogs(strings.NewReader(logSample), &out, []string{"ns/foo"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != fooLogSection {
		t.Fatalf("printLogs() = %q, want %q", out.String(), fooLogSection)
	}
}

func TestPrintLogsMultiplePods(t *testing.T) {
	saveLogGlobals(t)
	resContainer = ""

	var out bytes.Buffer
	if err := printLogs(strings.NewReader(logSample), &out, []string{"ns/foo", "ns/bar"}); err != nil {
		t.Fatal(err)
	}
	want := fooLogSection + `==== START logs for container c of pod ns/bar ====
bar line
==== END logs for container c of pod ns/bar ====
`
	if out.String() != want {
		t.Fatalf("printLogs() = %q, want %q", out.String(), want)
	}
	if strings.Contains(out.String(), "foobar line") {
		t.Fatal("printLogs() matched pod ns/foo when only ns/foo and ns/bar were requested")
	}
}

func TestPrintLogsContainerMarker(t *testing.T) {
	saveLogGlobals(t)
	resContainer = "c"

	var out bytes.Buffer
	if err := printLogs(strings.NewReader(logSample), &out, []string{"container c of pod ns/foo"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != fooLogSection {
		t.Fatalf("printLogs() = %q, want %q", out.String(), fooLogSection)
	}
}

func TestPrintLogsAll(t *testing.T) {
	saveLogGlobals(t)
	resContainer = ""

	var out bytes.Buffer
	if err := printLogs(strings.NewReader(logSample), &out, []string{""}); err != nil {
		t.Fatal(err)
	}
	if out.String() != logSample {
		t.Fatalf("printLogs() did not print the whole file")
	}
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

func TestPrintLogsStopsAfterLastContainerMarker(t *testing.T) {
	saveLogGlobals(t)
	resContainer = "c"

	input := logSample + strings.Repeat("padding line\n", 100000)
	reader := &countingReader{r: strings.NewReader(input)}
	if err := printLogs(reader, io.Discard, []string{"container c of pod ns/foo"}); err != nil {
		t.Fatal(err)
	}
	if reader.n >= len(input) {
		t.Fatalf("printLogs() read the whole file (%d bytes) instead of stopping early", reader.n)
	}
}

func TestPrintLogsNoMarkers(t *testing.T) {
	saveLogGlobals(t)
	resContainer = ""

	var out bytes.Buffer
	if err := printLogs(strings.NewReader(logSample), &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("printLogs() with no markers wrote %d bytes", out.Len())
	}
}

func TestLogMarker(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"

	resContainer = ""
	if got := logMarker("pod"); got != "ns/pod" {
		t.Fatalf("logMarker() = %q, want ns/pod", got)
	}

	resContainer = "c"
	if got := logMarker("pod"); got != "container c of pod ns/pod" {
		t.Fatalf("logMarker() = %q, want container c of pod ns/pod", got)
	}
}

func TestParseLogHeader(t *testing.T) {
	pod, container, ok := parseLogHeader("==== START logs for container c of pod ns/pod ====", logStartPrefix)
	if !ok {
		t.Fatal("parseLogHeader() did not parse a valid header")
	}
	if pod != "ns/pod" || container != "container c of pod ns/pod" {
		t.Fatalf("parseLogHeader() = %q, %q", pod, container)
	}

	if _, _, ok := parseLogHeader("not a header", logStartPrefix); ok {
		t.Fatal("parseLogHeader() parsed an invalid header")
	}
}

func setupSosPodLogs(t *testing.T, podName string, containers map[string]string) {
	t.Helper()
	root := t.TempDir()
	dumpDir = filepath.Join(root, "sos_commands", "kubernetes", "cluster-info")
	podLogDir := filepath.Join(dumpDir, resNamespace, "podlogs", podName)
	if err := os.MkdirAll(podLogDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for container, content := range containers {
		name := "kubectl_--namespace_" + resNamespace + "_logs_" + podName + "_-c_" + container
		if err := os.WriteFile(filepath.Join(podLogDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLogFilesForPodInDirMultipleContainers(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"
	resContainer = ""
	setupSosPodLogs(t, "pod", map[string]string{
		"alpha": "alpha log\n",
		"beta":  "beta log\n",
	})

	files, err := logFilesForPodInDir("pod")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("logFilesForPodInDir() returned %d files, want 2", len(files))
	}
	if files[0].container != "alpha" || files[1].container != "beta" {
		t.Fatalf("containers = %q, %q; want alpha, beta", files[0].container, files[1].container)
	}

	resContainer = "beta"
	files, err = logFilesForPodInDir("pod")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].container != "beta" {
		t.Fatalf("logFilesForPodInDir() with -c beta returned %+v", files)
	}

	resContainer = "missing"
	if _, err := logFilesForPodInDir("pod"); err == nil {
		t.Fatal("logFilesForPodInDir() did not return an error for a missing container")
	}
}

func expectedLogSection(container, content string) string {
	return "==== START logs for container " + container + " of pod ns/pod ====\n" +
		content +
		"==== END logs for container " + container + " of pod ns/pod ====\n"
}

func TestPrintDirLogsShowsAllContainers(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"
	resContainer = ""
	setupSosPodLogs(t, "pod", map[string]string{
		"alpha": "alpha log\n",
		"beta":  "beta log\n",
	})

	var out, errOut bytes.Buffer
	if err := printDirLogs(&out, &errOut, "pod", []string{"pod"}, false); err != nil {
		t.Fatal(err)
	}
	want := expectedLogSection("alpha", "alpha log\n") + expectedLogSection("beta", "beta log\n")
	if out.String() != want {
		t.Fatalf("printDirLogs() = %q, want %q", out.String(), want)
	}
	if errOut.Len() != 0 {
		t.Fatalf("printDirLogs() wrote unexpected stderr: %q", errOut.String())
	}
}

func TestPrintDirLogsSelectedContainer(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"
	resContainer = "beta"
	setupSosPodLogs(t, "pod", map[string]string{
		"alpha": "alpha log\n",
		"beta":  "beta log\n",
	})

	var out, errOut bytes.Buffer
	if err := printDirLogs(&out, &errOut, "pod", []string{"pod"}, false); err != nil {
		t.Fatal(err)
	}
	want := expectedLogSection("beta", "beta log\n")
	if out.String() != want {
		t.Fatalf("printDirLogs() = %q, want %q", out.String(), want)
	}
}

func TestPrintDirLogsEmptyContainerLog(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "tcpdump"
	resContainer = ""
	podName := "tcpdump-daemonset-6kd2b"
	setupSosPodLogs(t, podName, map[string]string{
		"tcpdump": "",
	})

	var out, errOut bytes.Buffer
	if err := printDirLogs(&out, &errOut, podName, []string{podName}, false); err != nil {
		t.Fatal(err)
	}
	want := "==== START logs for container tcpdump of pod tcpdump/" + podName + " ====\n" +
		"==== END logs for container tcpdump of pod tcpdump/" + podName + " ====\n"
	if out.String() != want {
		t.Fatalf("printDirLogs() = %q, want %q", out.String(), want)
	}
}

func TestPrintDirLogsWorkloadMultipleContainers(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"
	resContainer = ""
	setupSosPodLogs(t, "pod", map[string]string{
		"alpha": "alpha log\n",
		"beta":  "beta log\n",
	})

	var out, errOut bytes.Buffer
	if err := printDirLogs(&out, &errOut, "deploy/web", []string{"pod"}, true); err != nil {
		t.Fatal(err)
	}
	want := expectedLogSection("alpha", "alpha log\n") + expectedLogSection("beta", "beta log\n")
	if out.String() != want {
		t.Fatalf("printDirLogs() = %q, want %q", out.String(), want)
	}
}

func TestPrintDirLogsWorkloadHeader(t *testing.T) {
	saveLogGlobals(t)
	resNamespace = "ns"
	resContainer = ""
	setupSosPodLogs(t, "pod", map[string]string{
		"alpha": "alpha log\n",
	})

	var out, errOut bytes.Buffer
	if err := printDirLogs(&out, &errOut, "deploy/web", []string{"pod"}, true); err != nil {
		t.Fatal(err)
	}
	want := expectedLogSection("alpha", "alpha log\n")
	if out.String() != want {
		t.Fatalf("printDirLogs() = %q, want %q", out.String(), want)
	}
}

func TestContainerFromSosLogFileName(t *testing.T) {
	if got := containerFromSosLogFileName("kubectl_--namespace_ns_logs_pod_-c_my-container"); got != "my-container" {
		t.Fatalf("containerFromSosLogFileName() = %q, want my-container", got)
	}
	if got := containerFromSosLogFileName("no-container-suffix"); got != "" {
		t.Fatalf("containerFromSosLogFileName() = %q, want empty", got)
	}
}
