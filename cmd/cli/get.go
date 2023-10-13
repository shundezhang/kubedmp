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
	Long: `Display one or many resources of a type. 
Prints a table of the most important information about resources of the specific type.`,
	Example: `  # Lists all pods in kube-system namespace in ps output format, the output contains all fields in 'kubectl get -o wide'
  kubedmp get po -n kube-system
  
  # List all nodes
  kubedmp get no`,
	// Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			log.Fatalf("Please specify a type.\n")
			return
		}

		resType = args[0]
		resName = ""
		if len(args) > 1 {
			resName = args[1]
		}
		if !hasType(resType) {
			return
		}
		if strings.HasPrefix(resType, "no") || resType == "pv" || resType == "persistentvolumes" {
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
	getCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	getCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump directory")
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
	// fmt.Println("=====================================================")
	err := json.Unmarshal([]byte(buffer), &result)

	if err != nil {
		// log.Fatalf("Error processing buffer: %v\n%v\n", err.Error(), buffer)
		return
	}
	if result["kind"] == nil {
		return
	}
	// log.Print(resType, resNamespace, resName, result["kind"])
	if (resType == "no" || resType == "node" || resType == "nodes") && result["kind"] == "NodeList" {
		for _, item := range result["items"].([]interface{}) {
			node := item.(map[string]interface{})
			metadata := node["metadata"].(map[string]interface{})
			nodeName := metadata["name"].(string)
			if resName != "" && nodeName != resName {
				continue
			}
			displayItems = append(displayItems, item)
		}
	} else if (resType == "pv" || resType == "persistentvolume" || resType == "persistentvolumes") && result["kind"] == "PersistentVolumeList" {
		for _, item := range result["items"].([]interface{}) {
			pv := item.(map[string]interface{})
			metadata := pv["metadata"].(map[string]interface{})
			pvName := metadata["name"].(string)
			if resName != "" && pvName != resName {
				continue
			}
			displayItems = append(displayItems, item)
		}
	} else if (resType == "po" || resType == "pod" || resType == "pods") && result["kind"] == "PodList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "svc" || resType == "service" || resType == "services") && result["kind"] == "ServiceList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "deploy" || resType == "deployment" || resType == "deployments") && result["kind"] == "DeploymentList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "ds" || resType == "daemonset" || resType == "daemonsets") && result["kind"] == "DaemonSetList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "rs" || resType == "replicaset" || resType == "replicasets") && result["kind"] == "ReplicaSetList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "sts" || resType == "statefulset" || resType == "statefulsets") && result["kind"] == "StatefulSetList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "cm" || resType == "configmap" || resType == "configmaps") && result["kind"] == "ConfigMapList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "secrets" || resType == "secret") && result["kind"] == "SecretList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "sa" || resType == "serviceaccounts" || resType == "serviceaccount") && result["kind"] == "ServiceAccountList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "ing" || resType == "ingress" || resType == "ingresses") && result["kind"] == "IngressList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "pvc" || resType == "persistentvolumeclaim" || resType == "persistentvolumeclaims") && result["kind"] == "PersistentVolumeClaimList" {
		findItems(result["items"].([]interface{}))
	} else if (resType == "event" || resType == "events") && result["kind"] == "EventList" {
		findItems(result["items"].([]interface{}))
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
	case "no", "node", "nodes":
		prettyPrintNodeList(displayItems)
	case "po", "pod", "pods":
		prettyPrintPodList(displayItems)
	case "svc", "service", "services":
		prettyPrintServiceList(displayItems)
	case "deploy", "deployment", "deployments":
		prettyPrintDeploymentList(displayItems)
	case "ds", "daemonset", "daemonsets":
		prettyPrintDaemonSetList(displayItems)
	case "rs", "replicaset", "replicasets":
		prettyPrintReplicaSetList(displayItems)
	case "sts", "statefulset", "statefulsets":
		prettyPrintStatefulSetList(displayItems)
	case "event", "events":
		prettyPrintEventList(displayItems)
	case "pv", "persistentvolume", "persistentvolumes":
		prettyPrintPersistentVolumeList(displayItems)
	case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
		prettyPrintPersistentVolumeClaimList(displayItems)
	case "secret", "secrets":
		prettyPrintSecretList(displayItems)
	case "cm", "configmap", "configmaps":
		prettyPrintConfigMapList(displayItems)
	case "sa", "serviceaccount", "serviceaccounts":
		prettyPrintServiceAccountList(displayItems)
	case "ing", "ingress", "ingresses":
		prettyPrintIngressList(displayItems)
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
	case "no", "node", "nodes":
		filename = "nodes"
	case "po", "pod", "pods":
		filename = "pods"
	case "svc", "service", "services":
		filename = "services"
	case "deploy", "deployment", "deployments":
		filename = "deployments"
	case "ds", "daemonsets", "daemonset":
		filename = "daemonsets"
	case "rs", "replicasets", "replicaset":
		filename = "replicasets"
	case "sts", "statefulsets", "statefulset":
		filename = "statefulsets"
	case "event", "events":
		filename = "events"
	case "pv", "persistentvolume", "persistentvolumes":
		filename = "pvs"
	case "pvc", "persistentvolumeclaim", "persistentvolumeclaims":
		filename = "pvcs"
	case "cm", "configmap", "configmaps":
		filename = "configmaps"
	case "secret", "secrets":
		filename = "secrets"
	case "sa", "serviceaccount", "serviceaccounts":
		filename = "serviceaccounts"
	case "ing", "ingress", "ingresses":
		filename = "ingresses"
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
