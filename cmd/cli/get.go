package cli

import (
	// "bufio"
	"encoding/json"
	// "fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	displayItems []interface{}
)

var getCmd = &cobra.Command{
	Use:                   "get TYPE [-n NAMESPACE | -A]",
	DisableFlagsInUseLine: true,
	Short:                 "Display one or many resources",
	Long: `Display one or many resources of a type, which can be node/no, pod/po, service/svc, deployment/deploy, daemonset/ds, replicaset/rs or event. 
Prints a table of the most important information about resources of the specific type.`,
	Example: `  # Lists all pods in kube-system namespace in ps output format, the output contains all fields in 'kubectl get -o wide'
  kubedmp get po -n kube-system
  
  # List all nodes
  kubedmp get no`,
	// Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Println("dumpdir", len(dumpDir))
		// dumpFile, err := cmd.Flags().GetString(dumpFileFlag)
		// if err != nil {
		// 	log.Fatalf("Please provide a dump file\n")
		// 	return
		// }
		if len(args) == 0 {
			log.Fatalf("Please specify a type: node/no, pod/po, service/svc, deployment/deploy, daemonset/ds, replicaset/rs, event\n")
			return
		}
		// namespace := ""
		// allNS, err := cmd.Flags().GetBool(an)
		// if err != nil {
		// 	log.Fatalf("Error parsing all-namespace flag\n")
		// 	return
		// }
		// if !allNS {
		// 	namespace, err = cmd.Flags().GetString(ns)
		// 	if err != nil {
		// 		log.Fatalf("Error parsing namespace flag\n")
		// 		return
		// 	}
		// }
		resType = args[0]
		resName = ""
		if len(args) > 1 {
			resName = args[1]
		}
		if !hasType(resType) {
			return
		}
		if strings.HasPrefix(resType, "no") {
			resNamespace = ""
		}
		// fmt.Printf("In get: parsing dump file %s\n", dumpFile)
		displayItems = make([]interface{}, 0)
		if len(dumpDir) > 0 {
			traverseDir()
		} else {
			readFile(dumpFile, processDoc)
		}
		printItems()
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&resNamespace, ns, "n", "default", "namespace of the resources, not applicable to node")
	getCmd.Flags().BoolVarP(&allNamespaces, an, "A", false, "If present, list the requested object(s) across all namespaces.")
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func processDoc(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)

	if err != nil {
		log.Fatalf("Error processing buffer: %v\n%v\n", err.Error(), buffer)
	}
	// log.Print(resType, resNamespace, resName, result["kind"])
	if (resType == "no" || resType == "node") && result["kind"] == "NodeList" {
		for _, item := range result["items"].([]interface{}) {
			node := item.(map[string]interface{})
			metadata := node["metadata"].(map[string]interface{})
			nodeName := metadata["name"].(string)
			if resName != "" && nodeName != resName {
				continue
			}
			displayItems = append(displayItems, item)
		}
	} else if (resType == "po" || resType == "pod") && result["kind"] == "PodList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "svc" || resType == "service") && result["kind"] == "ServiceList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "deploy" || resType == "deployment") && result["kind"] == "DeploymentList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "ds" || resType == "daemonset") && result["kind"] == "DaemonSetList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "rs" || resType == "replicaset") && result["kind"] == "ReplicaSetList" {
		findItems(result["items"].([]interface{}))
		// } else if namespace == metadata["namespace"] && resName == metadata["name"] {
		// 	fmt.Println("item: ", item)
	}
}

func findItems(items []interface{}) {
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		res := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		metadata := res["metadata"].(map[string]interface{})
		// fmt.Printf("object ns %s pod %s \n", metadata["namespace"], metadata["name"])
		if !allNamespaces && resNamespace != "" && resNamespace != metadata["namespace"] {
			continue
		}
		if resName != "" && resName != metadata["name"] {
			continue
		}
		displayItems = append(displayItems, item)
	}
}

func printItems() {
	switch resType {
	case "no", "node":
		prettyPrintNodeList(displayItems)
	case "po", "pod":
		prettyPrintPodList(displayItems)
	case "svc", "service":
		prettyPrintServiceList(displayItems)
	case "deploy", "deployment":
		prettyPrintDeploymentList(displayItems)
	case "ds", "daemonset":
		prettyPrintDaemonSetList(displayItems)
	case "rs", "replicaset":
		prettyPrintReplicaSetList(displayItems)
	case "event":
		prettyPrintEventList(displayItems)
	}
}

func traverseDir() {
	dirInfo, err := os.Stat(dumpDir)
	if err != nil {
		log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err.Error())
	}
	if !dirInfo.IsDir() {
		log.Fatalf("Path (%v) is not a dir.", dumpDir)
	}
	filename := ""
	switch resType {
	case "no", "node":
		filename = "nodes"
	case "po", "pod":
		filename = "pods"
	case "svc", "service":
		filename = "services"
	case "deploy", "deployment":
		filename = "deployments"
	case "ds", "daemonset":
		filename = "daemonsets"
	case "rs", "replicaset":
		filename = "replicasets"
	case "event":
		filename = "events"
	}
	if allNamespaces && !strings.HasPrefix(resType, "no") {
		subdirs, err1 := os.ReadDir(dumpDir)
		if err1 != nil {
			log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err1.Error())
		}
		for _, dir := range subdirs {
			subdirInfo, _ := os.Stat(filepath.Join(dumpDir, dir.Name()))
			if !subdirInfo.IsDir() {
				continue
			}
			itemFilename := filepath.Join(dumpDir, dir.Name(), filename+"."+dumpFormat)
			readFile(itemFilename, processDoc)
		}
	} else {
		if _, err1 := os.Stat(filepath.Join(dumpDir, resNamespace)); os.IsNotExist(err1) {
			log.Fatalf("namespace %v does not exist: %v", resNamespace, err1.Error())
		}
		itemFilename := filepath.Join(dumpDir, resNamespace, filename+"."+dumpFormat)
		readFile(itemFilename, processDoc)
	}
}
