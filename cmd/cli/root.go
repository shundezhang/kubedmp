package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/shundezhang/kubedmp/cmd/build"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ProcessBuffer is called with the contents of one JSON document. The slice is
// only valid for the duration of the call; callbacks must not retain it.
// Returning false stops reading the file.
type ProcessBuffer func([]byte) bool

var (
	dumpFileFlag = "dumpfile"
	dumpDirFlag  = "dumpdir"
	getVersion   bool
	getTypes     bool

	dumpFile   string
	dumpDir    string
	dumpFormat = "json"

	resType       string
	resNamespace  string
	resName       string
	resContainer  string
	allNamespaces bool
	resKind       string

	SupportTypes = map[string][]string{
		"Node":                  {"no", "node", "nodes"},
		"Pod":                   {"po", "pod", "pods"},
		"Service":               {"svc", "service", "services"},
		"Deployment":            {"deploy", "deployment", "deployments"},
		"DaemonSet":             {"ds", "daemonset", "daemonsets"},
		"ReplicaSet":            {"rs", "replicaset", "replicasets"},
		"Event":                 {"event", "events"},
		"PersistentVolume":      {"pv", "persistentvolumes"},
		"PersistentVolumeClaim": {"pvc", "persistentvolumeclaim", "persistentvolumeclaims"},
		"StatefulSet":           {"sts", "statefulset", "statefulsets"},
		"Secret":                {"secrets", "secret"},
		"ConfigMap":             {"cm", "configmap", "configmaps"},
		"ServiceAccount":        {"sa", "serviceaccount", "serviceaccounts"},
		"Ingress":               {"ing", "ingress", "ingresses"},
		"StorageClass":          {"sc", "storageclass", "storageclasses"},
		"ClusterRole":           {"clusterrole", "clusterroles"},
		"ClusterRoleBinding":    {"clusterrolebinding", "clusterrolebindings"},
		"Endpoints":             {"ep", "endpoint", "endpoints"},
		"Job":                   {"job", "jobs"},
		"CronJob":               {"cj", "cronjob", "cronjobs"},
		"Role":                  {"role", "roles"},
		"RoleBinding":           {"rolebinding", "rolebindings"},
	}

	DumpFileNames = map[string]string{
		"Node":                  "nodes",
		"Pod":                   "pods",
		"Service":               "services",
		"Deployment":            "deployments",
		"DaemonSet":             "daemonsets",
		"ReplicaSet":            "replicasets",
		"Event":                 "events",
		"PersistentVolume":      "pv",
		"PersistentVolumeClaim": "pvc",
		"StatefulSet":           "statefulsets",
		"Secret":                "secrets",
		"ConfigMap":             "configmaps",
		"ServiceAccount":        "serviceaccounts",
		"Ingress":               "ingresses",
		"StorageClass":          "sc",
		"ClusterRole":           "clusterroles",
		"ClusterRoleBinding":    "clusterrolebindings",
		"Endpoints":             "endpoints",
		"Job":                   "jobs",
		"CronJob":               "cronjobs",
		"Role":                  "roles",
		"RoleBinding":           "rolebindings",
	}

	UnnamespacedTypes = []string{"Node", "PersistentVolume", "StorageClass", "ClusterRole", "ClusterRoleBinding"}
)

const (
	ns = "namespace"
	an = "all-namespaces"
)

var rootCmd = &cobra.Command{
	Use:                   "kubedmp [-f dump/file | -d dump/dir] command",
	DisableFlagsInUseLine: true,
	Short:                 "Display k8s cluster-info dump in ps format",
	Long:                  "Display k8s cluster-info dump in ps format",
	Run: func(cmd *cobra.Command, args []string) {
		if getVersion {
			fmt.Println("Version:\t", build.Version)
			fmt.Println("build.Time:\t", build.Time)
			fmt.Println("build.User:\t", build.User)
			os.Exit(0)
		}
		if getTypes {
			writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
			fmt.Println("Supported types: ")
			for name, types := range SupportTypes {
				fmt.Fprintf(writer, "  %s:\t%s\n", name, strings.Join(types, " "))
			}
			writer.Flush()
			os.Exit(0)
		}
		cmd.Help()
		os.Exit(0)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().BoolVarP(&getVersion, "version", "v", false, "get version")
	rootCmd.Flags().BoolVarP(&getTypes, "types", "t", false, "show supported resource types")
}

func initConfig() {
	viper.AutomaticEnv()
}

func hasType(resType string) bool {
	for _, types := range SupportTypes {
		if contains(types, resType) {
			return true
		}
	}

	log.Fatalf("%s is not a supported resource type.\n", resType)
	return false

}

func inType(resType string, key string) bool {
	if contains(SupportTypes[key], resType) {
		return true
	}
	return false
}

func getKind(resType string) (string, error) {
	for kind, types := range SupportTypes {
		if contains(types, resType) {
			return kind, nil
		}
	}

	return "", errors.New("resource type not supported")
}

// parseResourceArgs parses resource arguments in either "TYPE NAME" or
// "TYPE/NAME" form. The name is optional so commands like get can list a type.
func parseResourceArgs(args []string) (resourceType, resourceName string, err error) {
	if len(args) == 0 {
		return "", "", errors.New("resource type is required")
	}

	if strings.Contains(args[0], "/") {
		parts := strings.SplitN(args[0], "/", 2)
		if parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
			return "", "", fmt.Errorf("invalid resource %q: expected TYPE/NAME", args[0])
		}
		if len(args) > 1 {
			return "", "", fmt.Errorf("unexpected argument %q after %q", args[1], args[0])
		}
		return parts[0], parts[1], nil
	}

	resourceType = args[0]
	if len(args) > 1 {
		resourceName = args[1]
	}
	return resourceType, resourceName, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// isSosreportDir reports whether dir uses the sosreport cluster-info layout.
func isSosreportDir(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return strings.Contains(abs, "sos_commands") && strings.Contains(abs, "kubernetes")
}

// docKind returns the top-level "kind" field of a JSON document without
// decoding the rest of it. Kubernetes writes "kind" first, so this avoids
// decoding large documents that are irrelevant to the current command.
func docKind(data []byte) string {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return ""
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return ""
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		key, ok := tok.(string)
		if !ok {
			return ""
		}
		if key == "kind" {
			var kind string
			if err := dec.Decode(&kind); err != nil {
				return ""
			}
			return kind
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return ""
		}
	}
	return ""
}

// docMatchesKind reports whether a document of the given kind contains
// resources of resourceKind. A generic "List" may contain any resource kind.
func docMatchesKind(kind, resourceKind string) bool {
	if kind == "List" {
		return true
	}
	return strings.HasSuffix(kind, "List") && strings.TrimSuffix(kind, "List") == resourceKind
}

func readFile(filePath string, pb ProcessBuffer) {
	f, err := os.Open(filePath)
	if err != nil {
		// A missing file is expected when a dump directory does not contain a
		// particular resource type.
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("Error to read [file=%v]: %v", filePath, err.Error())
	}
	defer f.Close()

	var buffer bytes.Buffer
	var inject bool

	reader := bufio.NewReaderSize(f, 64*1024)

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if inject {
				if line == "}" {
					buffer.WriteString(line)
					inject = false
					keepGoing := pb(buffer.Bytes())
					buffer.Reset()
					if !keepGoing {
						return
					}
				} else {
					buffer.WriteString(line)
					buffer.WriteByte('\n')
				}
			} else if line == "{" {
				buffer.Reset()
				buffer.WriteString(line)
				inject = true
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Error while reading [file=%v]: %v", filePath, err)
			}
			break
		}
	}
}
