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
		var err error
		resKind, err = getKind(resType)
		if err != nil {
			log.Fatalf("%s is not a supported resource type.\n", resType)
			return
		}
		if contains(UnnamespacedTypes, resKind) {
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
	if result["kind"] == "List" {
		for _, item := range result["items"].([]interface{}) {
			obj := item.(map[string]interface{})
			kind := obj["kind"]
			if kind != resKind {
				continue
			}
			metadata := obj["metadata"].(map[string]interface{})
			objName := metadata["name"].(string)
			if resName != "" && objName != resName {
				continue
			}
			displayItems = append(displayItems, item)
		}

	} else if resKind == result["kind"].(string)[0:len(result["kind"].(string))-4] {
		if contains(UnnamespacedTypes, resKind) {
			for _, item := range result["items"].([]interface{}) {
				obj := item.(map[string]interface{})
				metadata := obj["metadata"].(map[string]interface{})
				objName := metadata["name"].(string)
				if resName != "" && objName != resName {
					continue
				}
				displayItems = append(displayItems, item)
			}
		} else {
			findItems(result["items"].([]interface{}))
		}
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
	case "sc", "storageclass", "storageclasses":
		prettyPrintStorageClassList(displayItems)
	case "clusterrole", "clusterroles":
		prettyPrintClusterRoleList(displayItems)
	case "clusterrolebinding", "clusterrolebindings":
		prettyPrintClusterRoleBindingList(displayItems)
	case "ep", "endpoint", "endpoints":
		prettyPrintEndpointsList(displayItems)
	case "job", "jobs":
		prettyPrintJobList(displayItems)
	case "cj", "cronjob", "cronjobs":
		prettyPrintCronJobList(displayItems)
	case "role", "roles":
		prettyPrintRoleList(displayItems)
	case "rolebinding", "rolebindings":
		prettyPrintRoleBindingList(displayItems)
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
	filename := DumpFileNames[resKind]
	// fmt.Println("filename: ", filename)
	dumpDirPath, _ := filepath.Abs(dumpDir)
	// fmt.Println("fullPath: ", dumpDirPath)
	if strings.Contains(dumpDirPath, "sos_commands") && strings.Contains(dumpDirPath, "kubernetes") {
		subdirs, err1 := os.ReadDir(dumpDir)
		if err1 != nil {
			log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err1.Error())
		}
		for _, dir := range subdirs {
			// fmt.Println("dir: ", dir)
			// subdirInfo, _ := os.Stat(filepath.Join(dumpDir, dir.Name()))
			// fmt.Println("subdirInfo.Name(): ", subdirInfo.Name())
			// fmt.Println("subdirInfo.IsDir(): ", subdirInfo.IsDir())
			// fmt.Println("!contains(UnnamespacedTypes, resKind): ", !contains(UnnamespacedTypes, resKind))
			// fmt.Println("subdirInfo.Name() == filename: ", subdirInfo.Name() == filename)
			if dir.IsDir() && !contains(UnnamespacedTypes, resKind) && dir.Name() == filename {
				resFiles, err2 := os.ReadDir(filepath.Join(dumpDir, dir.Name()))
				if err2 != nil {
					log.Fatalf("Error to open [dir=%v]: %v", dir.Name(), err2.Error())
				}
				// fmt.Println("subdirInfo.Name(): ", subdirInfo.Name())
				for _, resFile := range resFiles {
					if allNamespaces || (!allNamespaces && strings.HasSuffix(resFile.Name(), "_--namespace_"+resNamespace+"_"+filename)) {
						itemFilename := filepath.Join(dumpDir, dir.Name(), resFile.Name())
						// fmt.Println("itemFilename: ", itemFilename)
						readFile(itemFilename, processDoc)
					}
				}
			} else if !dir.IsDir() && strings.HasSuffix(filepath.Base(dir.Name()), filename) {
				itemFilename := filepath.Join(dumpDir, dir.Name())
				// fmt.Println("itemFilename: ", itemFilename)
				readFile(itemFilename, processDoc)
			}
		}
	} else {
		// fmt.Println("filename: ", filename)
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
			// fmt.Println("itemFilename: ", itemFilename)
			readFile(itemFilename, processDoc)
		}
	}
}
