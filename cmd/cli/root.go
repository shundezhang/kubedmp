package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dumpFile = "dumpfile"
)

var rootCmd = &cobra.Command{
	Use:   "k8sinfo",
	Short: "show k8s cluster-info dump",
	Long:  "show k8s cluster-info dump",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP(dumpFile, "f", "./cluster-info-dump", "Path to dump file")
}

func initConfig() {
	viper.AutomaticEnv()
}
