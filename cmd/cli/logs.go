package cli

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	cont = "container"

	logStartPrefix = "==== START logs for container "
	logEndPrefix   = "==== END logs for container "
)

var logsCmd = &cobra.Command{
	Use:                   "logs POD_NAME | TYPE/NAME [-n NAMESPACE] [-c CONTAINER_NAME]",
	DisableFlagsInUseLine: true,
	Short:                 "Print the logs for a pod or workload",
	Long: `Print the logs for a container in a pod or specified resource.
If the pod has more than one container, and a container name is not specified, logs of all containers will be printed out.

A deployment, daemonset or statefulset can be specified as TYPE/NAME to print the logs of
all pods selected by that resource, for example:
  kubedmp logs deployment/nginx
  kubedmp logs ds/fluentd
  kubedmp logs sts/database`,
	Example: `  # Return logs from pod nginx with all containers
  kubedmp logs nginx

  # Return logs of ruby container logs from pod web-1
  kubedmp logs web-1 -c ruby

  # Return logs from all pods in deployment web in namespace prod
  kubedmp logs deploy/web -n prod

  # Return logs from all pods in statefulset db in namespace prod
  kubedmp logs sts/db -n prod`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalf("Please provide a pod name, or a resource in the form TYPE/NAME.\n")
			return
		}

		podNames, fromWorkload, err := resolveLogTarget(args[0])
		if err != nil {
			log.Fatalf("%v\n", err)
			return
		}

		if len(dumpDir) > 0 {
			if err := printDirLogs(os.Stdout, os.Stderr, args[0], podNames, fromWorkload); err != nil {
				log.Fatalf("%v\n", err)
			}
			return
		}

		markers := make([]string, 0, len(podNames))
		for _, podName := range podNames {
			markers = append(markers, logMarker(podName))
		}
		f, err := os.Open(dumpFile)
		if err != nil {
			log.Fatalf("Error to read [file=%v]: %v", dumpFile, err.Error())
		}
		defer f.Close()
		if err := printLogs(f, os.Stdout, markers); err != nil {
			log.Fatalf("Error while reading [file=%v]: %v", dumpFile, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVarP(&resNamespace, ns, "n", "default", "namespace of the pod or workload")
	logsCmd.Flags().StringVarP(&resContainer, cont, "c", "", "container")
	logsCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	logsCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump directory")
}

// resolveLogTarget returns the pods whose logs should be printed. A plain name
// is a pod name; TYPE/NAME can select all pods of a Deployment, DaemonSet or
// StatefulSet.
func resolveLogTarget(target string) ([]string, bool, error) {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) == 1 {
		return []string{target}, false, nil
	}

	resource, name := parts[0], parts[1]
	if resource == "" || name == "" {
		return nil, false, fmt.Errorf("invalid target %q: expected TYPE/NAME", target)
	}

	kind, err := getKind(resource)
	if err != nil {
		return nil, false, fmt.Errorf("%s is not a supported resource type", resource)
	}

	switch kind {
	case "Pod":
		return []string{name}, false, nil
	case "Deployment", "DaemonSet", "StatefulSet":
		pods, err := resolveWorkloadPods(kind, resNamespace, name)
		return pods, true, err
	default:
		return nil, false, fmt.Errorf("logs does not support %s; use a pod, deployment, daemonset or statefulset", resource)
	}
}

func logMarker(podName string) string {
	if len(resContainer) > 0 {
		return "container " + resContainer + " of pod " + resNamespace + "/" + podName
	}
	return resNamespace + "/" + podName
}

type podLogFile struct {
	path      string
	marker    string
	container string
}

func printDirLogs(out, errOut io.Writer, target string, podNames []string, fromWorkload bool) error {
	printed := 0
	for _, podName := range podNames {
		logFiles, err := logFilesForPodInDir(podName)
		if err != nil {
			if fromWorkload {
				fmt.Fprintf(errOut, "No log found for pod %s/%s: %v\n", resNamespace, podName, err)
				continue
			}
			return err
		}

		for _, logFile := range logFiles {
			f, err := os.Open(logFile.path)
			if err != nil {
				if fromWorkload {
					fmt.Fprintf(errOut, "No log found for pod %s/%s: %v\n", resNamespace, podName, err)
					continue
				}
				return fmt.Errorf("error reading [file=%s]: %w", logFile.path, err)
			}

			// sosreport stores one raw log file per container without the
			// "==== START logs" markers, so add markers around the container's
			// logs in the same format as cluster-info dump's logs.txt.
			showHeader := logFile.marker == ""
			if showHeader {
				writeLogHeader(out, "==== START", logFile.container, resNamespace, podName)
			}

			err = printLogs(f, out, []string{logFile.marker})
			f.Close()
			if err != nil {
				return fmt.Errorf("error reading [file=%s]: %w", logFile.path, err)
			}
			if showHeader {
				writeLogHeader(out, "==== END", logFile.container, resNamespace, podName)
			}
			printed++
		}
	}

	if printed == 0 {
		return fmt.Errorf("no logs found for %s", target)
	}
	return nil
}

