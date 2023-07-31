package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func prettyPrint(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)
	if err != nil {
		log.Fatalf("Error processing buffer: %v\n%v\n", err.Error(), buffer)
	}
	// fmt.Println("items: ", reflect.TypeOf(result["items"]).String())
	if len(result["items"].([]interface{})) > 0 {
		fmt.Println("Kind: ", result["kind"])
		fmt.Println("================================================")
		switch result["kind"] {
		case "NodeList":
			prettyPrintNodeList(result["items"].([]interface{}))
		case "PodList":
			prettyPrintPodList(result["items"].([]interface{}))
		case "ServiceList":
			prettyPrintServiceList(result["items"].([]interface{}))
		case "DeploymentList":
			prettyPrintDeploymentList(result["items"].([]interface{}))
		case "DaemonSetList":
			prettyPrintDaemonSetList(result["items"].([]interface{}))
		case "ReplicaSetList":
			prettyPrintReplicaSetList(result["items"].([]interface{}))
		case "EventList":
			prettyPrintEventList(result["items"].([]interface{}))
		}
		fmt.Println()
	}
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show all objects in cluster info dump file in ps output format",
	Long:  `show all objects in cluster info dump file in ps output format`,
	Run: func(cmd *cobra.Command, args []string) {
		// dumpFile, err := cmd.Flags().GetString(dumpFileFlag)
		// if err != nil {
		// 	log.Fatalf("Please provide a dump file\n")
		// 	return
		// }
		if len(dumpDir) > 0 {
			dirInfo, err := os.Stat(dumpDir)
			if err != nil {
				log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err.Error())
			}
			if !dirInfo.IsDir() {
				log.Fatalf("Path (%v) is not a dir.", dumpDir)
			}
			subdirs, err1 := os.ReadDir(dumpDir)
			if err1 != nil {
				log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err1.Error())
			}
			readFile(filepath.Join(dumpDir, "nodes."+dumpFormat), prettyPrint)
			// fmt.Println("-------------")
			for _, dir := range subdirs {
				subdirInfo, _ := os.Stat(filepath.Join(dumpDir, dir.Name()))
				if !subdirInfo.IsDir() {
					continue
				}
				// fmt.Println("Showing namespace:", dir.Name())
				readFile(filepath.Join(dumpDir, dir.Name(), "events."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, dir.Name(), "services."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, dir.Name(), "daemonsets."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, dir.Name(), "deployments."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, dir.Name(), "replicasets."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, dir.Name(), "pods."+dumpFormat), prettyPrint)
				// readFile(dumpDir, prettyPrint, "event", dir.Name(), "")
				// fmt.Println("-------------")
				// readFromDir(dumpDir, prettyPrint, "svc", dir.Name(), "")
				// fmt.Println("-------------")
				// readFromDir(dumpDir, prettyPrint, "ds", dir.Name(), "")
				// fmt.Println("-------------")
				// readFromDir(dumpDir, prettyPrint, "deploy", dir.Name(), "")
				// fmt.Println("-------------")
				// readFromDir(dumpDir, prettyPrint, "rs", dir.Name(), "")
				// fmt.Println("-------------")
				// readFromDir(dumpDir, prettyPrint, "po", dir.Name(), "")
				// fmt.Println("-------------")
			}
		} else {
			readFile(dumpFile, prettyPrint)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
