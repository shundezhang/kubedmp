package cli

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func prettyPrint(buffer string, queryType, namespace, objectName string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(buffer)
		return
	}
	fmt.Println("Kind: ", result["kind"])
	fmt.Println("================================================")
	// fmt.Println("items: ", reflect.TypeOf(result["items"]).String())
	if len(result["items"].([]interface{})) == 0 {
		fmt.Println("No Items")
	} else {
		switch result["kind"] {
		case "NodeList":
			prettyPrintNodeList(result["items"].([]interface{}), "")
		case "PodList":
			prettyPrintPodList(result["items"].([]interface{}), "", "")
		case "ServiceList":
			prettyPrintServiceList(result["items"].([]interface{}))
		case "DeploymentList":
			prettyPrintDeploymentList(result["items"].([]interface{}))
		case "DaemonSetList":
			prettyPrintDaemonSetList(result["items"].([]interface{}))
		case "EventList":
			prettyPrintEventList(result["items"].([]interface{}))
		}
	}
	fmt.Println()
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show details in cluster info dump",
	Long:  `show details in cluster info dump`,
	Run: func(cmd *cobra.Command, args []string) {
		dumpFile, err := cmd.Flags().GetString(dumpFile)
		if err != nil {
			log.Fatalf("Please provide a dump file\n")
			return
		}
		show(dumpFile, prettyPrint, "", "", "")
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
