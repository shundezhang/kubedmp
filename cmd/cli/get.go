package cli

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

const (
	ns = "namespace"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get a resource or resources",
	Long:  `get a resource or resources`,
	Run: func(cmd *cobra.Command, args []string) {
		dumpFile, err := cmd.Flags().GetString(dumpFile)
		if err != nil {
			log.Fatalf("Please provide a dump file\n")
			return
		}
		namespace, err := cmd.Flags().GetString(ns)
		if err != nil {
			log.Fatalf("Error parsing namespace\n")
			return
		}

		if len(args) == 0 {
			log.Fatalf("Please specify a type, e.g. pod, svc, deploy, daemonset, event\n")
			return
		}
		queryType := args[0]
		var objectName string
		if len(args) > 1 {
			objectName = args[1]
		}
		fmt.Printf("parsing dump file %s\n", dumpFile)
		show(dumpFile, func(buffer string, queryType, namespace, objectName string) {
			var result map[string]interface{}
			// fmt.Println(buffer)
			err := json.Unmarshal([]byte(buffer), &result)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println(buffer)
				return
			}
		}, queryType, namespace, objectName)

	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().String(ns, "kube-system", "namespace")
}