func writeLogHeader(w io.Writer, prefix, container, namespace, pod string) {
	if container != "" {
		fmt.Fprintf(w, "%s logs for container %s of pod %s/%s ====\n", prefix, container, namespace, pod)
		return
	}
	fmt.Fprintf(w, "%s logs for pod %s/%s ====\n", prefix, namespace, pod)
}

func logFilesForPodInDir(podName string) ([]podLogFile, error) {
	marker := logMarker(podName)
	if isSosreportDir(dumpDir) {
		// Each file contains a single container's raw logs, so there are no
		// START/END markers to search for.
		podLogDir := filepath.Join(dumpDir, resNamespace, "podlogs", podName)
		entries, err := os.ReadDir(podLogDir)
		if err != nil {
			return nil, fmt.Errorf("no logs for pod %s/%s: %w", resNamespace, podName, err)
		}

		var names []string
		for _, entry := range entries {
			if !entry.IsDir() {
				names = append(names, entry.Name())
			}
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("no logs for pod %s/%s", resNamespace, podName)
		}
		sort.Strings(names)

		if len(resContainer) > 0 {
			suffix := "_--namespace_" + resNamespace + "_logs_" + podName + "_-c_" + resContainer
			for _, name := range names {
				if strings.HasSuffix(name, suffix) {
					return []podLogFile{{
						path:      filepath.Join(podLogDir, name),
						marker:    "",
						container: resContainer,
					}}, nil
				}
			}
			return nil, fmt.Errorf("container %q not found for pod %s/%s", resContainer, resNamespace, podName)
		}

		logFiles := make([]podLogFile, 0, len(names))
		for _, name := range names {
			logFiles = append(logFiles, podLogFile{
				path:      filepath.Join(podLogDir, name),
				marker:    "",
				container: containerFromSosLogFileName(name),
			})
		}
		return logFiles, nil
	}

	return []podLogFile{{path: filepath.Join(dumpDir, resNamespace, podName, "logs.txt"), marker: marker}}, nil
}

func containerFromSosLogFileName(name string) string {
	const marker = "_-c_"
	index := strings.LastIndex(name, marker)
	if index < 0 {
		return ""
	}
	return name[index+len(marker):]
}

// printLogs copies the log sections identified by markers from r to w. Markers
// are either "namespace/pod" or "container NAME of pod namespace/pod".
func printLogs(r io.Reader, w io.Writer, markers []string) error {
	reader := bufio.NewReaderSize(r, 64*1024)
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	printAll := false
	remaining := make(map[string]struct{}, len(markers))
	for _, marker := range markers {
		if marker == "" {
			printAll = true
			continue
		}
		remaining[marker] = struct{}{}
	}
	if !printAll && len(remaining) == 0 {
		return nil
	}

	// A marker containing a container name identifies exactly one log section,
	// so there is no need to keep scanning once all of them have ended.
	stopAtEnd := len(resContainer) > 0 && !printAll
	canPrint := false

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			if printAll {
				writer.WriteString(line)
				writer.WriteByte('\n')
			} else {
				if podMarker, containerMarker, ok := parseLogHeader(line, logStartPrefix); ok {
					if markerMatches(remaining, podMarker, containerMarker) {
						canPrint = true
					}
				}
				if canPrint {
					writer.WriteString(line)
					writer.WriteByte('\n')
				}
				if podMarker, containerMarker, ok := parseLogHeader(line, logEndPrefix); ok {
					if markerMatches(remaining, podMarker, containerMarker) {
						canPrint = false
						if stopAtEnd {
							delete(remaining, podMarker)
							delete(remaining, containerMarker)
							if len(remaining) == 0 {
								return nil
							}
						}
					}
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

func parseLogHeader(line, prefix string) (podMarker, containerMarker string, ok bool) {
	if !strings.HasPrefix(line, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(line, prefix)
	rest = strings.TrimSuffix(rest, "====")
	rest = strings.TrimSpace(rest)
	index := strings.Index(rest, " of pod ")
	if index < 0 {
		return "", "", false
	}
	container := rest[:index]
	pod := rest[index+len(" of pod "):]
	if container == "" || pod == "" {
		return "", "", false
	}
	return pod, "container " + container + " of pod " + pod, true
}

func markerMatches(markers map[string]struct{}, podMarker, containerMarker string) bool {
	if _, ok := markers[containerMarker]; ok {
		return true
	}
	_, ok := markers[podMarker]
	return ok
}
