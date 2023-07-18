package cli

import (
	"fmt"
	"os"

	"github.com/shundezhang/kubedmp/cmd/build"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dumpFile   = "dumpfile"
	getVersion bool
)

const (
	ns = "namespace"
	an = "all-namespaces"
)

var rootCmd = &cobra.Command{
	Use:   "kubedmp",
	Short: "show k8s cluster-info dump",
	Long:  "show k8s cluster-info dump",
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
	rootCmd.PersistentFlags().StringP(dumpFile, "f", "./cluster-info.dump", "Path to dump file")
	rootCmd.Flags().BoolVarP(&getVersion, "version", "v", false, "get version")
}

func initConfig() {
	viper.AutomaticEnv()
}
