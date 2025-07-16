package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func prettyPrint(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)
	if err != nil {
		// log.Fatalf("Error processing buffer: %v\n%v\n", err.Error(), buffer)
		return
	}
	// fmt.Println("items: ", reflect.TypeOf(result["items"]).String())
	kind := result["kind"]
	if result["kind"] == "List" {
		kind = resKind + "List"
	}
	if result["items"] == nil {
		return
	}
	if kind != nil && len(result["items"].([]interface{})) > 0 {
		fmt.Println("Kind: ", kind)
		fmt.Println("================================================")
		switch kind {
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
		case "StatefulSetList":
			prettyPrintStatefulSetList(result["items"].([]interface{}))
		case "EventList":
			prettyPrintEventList(result["items"].([]interface{}))
		case "PersistentVolumeList":
			prettyPrintPersistentVolumeList(result["items"].([]interface{}))
		case "PersistentVolumeClaimList":
			prettyPrintPersistentVolumeClaimList(result["items"].([]interface{}))
		case "ConfigMapList":
			prettyPrintConfigMapList(result["items"].([]interface{}))
		case "SecretList":
			prettyPrintSecretList(result["items"].([]interface{}))
		case "ServiceAccountList":
			prettyPrintServiceAccountList(result["items"].([]interface{}))
		case "IngressList":
			prettyPrintIngressList(result["items"].([]interface{}))
		case "StorageClassList":
			prettyPrintStorageClassList(result["items"].([]interface{}))
		case "ClusterRoleList":
			prettyPrintClusterRoleList(result["items"].([]interface{}))
		case "ClusterRoleBindingList":
			prettyPrintClusterRoleBindingList(result["items"].([]interface{}))
		case "EndpointsList":
			prettyPrintEndpointsList(result["items"].([]interface{}))
		case "JobList":
			prettyPrintJobList(result["items"].([]interface{}))
		case "CronJobList":
			prettyPrintCronJobList(result["items"].([]interface{}))
		case "RoleList":
			prettyPrintRoleList(result["items"].([]interface{}))
		case "RoleBindingList":
			prettyPrintRoleBindingList(result["items"].([]interface{}))
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

			dumpDirPath, _ := filepath.Abs(dumpDir)

			if strings.Contains(dumpDirPath, "sos_commands") && strings.Contains(dumpDirPath, "kubernetes") {
				resNamespace = ""
				subdirs, err1 := os.ReadDir(dumpDir)
				if err1 != nil {
					log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err1.Error())
				}
				for _, dir := range subdirs {
					// fmt.Println("dir.Name(): ", dir.Name())
					if dir.IsDir() {
						resFiles, err2 := os.ReadDir(filepath.Join(dumpDir, dir.Name()))
						// resType = dir.Name()
						if err2 != nil {
							log.Fatalf("Error to open [dir=%v]: %v", dir.Name(), err2.Error())
						}
						for _, resFile := range resFiles {
							if strings.Contains(resFile.Name(), "_get_-o_json_") {
								// fmt.Println("kind: ", resFile.Name()[strings.LastIndex(resFile.Name(), "_")+1:])
								resKind, _ = getKind(resFile.Name()[strings.LastIndex(resFile.Name(), "_")+1:])
								// fmt.Println("resKind: ", resKind)
								itemFilename := filepath.Join(dumpDir, dir.Name(), resFile.Name())
								// fmt.Println("itemFilename: ", itemFilename)
								readFile(itemFilename, prettyPrint)
							}
						}
					} else if !dir.IsDir() && strings.Contains(dir.Name(), "_get_-o_json_") {
						// fmt.Println("kind: ", dir.Name()[strings.LastIndex(dir.Name(), "_")+1:])
						resKind, _ = getKind(dir.Name()[strings.LastIndex(dir.Name(), "_")+1:])
						itemFilename := filepath.Join(dumpDir, dir.Name())
						// fmt.Println("resKind: ", resKind)
						// fmt.Println("itemFilename: ", itemFilename)
						readFile(itemFilename, prettyPrint)
					}
				}
			} else {
				readFile(filepath.Join(dumpDir, "nodes."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, "pvs."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, "scs."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, "clusterroles."+dumpFormat), prettyPrint)
				readFile(filepath.Join(dumpDir, "clusterrolebindings."+dumpFormat), prettyPrint)
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
					readFile(filepath.Join(dumpDir, dir.Name(), "statefulsets."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "pods."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "pvcs."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "configmaps."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "secrets."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "serviceaccounts."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "ingresses."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "endpoints."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "jobs."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "cronjobs."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "roles."+dumpFormat), prettyPrint)
					readFile(filepath.Join(dumpDir, dir.Name(), "rolebindings."+dumpFormat), prettyPrint)
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
			}
		} else {
			readFile(dumpFile, prettyPrint)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	showCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump directory")
}
