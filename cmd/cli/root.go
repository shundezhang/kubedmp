package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/shundezhang/kubedmp/cmd/build"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ProcessBuffer func(string)

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

	SupportTypes = map[string][]string{
		"Node":                    {"no", "node", "nodes"},
		"Pod":                     {"po", "pod", "pods"},
		"Service":                 {"svc", "service", "services"},
		"Deployment":              {"deploy", "deployment", "deployments"},
		"DaemonSet":               {"ds", "daemonset", "daemonsets"},
		"ReplicaSet":              {"rs", "replicaset", "replicasets"},
		"Event":                   {"event", "events"},
		"Persistent Volume":       {"pv", "persistentvolumes"},
		"Persistent Volume Claim": {"pvc", "persistentvolumeclaim", "persistentvolumeclaims"},
		"StatefulSet":             {"sts", "statefulset", "statefulsets"},
		"Secret":                  {"secrets", "secret"},
		"ConfigMap":               {"cm", "configmap", "configmaps"},
		"Service Account":         {"sa", "serviceaccount", "serviceaccounts"},
		"Ingress":                 {"ing", "ingress", "ingresses"},
		"Storage Class":           {"sc", "storageclass", "storageclasses"},
		"Cluster Role":            {"clusterrole", "clusterroles"},
		"Cluster Role Binding":    {"clusterrolebinding", "clusterrolebindings"},
		"Endpoints":               {"ep", "endpoint", "endpoints"},
		"Job":                     {"job", "jobs"},
		"Cron Job":                {"cj", "cronjob", "cronjobs"},
		"Role":                    {"role", "roles"},
		"Role Binding":            {"rolebinding", "rolebindings"},
	}

	UnnamespacedTypes = []string{"Node", "Persistent Volume", "Storage Class", "Cluster Role", "Cluster Role Binding"}
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

func readFile(filePath string, pb ProcessBuffer) {
	// log.Print(filePath)
	_, error := os.Stat(filePath)

	// check if error is "file not exists"
	if os.IsNotExist(error) {
		return
	}
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error to read [file=%v]: %v", filePath, err.Error())
	}
	var buffer string
	var inject bool

	scanner := bufio.NewScanner(f)
	defer f.Close()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "{" {
			buffer = line
			inject = true
		} else if line == "}" && inject {
			buffer += line
			inject = false
			pb(buffer)
			buffer = ""
		} else if inject {
			buffer += line
		}
	}

}
