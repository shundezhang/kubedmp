package cli

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
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
		allNS, err := cmd.Flags().GetBool(an)
		if err != nil {
			log.Fatalf("Error parsing all-namespace\n")
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
		if !contains([]string{"no", "node", "po", "pod", "svc", "service", "deploy", "deployment", "ds", "daemonset", "rs", "replicaset", "event"}, queryType) {
			fmt.Printf("%s is not a supported resource.\n", queryType)
			return
		}
		// fmt.Printf("In get: parsing dump file %s\n", dumpFile)
		show(dumpFile, func(buffer string, queryType, namespace, objectName string) []interface{} {
			var result map[string]interface{}
			// fmt.Println(buffer)
			err := json.Unmarshal([]byte(buffer), &result)
			items := make([]interface{}, 0)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println(buffer)
				return items
			}
			switch queryType {
			case "no", "node":
				if result["kind"] == "NodeList" {
					for _, item := range result["items"].([]interface{}) {
						node := item.(map[string]interface{})
						metadata := node["metadata"].(map[string]interface{})
						nodeName := metadata["name"].(string)
						if objectName != "" && nodeName != objectName {
							continue
						}
						items = append(items, item)
					}
				}
			case "po", "pod":
				// fmt.Printf("showing ns %s pod %s all-ns %t\n", namespace, objectName, allNS)
				if result["kind"] == "PodList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			case "svc", "service":
				if result["kind"] == "ServiceList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			case "deploy", "deployment":
				if result["kind"] == "DeploymentList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			case "ds", "daemonset":
				if result["kind"] == "DaemonSetList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			case "rs", "replicaset":
				if result["kind"] == "ReplicaSetList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			case "event":
				if result["kind"] == "EventList" {
					items = findItems(result["items"].([]interface{}), allNS, namespace, objectName)
				}
			}
			// fmt.Println(items)
			return items
		}, queryType, namespace, objectName)

	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringP(ns, "n", "default", "namespace")
	getCmd.Flags().BoolP(an, "A", false, "If present, list the requested object(s) across all namespaces.")
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func findItems(items []interface{}, allNS bool, namespace string, objectName string) []interface{} {
	result := make([]interface{}, 0)
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		metadata := pod["metadata"].(map[string]interface{})
		// fmt.Printf("object ns %s pod %s \n", metadata["namespace"], metadata["name"])
		if !allNS && namespace != "" && namespace != metadata["namespace"] {
			continue
		}
		if objectName != "" && objectName != metadata["name"] {
			continue
		}
		result = append(result, item)
	}
	return result
}
