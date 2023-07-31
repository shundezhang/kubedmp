package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/shundezhang/kubedmp/cmd/build"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ProcessBuffer func(string)

var (
	dumpFileFlag = "dumpfile"
	dumpDirFlag  = "dumpdir"
	getVersion   bool

	dumpFile   string
	dumpDir    string
	dumpFormat = "json"

	resType       string
	resNamespace  string
	resName       string
	resContainer  string
	allNamespaces bool
)

const (
	ns = "namespace"
	an = "all-namespaces"
)

var rootCmd = &cobra.Command{
	Use:   "kubedmp",
	Short: "Display k8s cluster-info dump in ps format",
	Long:  "Display k8s cluster-info dump in ps format",
	Run: func(cmd *cobra.Command, args []string) {
		if getVersion {
			fmt.Println("Version:\t", build.Version)
			fmt.Println("build.Time:\t", build.Time)
			fmt.Println("build.User:\t", build.User)
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
	rootCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	rootCmd.Flags().BoolVarP(&getVersion, "version", "v", false, "get version")
	rootCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump dir")
}

func initConfig() {
	viper.AutomaticEnv()
}

func hasType(resType string) bool {
	if !contains([]string{"no", "node", "po", "pod", "svc", "service", "deploy", "deployment", "ds", "daemonset", "rs", "replicaset", "event"}, resType) {
		log.Fatalf("%s is not a supported resource.\n", resType)
		return false
	}
	return true
}

func readFile(filePath string, pb ProcessBuffer) {
	// log.Print(filePath)
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
		} else if line == "}" {
			buffer += line
			inject = false
			pb(buffer)
			buffer = ""
		} else if inject {
			buffer += line
		}
	}

}
