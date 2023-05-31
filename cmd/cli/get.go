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
			fmt.Printf("Please specify a type, e.g. node/no, pod, svc/service, deploy/deployment, daemonset, event\n")
			return
		}
		queryType := args[0]
		objectName := ""
		if len(args) > 1 {
			objectName = args[1]
		}
		if !contains([]string{"no", "node", "po", "pod"}, queryType) {
			fmt.Printf("%s is not a supported resource.\n", queryType)
			return
		}
		fmt.Printf("In get: parsing dump file %s\n", dumpFile)
		show(dumpFile, func(buffer string, queryType, namespace, objectName string) {
			var result map[string]interface{}
			// fmt.Println(buffer)
			err := json.Unmarshal([]byte(buffer), &result)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println(buffer)
				return
			}
			switch queryType {
			case "no", "node":
				if result["kind"] == "NodeList" {
					prettyPrintNodeList(result["items"].([]interface{}), objectName)
				}
			case "po", "pod":
				if result["kind"] == "PodList" {
					prettyPrintPodList(result["items"].([]interface{}), namespace, objectName)
				}
			}
		}, queryType, namespace, objectName)

	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().String(ns, "kube-system", "namespace")
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
