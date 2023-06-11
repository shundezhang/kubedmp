package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "describe a resource",
	Long:  `describe a resource`,
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
		if len(args) < 2 {
			fmt.Printf("Please specify a type, e.g. node/no, pod, svc/service, deploy/deployment, daemonset, event, and an object name\n")
			return
		}
		queryType := args[0]
		objectName := args[1]

		if !contains([]string{"no", "node", "po", "pod", "svc", "service", "deploy", "deployment", "ds", "daemonset"}, queryType) {
			fmt.Printf("%s is not a supported resource.\n", queryType)
			return
		}
		f, err := os.Open(dumpFile)
		if err != nil {
			log.Fatalf("Error to read [file=%v]: %v", dumpFile, err.Error())
		}
		var buffer string
		var inject bool

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "{" {
				buffer = line
				inject = true
			} else if line == "}" {
				buffer += line
				inject = false
				describeObject(buffer, queryType, namespace, objectName)
				buffer = ""
			} else if inject {
				buffer += line
			}
		}

		f.Close()

	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.Flags().StringP(ns, "n", "default", "namespace")

}

func describeObject(buffer, queryType, namespace, objectName string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(buffer)
		return
	}

	for _, item := range result["items"].([]interface{}) {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		obj := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		metadata := obj["metadata"].(map[string]interface{})
		// fmt.Printf("object ns %s pod %s \n", metadata["namespace"], metadata["name"])
		if (queryType == "no" || queryType == "node") && objectName == metadata["name"] && result["kind"] == "NodeList" {
			describeNode(item)
		} else if (queryType == "po" || queryType == "pod") && objectName == metadata["name"] && namespace == metadata["namespace"] && result["kind"] == "PodList" {
			describePod(item)
		} else if (queryType == "svc" || queryType == "service") && objectName == metadata["name"] && namespace == metadata["namespace"] && result["kind"] == "ServiceList" {
			describeService(item)
		} else if (queryType == "deploy" || queryType == "deployment") && objectName == metadata["name"] && namespace == metadata["namespace"] && result["kind"] == "DeploymentList" {
			describeDeployment(item)
		} else if (queryType == "ds" || queryType == "daemonset") && objectName == metadata["name"] && namespace == metadata["namespace"] && result["kind"] == "DaemonSetList" {
			describeDaemonSet(item)
		} else if (queryType == "rs" || queryType == "replicaset") && objectName == metadata["name"] && namespace == metadata["namespace"] && result["kind"] == "ReplicaSetList" {
			describeReplicaSet(item)
			// } else if namespace == metadata["namespace"] && objectName == metadata["name"] {
			// 	fmt.Println("item: ", item)
		}

	}
}

func describeNode(item interface{}) {
	node := item.(map[string]interface{})
	metadata := node["metadata"].(map[string]interface{})
	nodeName := metadata["name"].(string)

	// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
	status := node["status"].(map[string]interface{})
	addresses := status["addresses"].([]interface{})
	conditions := status["conditions"].([]interface{})
	nodeInfo := status["nodeInfo"].(map[string]interface{})
	// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
	creationTimeStr := metadata["creationTimestamp"].(string)
	// fmt.Println("creationTimeStr: ", creationTimeStr)
	labels := metadata["labels"].(map[string]interface{})
	annotations := metadata["annotations"].(map[string]interface{})
	role := "<none>"
	roles := []string{}
	labelList := []string{}
	annotationList := []string{}
	for k, v := range labels {
		labelList = append(labelList, k+"="+v.(string))
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			roles = append(roles, strings.Split(k, "/")[1])
		}
	}
	for k, v := range annotations {
		annotationList = append(annotationList, k+": "+v.(string))
	}
	if len(roles) > 0 {
		role = strings.Join(roles, ",")
	}

	spec := node["spec"].(map[string]interface{})

	var state string
	for _, condition := range conditions {
		cond := condition.(map[string]interface{})
		if cond["status"] == "True" {
			state += cond["type"].(string) + " "
		}
	}
	// unschedulable := false
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", nodeName)
	fmt.Fprintf(writer, "Roles:\t%s\n", role)
	if len(labelList) == 0 {
		fmt.Fprintf(writer, "Labels:\t<none>\n")
	} else {
		for i, v := range labelList {
			if i == 0 {
				fmt.Fprintf(writer, "Labels:\t%s\n", v)
			} else {
				fmt.Fprintf(writer, " \t%s\n", v)
			}
		}
	}
	if len(annotationList) == 0 {
		fmt.Fprintf(writer, "annotationList:\t<none>\n")
	} else {
		for i, v := range annotationList {
			if i == 0 {
				fmt.Fprintf(writer, "Annotations:\t%s\n", v)
			} else {
				fmt.Fprintf(writer, " \t%s\n", v)
			}
		}

	}

	creationTime, err := time.Parse("2006-01-02T15:04:05Z", creationTimeStr)
	if err == nil {
		fmt.Fprintf(writer, "CreationTimestamp:\t%s\n", creationTime.Format(time.RFC1123Z))
	}

	if taints, ok := spec["taints"].([]interface{}); ok {
		for i, v := range taints {
			taint := v.(map[string]interface{})
			if i == 0 {
				fmt.Fprintf(writer, "Taints:\t%s\n", taint["key"].(string)+":"+taint["effect"].(string))
			} else {
				fmt.Fprintf(writer, " \t%s\n", taint["key"].(string)+":"+taint["effect"].(string))
			}
		}
	} else {
		fmt.Fprintf(writer, "Taints:\t<none>\n")
	}

	if unschedulable, ok := spec["unschedulable"]; ok {
		fmt.Fprintf(writer, "Unschedulable:\t%t\n", unschedulable)
	}
	fmt.Fprintf(writer, "Conditions:\n")

	writer.Flush()
	fmt.Fprintf(writer, "  Type\tStatus\tLastHeartbeatTime\tLastTransitionTime\tReason\tMessage\n")
	fmt.Fprintf(writer, "  ----\t------\t-----------------\t------------------\t------\t-------\n")
	for _, item := range conditions {
		cond := item.(map[string]interface{})
		lastHeartbeatTime, _ := time.Parse("2006-01-02T15:04:05Z", cond["lastHeartbeatTime"].(string))
		lastTransitionTime, _ := time.Parse("2006-01-02T15:04:05Z", cond["lastTransitionTime"].(string))
		fmt.Fprintf(writer, "  %s \t%s \t%s \t%s \t%s \t%s\n", cond["type"], cond["status"], lastHeartbeatTime.Format(time.RFC1123Z), lastTransitionTime.Format(time.RFC1123Z), cond["reason"], cond["message"])
	}
	writer.Flush()
	fmt.Fprintf(writer, "Addresses:\n")
	for _, item := range addresses {
		addr := item.(map[string]interface{})
		fmt.Fprintf(writer, "  %s:\t%s\n", addr["type"], addr["address"])
	}
	writer.Flush()
	fmt.Fprintf(writer, "Capacity:\n")
	capacity := status["capacity"].(map[string]interface{})
	for k, v := range capacity {
		fmt.Fprintf(writer, "  %s:\t%s\n", k, v)
	}
	fmt.Fprintf(writer, "Allocatable:\n")
	allocatable := status["allocatable"].(map[string]interface{})
	for k, v := range allocatable {
		fmt.Fprintf(writer, "  %s:\t%s\n", k, v)
	}
	fmt.Fprintf(writer, "System Info:\n")
	fmt.Fprintf(writer, "  Machine ID:\t%s\n", nodeInfo["machineID"])
	fmt.Fprintf(writer, "  System UUID:\t%s\n", nodeInfo["systemUUID"])
	fmt.Fprintf(writer, "  Boot ID:\t%s\n", nodeInfo["bootID"])
	fmt.Fprintf(writer, "  Kernel Version:\t%s\n", nodeInfo["kernelVersion"])
	fmt.Fprintf(writer, "  OS Image:\t%s\n", nodeInfo["osImage"])
	fmt.Fprintf(writer, "  Operating System:\t%s\n", nodeInfo["operatingSystem"])
	fmt.Fprintf(writer, "  Architecture:\t%s\n", nodeInfo["architecture"])
	fmt.Fprintf(writer, "  Container Runtime Version:\t%s\n", nodeInfo["containerRuntimeVersion"])
	fmt.Fprintf(writer, "  Kubelet Version:\t%s\n", nodeInfo["kubeletVersion"])
	fmt.Fprintf(writer, "  Kube-Proxy Version:\t%s\n", nodeInfo["kubeProxyVersion"])

	if podCIDR, ok := spec["podCIDR"]; ok {
		fmt.Fprintf(writer, "PodCIDR:\t%s\n", podCIDR)
	}
	if podCIDRs, ok := spec["podCIDRs"]; ok {
		podCIDRList := podCIDRs.([]string)
		fmt.Fprintf(writer, "PodCIDRs:\t%s\n", strings.Join(podCIDRList, ","))
	}
	if providerID, ok := spec["providerID"]; ok {
		fmt.Fprintf(writer, "ProviderID:\t%s\n", providerID)
	}
	writer.Flush()
}

func describePod(item interface{}) {
	pod := item.(map[string]interface{})
	metadata := pod["metadata"].(map[string]interface{})
	status := pod["status"].(map[string]interface{})
	spec := pod["spec"].(map[string]interface{})

	creationTimeStr := metadata["creationTimestamp"].(string)
	// fmt.Println("metadata: ", metadata)

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	fmt.Fprintf(writer, "Priority:\t%s\n", strconv.FormatInt(int64((spec["priority"].(float64))), 10))
	fmt.Fprintf(writer, "Service Account:\t%s\n", spec["serviceAccount"])

	if nodeName, ok := spec["nodeName"]; ok {
		fmt.Fprintf(writer, "Node:\t%s\n", nodeName)
	}

	creationTime, err := time.Parse("2006-01-02T15:04:05Z", creationTimeStr)
	if err == nil {
		fmt.Fprintf(writer, "Start Time:\t%s\n", creationTime.Format(time.RFC1123Z))
	}

	describeLabels(writer, metadata["labels"])
	describeAnnotations(writer, metadata["annotations"])

	fmt.Fprintf(writer, "Status:\t%s\n", status["phase"])

	if podIP, ok := status["podIP"]; ok {
		fmt.Fprintf(writer, "IP:\t%s\n", podIP)
	}
	if podIPs, ok := status["podIPs"]; ok {
		podIPList := podIPs.([]interface{})
		fmt.Fprintf(writer, "IPs:\n")
		for _, item1 := range podIPList {
			ip := item1.(map[string]interface{})
			fmt.Fprintf(writer, "  IP:\t%s\n", ip["ip"])
		}
	}
	fmt.Fprintf(writer, "Containers:\n")
	containerStatuses, ok := status["containerStatuses"].([]interface{})
	for _, item1 := range spec["containers"].([]interface{}) {
		cont := item1.(map[string]interface{})
		fmt.Fprintf(writer, "  %s:\n", cont["name"])
		fmt.Fprintf(writer, "    Image: %s\n", cont["image"])
		if _, ok1 := cont["command"]; ok1 {
			fmt.Fprintf(writer, "    Command:\n")
			for _, item2 := range cont["command"].([]interface{}) {
				fmt.Fprintf(writer, "      %s\n", item2)
			}
		}
		if ok {
			for _, item3 := range containerStatuses {
				// fmt.Fprintf(writer, "item3: %s\n", item3)
				cStatus := item3.(map[string]interface{})
				if cStatus["name"] == cont["name"] {
					cState := cStatus["state"].(map[string]interface{})
					// fmt.Fprintf(writer, "cState: %s\n", cState)
					if cRunning, ok2 := cState["running"]; ok2 {
						cRunningMap := cRunning.(map[string]interface{})
						fmt.Fprintf(writer, "    State:\tRunning\n")
						startedTimeStr := cRunningMap["startedAt"].(string)
						startedTime, err := time.Parse("2006-01-02T15:04:05Z", startedTimeStr)
						if err == nil {
							fmt.Fprintf(writer, "      Started:\t%s\n", startedTime.Format(time.RFC1123Z))
						}
					} else if cWaiting, ok2 := cState["waiting"]; ok2 {
						cWaitingMap := cWaiting.(map[string]interface{})
						fmt.Fprintf(writer, "    State:\tWaiting\n")
						if reasonStr, ok3 := cWaitingMap["reason"]; ok3 {
							fmt.Fprintf(writer, "      Reason:\t%s\n", reasonStr)
						}
					} else if cTerminated, ok2 := cState["terminated"]; ok2 {
						cTerminatedMap := cTerminated.(map[string]interface{})
						fmt.Fprintf(writer, "    State:\tTerminated\n")
						if reasonStr, ok3 := cTerminatedMap["reason"]; ok3 {
							fmt.Fprintf(writer, "      Reason:\t%s\n", reasonStr)
						}
						if messageStr, ok3 := cTerminatedMap["message"]; ok3 {
							fmt.Fprintf(writer, "      Message:\t%s\n", messageStr)
						}
						fmt.Fprintf(writer, "      Exit Code:\t%s\n", cTerminatedMap["exitCode"])
						if signalStr, ok3 := cTerminatedMap["signal"]; ok3 {
							fmt.Fprintf(writer, "      Signal:\t%s\n", signalStr)
						}
						startedTimeStr := cTerminatedMap["startedAt"].(string)
						startedTime, err := time.Parse("2006-01-02T15:04:05Z", startedTimeStr)
						if err == nil {
							fmt.Fprintf(writer, "      Started:\t%s\n", startedTime.Format(time.RFC1123Z))
						}
						finishedTimeStr := cTerminatedMap["startedAt"].(string)
						finishedTime, err := time.Parse("2006-01-02T15:04:05Z", finishedTimeStr)
						if err == nil {
							fmt.Fprintf(writer, "      Finished:\t%s\n", finishedTime.Format(time.RFC1123Z))
						}
					} else {
						fmt.Fprintf(writer, "    State:\tWaiting\n")
					}
				}
				fmt.Fprintf(writer, "    Ready:\t%v\n", cStatus["ready"])
				fmt.Fprintf(writer, "    Restart Count:\t%v\n", cStatus["restartCount"])
			}
		}
		resources := cont["resources"].(map[string]interface{})
		if limits, ok4 := resources["limits"].(map[string]interface{}); ok4 {
			fmt.Fprintf(writer, "    Limits:\n")
			for k, v := range limits {
				fmt.Fprintf(writer, "      %s:\t%s\n", k, v)
			}
		}
		writer.Flush()
		if reqs, ok4 := resources["requests"].(map[string]interface{}); ok4 {
			fmt.Fprintf(writer, "    Requests:\n")
			for k, v := range reqs {
				fmt.Fprintf(writer, "      %s:\t%s\n", k, v)
			}
		}
		if livenessProbe, ok4 := cont["livenessProbe"].(map[string]interface{}); ok4 {
			attrs := fmt.Sprintf("delay=%vs timeout=%vs period=%vs #success=%v #failure=%v", livenessProbe["initialDelaySeconds"], livenessProbe["timeoutSeconds"], livenessProbe["periodSeconds"], livenessProbe["successThreshold"], livenessProbe["failureThreshold"])
			if _, ok5 := livenessProbe["exec"]; ok5 {
				probe := livenessProbe["exec"].(map[string]interface{})
				fmt.Fprintf(writer, "    Liveness:\texec %v %s\n", probe["command"], attrs)
			} else if _, ok5 := livenessProbe["httpGet"]; ok5 {
				probe := livenessProbe["httpGet"].(map[string]interface{})
				url := &url.URL{}
				url.Scheme = strings.ToLower(probe["scheme"].(string))
				host := ""
				if _, ok6 := probe["host"]; ok6 {
					host = probe["host"].(string)
				}
				if _, ok6 := probe["port"]; ok6 {
					url.Host = net.JoinHostPort(host, strconv.FormatInt(int64((probe["port"].(float64))), 10))
				} else {
					url.Host = host
				}
				url.Path = probe["path"].(string)
				fmt.Fprintf(writer, "    Liveness:\thttp-get %s %s\n", url.String(), attrs)
			} else if _, ok5 := livenessProbe["tcpSocket"]; ok5 {
				probe := livenessProbe["tcpSocket"].(map[string]interface{})
				fmt.Fprintf(writer, "    Liveness:\ttcp-socket %s:%s %s\n", probe["host"], probe["port"], attrs)
			} else if _, ok5 := livenessProbe["grpc"]; ok5 {
				probe := livenessProbe["grpc"].(map[string]interface{})
				fmt.Fprintf(writer, "    Liveness:\tgrpc <pod>:%d %s %s\n", probe["port"], probe["service"], attrs)
			} else {
				fmt.Fprintf(writer, "    Liveness:\tunknown %s\n", attrs)
			}
		}
		if envs, ok4 := cont["env"].([]interface{}); ok4 {
			fmt.Fprintf(writer, "    Environment:\n")
			// envs := cont["env"].([]interface{})
			for _, env := range envs {
				envMap := env.(map[string]interface{})
				if _, ok5 := envMap["valueFrom"]; !ok5 {
					for i, s := range strings.Split(envMap["value"].(string), "\n") {
						if i == 0 {
							fmt.Fprintf(writer, "      %s:\t%s\n", envMap["name"], s)
						} else {
							fmt.Fprintf(writer, "      \t%s\n", s)
						}
					}
				} else {
					valueFrom := envMap["valueFrom"].(map[string]interface{})
					if _, ok6 := valueFrom["fieldRef"]; ok6 {
						fieldRef := valueFrom["fieldRef"].(map[string]interface{})
						fmt.Fprintf(writer, "      %s:\t (%s:%s)\n", envMap["name"], fieldRef["apiVersion"], fieldRef["fieldPath"])
					} else if _, ok6 := valueFrom["resourceFieldRef"]; ok6 {
						resourceFieldRef := valueFrom["resourceFieldRef"].(map[string]interface{})
						fmt.Fprintf(writer, "      %s:\t%s (%s)\n", envMap["name"], resourceFieldRef["containerName"], resourceFieldRef["resource"])
					} else if _, ok6 := valueFrom["secretKeyRef"]; ok6 {
						secretKeyRef := valueFrom["secretKeyRef"].(map[string]interface{})
						optional := false
						if _, ok7 := secretKeyRef["optional"]; ok7 {
							optional = secretKeyRef["optional"].(bool)
						}
						fmt.Fprintf(writer, "      %s:\t<set to the key '%s' in secret '%s'>\tOptional: %t\n", envMap["name"], secretKeyRef["key"], secretKeyRef["name"], optional)
					} else if _, ok6 := valueFrom["configMapKeyRef"]; ok6 {
						configMapKeyRef := valueFrom["configMapKeyRef"].(map[string]interface{})
						optional := false
						if _, ok7 := configMapKeyRef["optional"]; ok7 {
							optional = configMapKeyRef["optional"].(bool)
						}
						fmt.Fprintf(writer, "      %s:\t<set to the key '%s' of config map '%s'>\tOptional: %t\n", envMap["name"], configMapKeyRef["key"], configMapKeyRef["name"], optional)
					}
				}
			}
		} else {
			fmt.Fprintf(writer, "    Environment:\t<none>\n")
		}
		if volumeMounts, ok4 := cont["volumeMounts"].([]interface{}); ok4 {
			fmt.Fprintf(writer, "    Mounts:\n")
			for _, item4 := range volumeMounts {
				mount := item4.(map[string]interface{})
				flags := []string{}
				if readOnly, ok5 := mount["readOnly"].(bool); ok5 && readOnly {
					flags = append(flags, "ro")
				} else {
					flags = append(flags, "rw")
				}
				if subPath, ok5 := mount["subPath"]; ok5 {
					flags = append(flags, fmt.Sprintf("path=%q", subPath))
				}
				fmt.Fprintf(writer, "      %s from %s (%s)\n", mount["mountPath"], mount["name"], strings.Join(flags, ","))
			}
		} else {
			fmt.Fprintf(writer, "    Mounts:\t<none>\n")
		}
		if volumeDevices, ok4 := cont["volumeDevices"].([]interface{}); ok4 {
			fmt.Fprintf(writer, "    Devices:\n")
			for _, item4 := range volumeDevices {
				device := item4.(map[string]interface{})
				fmt.Fprintf(writer, "      %s from %s\n", device["devicePath"], device["name"])
			}
		}
	}

	// if ok {
	// 	for _, item1 := range containerStatuses {
	// 		cStatus := item1.(map[string]interface{})
	// 		if cStatus["ready"] == true {
	// 			ready++
	// 		}
	// 	}
	// 	if len(containerStatuses) > 0 {
	// 		firstStatus := containerStatuses[0].(map[string]interface{})
	// 		restartCount = strconv.FormatInt(int64((firstStatus["restartCount"].(float64))), 10)
	// 	}
	// 	startTimeStr := status["startTime"].(string)
	// 	age = getAge(startTimeStr)
	// }
	// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})

	if conditions, ok1 := status["conditions"].([]interface{}); ok1 {
		fmt.Fprintf(writer, "Conditions:\n")
		fmt.Fprintf(writer, "  Type\tStatus\n")
		for _, item1 := range conditions {
			cond := item1.(map[string]interface{})
			fmt.Fprintf(writer, "  %v \t%v \n", cond["type"], cond["status"])
		}

	}

	if volumes, ok1 := spec["volumes"].([]interface{}); ok1 {
		fmt.Fprintf(writer, "Volumes:\n")
		for _, item1 := range volumes {
			vol := item1.(map[string]interface{})
			fmt.Fprintf(writer, "  %v:\n", vol["name"])
			if hostPath, ok2 := vol["hostPath"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tHostPath (bare host directory volume)\n")
				fmt.Fprintf(writer, "    Path:\t%s\n", hostPath["path"])
				if hostPathType, ok3 := hostPath["type"]; ok3 {
					fmt.Fprintf(writer, "    HostPathType:\t%s\n", hostPathType)
				}
			} else if emptyDir, ok2 := vol["emptyDir"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tEmptyDir (a temporary directory that shares a pod's lifetime)\n")
				var sizeLimitStr string
				fmt.Fprintf(writer, "    Medium:")
				if medium, ok3 := emptyDir["medium"].(string); ok3 {
					fmt.Fprintf(writer, "\t%s", medium)
				}
				fmt.Fprintf(writer, "\n")
				if sizeLimit, ok3 := emptyDir["sizeLimit"].(float64); ok3 && sizeLimit > 0 {
					sizeLimitStr = fmt.Sprintf("%v", sizeLimit)
				}
				fmt.Fprintf(writer, "    SizeLimit:\t%s\n", sizeLimitStr)
			} else if secret, ok2 := vol["secret"].(map[string]interface{}); ok2 {
				optional := false
				if _, ok3 := secret["optional"]; ok3 {
					optional = secret["optional"].(bool)
				}
				fmt.Fprintf(writer, "    Type:\tSecret (a volume populated by a Secret)\n")
				fmt.Fprintf(writer, "    SecretName:\t%v\n", secret["secretName"])
				fmt.Fprintf(writer, "    Optional:\t%v\n", optional)
			} else if configMap, ok2 := vol["configMap"].(map[string]interface{}); ok2 {
				optional := false
				if _, ok3 := configMap["optional"]; ok3 {
					optional = configMap["optional"].(bool)
				}
				fmt.Fprintf(writer, "    Type:\tConfigMap (a volume populated by a ConfigMap)\n")
				fmt.Fprintf(writer, "    Name:\t%v\n", configMap["name"])
				fmt.Fprintf(writer, "    Optional:\t%v\n", optional)
			} else if nfs, ok2 := vol["nfs"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tNFS (an NFS mount that lasts the lifetime of a pod)\n")
				fmt.Fprintf(writer, "    Server:\t%v\n", nfs["server"])
				fmt.Fprintf(writer, "    Path:\t%v\n", nfs["path"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", nfs["readOnly"])
			} else if iscsi, ok2 := vol["iscsi"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tISCSI (an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod)\n")
				fmt.Fprintf(writer, "    TargetPortal:\t%v\n", iscsi["portals"])
				fmt.Fprintf(writer, "    IQN:\t%v\n", iscsi["iqn"])
				fmt.Fprintf(writer, "    Lun:\t%v\n", iscsi["lun"])
				fmt.Fprintf(writer, "    ISCSIInterface:\t%v\n", iscsi["iscsiInterface"])
				fmt.Fprintf(writer, "    FSType:\t%v\n", iscsi["fsType"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", iscsi["readOnly"])
				fmt.Fprintf(writer, "    Portals:\t%v\n", iscsi["targetPortal"])
				fmt.Fprintf(writer, "    DiscoveryCHAPAuth:\t%v\n", iscsi["chapAuthDiscovery"])
				fmt.Fprintf(writer, "    SessionCHAPAuth:\t%v\n", iscsi["chapAuthSession"])
				fmt.Fprintf(writer, "    SecretRef:\t%v\n", iscsi["server"])
				if _, ok3 := iscsi["initiatorName"]; ok3 {
					fmt.Fprintf(writer, "    InitiatorName:\t%v\n", iscsi["initiatorName"])
				}
			} else if pvc, ok2 := vol["persistentVolumeClaim"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tPersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)\n")
				fmt.Fprintf(writer, "    ClaimName:\t%v\n", pvc["claimName"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", pvc["readOnly"])
				// } else if ephemeral, ok2 := vol["ephemeral"].(map[string]interface{}); ok2 {
				// 	fmt.Fprintf(writer, "    Type:\tEphemeralVolume (an inline specification for a volume that gets created and deleted with the pod)\n")
				// 	fmt.Fprintf(writer, "    ClaimName:\t%v\n", pvc["claimName"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", pvc["readOnly"])
			} else if rbd, ok2 := vol["rbd"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tRBD (a Rados Block Device mount on the host that shares a pod's lifetime)\n")
				fmt.Fprintf(writer, "    CephMonitors:\t%v\n", rbd["monitors"])
				fmt.Fprintf(writer, "    RBDImage:\t%v\n", rbd["image"])
				fmt.Fprintf(writer, "    FSType:\t%v\n", rbd["fsType"])
				fmt.Fprintf(writer, "    RBDPool:\t%v\n", rbd["pool"])
				fmt.Fprintf(writer, "    RadosUser:\t%v\n", rbd["user"])
				fmt.Fprintf(writer, "    Keyring:\t%v\n", rbd["keyring"])
				fmt.Fprintf(writer, "    SecretRef:\t%v\n", rbd["secretRef"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", rbd["readOnly"])
			} else if vsphereVolume, ok2 := vol["vsphereVolume"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tvSphereVolume (a Persistent Disk resource in vSphere)\n")
				fmt.Fprintf(writer, "    VolumePath:\t%v\n", vsphereVolume["volumePath"])
				fmt.Fprintf(writer, "    FSType:\t%v\n", vsphereVolume["fsType"])
				fmt.Fprintf(writer, "    StoragePolicyName:\t%v\n", vsphereVolume["storagePolicyName"])
			} else if cinder, ok2 := vol["cinder"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tCinder (a Persistent Disk resource in OpenStack)\n")
				fmt.Fprintf(writer, "    VolumeID:\t%v\n", cinder["volumeID"])
				fmt.Fprintf(writer, "    FSType:\t%v\n", cinder["fsType"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cinder["readOnly"])
				fmt.Fprintf(writer, "    SecretRef:\t%v\n", cinder["secretRef"])
			} else if cephfs, ok2 := vol["cephfs"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tCephFS (a CephFS mount on the host that shares a pod's lifetime)\n")
				fmt.Fprintf(writer, "    Monitors:\t%v\n", cephfs["monitors"])
				fmt.Fprintf(writer, "    Path:\t%v\n", cephfs["path"])
				fmt.Fprintf(writer, "    User:\t%v\n", cephfs["user"])
				fmt.Fprintf(writer, "    SecretFile:\t%v\n", cephfs["secretFile"])
				fmt.Fprintf(writer, "    SecretRef:\t%v\n", cephfs["secretRef"])
				fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
			} else if projected, ok2 := vol["projected"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    Type:\tProjected (a volume that contains injected data from multiple sources)\n")
				sources := projected["sources"].([]interface{})
				for _, item2 := range sources {
					source := item2.(map[string]interface{})
					if pSecret, ok3 := source["secret"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    SecretName:\t%v\n", pSecret["name"])
						fmt.Fprintf(writer, "    SecretOptionalName:\t%v\n", pSecret["optional"])
					} else if _, ok3 := source["downwardAPI"]; ok3 {
						fmt.Fprintf(writer, "    DownwardAPI:\ttrue\n")
					} else if pConfigMap, ok3 := source["configMap"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    ConfigMapName:\t%v\n", pConfigMap["name"])
						fmt.Fprintf(writer, "    ConfigMapOptional:\t%v\n", pConfigMap["optional"])
					} else if pServiceAccountToken, ok3 := source["serviceAccountToken"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    TokenExpirationSeconds:\t%v\n", pServiceAccountToken["expirationSeconds"])
					}
				}
				// } else if csi, ok2 := vol["csi"].(map[string]interface{}); ok2 {
				// 	fmt.Fprintf(writer, "    Type:\tCSI (a Container Storage Interface (CSI) volume source)\n")
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
			} else {
				fmt.Fprintf(writer, "    <unknown>\n")
			}
		}
	} else {
		fmt.Fprintf(writer, "    Volumes:\t<none>\n")
	}
	if qos, ok1 := status["qosClass"]; ok1 {
		fmt.Fprintf(writer, "QoS Class:\t%s\n", qos)
	}
	if nodeSelector, ok1 := spec["nodeSelector"].(map[string]interface{}); ok1 {
		idx := 0
		for k, v := range nodeSelector {
			if idx == 0 {
				fmt.Fprintf(writer, "Node-Selectors:\t%s=%s\n", k, v)
			} else {
				fmt.Fprintf(writer, "  \t%s=%s\n", k, v)
			}
			idx++

		}
	}
	if tolerations, ok1 := spec["tolerations"].([]interface{}); ok1 {
		for idx, item1 := range tolerations {
			tlr := item1.(map[string]interface{})
			if idx == 0 {
				fmt.Fprintf(writer, "Tolerations:\t%s:%s", tlr["key"], tlr["effect"])
			} else {
				fmt.Fprintf(writer, "  \t%s:%s", tlr["key"], tlr["effect"])
			}
			if op, ok2 := tlr["operator"]; ok2 {
				fmt.Fprintf(writer, " op=%s", op)
			}
			if tlrSec, ok2 := tlr["tolerationSeconds"]; ok2 {
				fmt.Fprintf(writer, " for %ss", strconv.FormatInt(int64((tlrSec.(float64))), 10))
			}
			fmt.Fprintf(writer, "\n")
		}
	} else {
		fmt.Fprintf(writer, "Tolerations:\t<none>\n")
	}
	// events
	writer.Flush()

}

func describeService(item interface{}) {
	pod := item.(map[string]interface{})
	metadata := pod["metadata"].(map[string]interface{})
	status := pod["status"].(map[string]interface{})
	spec := pod["spec"].(map[string]interface{})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	if labels, ok := metadata["labels"].(map[string]interface{}); ok {
		describeLabels(writer, labels)
	} else {
		describeLabels(writer, map[string]interface{}{})
	}
	if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
		describeAnnotations(writer, annotations)
	} else {
		describeAnnotations(writer, map[string]interface{}{})
	}
	if selector, ok1 := spec["selector"].(map[string]interface{}); ok1 {
		idx := 0
		for k, v := range selector {
			if idx == 0 {
				fmt.Fprintf(writer, "Selector:\t%s=%s", k, v)
			} else {
				fmt.Fprintf(writer, ",%s=%s", k, v)
			}
			idx++
		}
		fmt.Fprintf(writer, "\n")
	}
	fmt.Fprintf(writer, "Type:\t%s\n", spec["type"])
	fmt.Fprintf(writer, "IP Family Policy:\t%s\n", spec["ipFamilyPolicy"])
	if ipFamilies, ok := spec["ipFamilies"].([]interface{}); ok {
		famList := []string{}
		for _, fam := range ipFamilies {
			famList = append(famList, fam.(string))
		}
		fmt.Fprintf(writer, "IP Families:\t%s\n", strings.Join(famList, ","))
	} else {
		fmt.Fprintf(writer, "IP Families:\t<none>\n")
	}
	fmt.Fprintf(writer, "IP:\t%s\n", spec["clusterIP"])
	if clusterIPs, ok := spec["clusterIPs"].([]interface{}); ok {
		ipList := []string{}
		for _, ip := range clusterIPs {
			ipList = append(ipList, ip.(string))
		}
		fmt.Fprintf(writer, "IPs:\t%s\n", strings.Join(ipList, ","))
	} else {
		fmt.Fprintf(writer, "IPs:\t<none>\n")
	}
	if externalIPs, ok := spec["externalIPs"].([]interface{}); ok {
		ipList := []string{}
		for _, ip := range externalIPs {
			ipList = append(ipList, ip.(string))
		}
		fmt.Fprintf(writer, "External IPs:\t%s\n", strings.Join(ipList, ","))
	}
	if loadBalancerIP, ok := spec["loadBalancerIP"]; ok {
		fmt.Fprintf(writer, "IP:\t%s\n", loadBalancerIP)
	}
	if externalName, ok := spec["externalName"]; ok {
		fmt.Fprintf(writer, "External Name:\t%s\n", externalName)
	}
	if loadBalancer, ok := status["loadBalancer"].(map[string]interface{}); ok {
		if ingress, ok1 := loadBalancer["ingress"].([]interface{}); ok1 {
			ingList := []string{}
			for _, ing := range ingress {
				ingMap := ing.(map[string]interface{})
				if _, ok2 := ingMap["ip"]; ok2 && ingMap["ip"] != "" {
					ingList = append(ingList, ingMap["ip"].(string))
				} else {
					ingList = append(ingList, ingMap["hostname"].(string))
				}

			}
			fmt.Fprintf(writer, "LoadBalancer Ingress:\t%s\n", strings.Join(ingList, ","))
		}
	}
	ports := spec["ports"].([]interface{})
	for _, port := range ports {
		portMap := port.(map[string]interface{})
		portName := portMap["name"]
		// fmt.Println(portName)
		if portName == nil || portName == "" {
			portName = "<unset>"
		}
		fmt.Fprintf(writer, "Port:\t%s\t%v/%s\n", portName, portMap["port"], portMap["protocol"])
		fmt.Fprintf(writer, "TargetPort:\t%v/%s\n", portMap["targetPort"], portMap["protocol"])
	}
	// fmt.Fprintf(writer, "Endpoints:\t%s\n", spec["type"])
	fmt.Fprintf(writer, "Session Affinity:\t%s\n", spec["sessionAffinity"])
	if externalTrafficPolicy, ok := spec["externalTrafficPolicy"]; ok {
		fmt.Fprintf(writer, "External Traffic Policy:\t%s\n", externalTrafficPolicy)
	}
	if healthCheckNodePort, ok := spec["healthCheckNodePort"]; ok {
		fmt.Fprintf(writer, "HealthCheck NodePort:\t%v\n", healthCheckNodePort)
	}
	if loadBalancerSourceRanges, ok := spec["loadBalancerSourceRanges"].([]interface{}); ok {
		srcList := []string{}
		for _, srcRange := range loadBalancerSourceRanges {
			srcList = append(srcList, srcRange.(string))
		}
		fmt.Fprintf(writer, "LoadBalancer Source Ranges:\t%s\n", strings.Join(srcList, ","))
	}
	writer.Flush()
}
func describeDeployment(item interface{}) {
}

func describeDaemonSet(item interface{}) {
}

func describeReplicaSet(item interface{}) {
}

func describeLabels(writer *tabwriter.Writer, labels interface{}) {
	labelList := []string{}
	if labels != nil {
		labelMap := labels.(map[string]interface{})
		for k, v := range labelMap {
			labelList = append(labelList, k+"="+v.(string))
		}
	}
	if len(labelList) == 0 {
		fmt.Fprintf(writer, "Labels:\t<none>\n")
	} else {
		for i, v := range labelList {
			if i == 0 {
				fmt.Fprintf(writer, "Labels:\t%s\n", v)
			} else {
				fmt.Fprintf(writer, " \t%s\n", v)
			}
		}
	}
}

func describeAnnotations(writer *tabwriter.Writer, annotations interface{}) {
	annotationList := []string{}
	if annotations != nil {
		lannotationMap := annotations.(map[string]interface{})
		for k, v := range lannotationMap {
			annotationList = append(annotationList, k+": "+v.(string))
		}
	}
	if len(annotationList) == 0 {
		fmt.Fprintf(writer, "Annotations:\t<none>\n")
	} else {
		for i, v := range annotationList {
			if i == 0 {
				fmt.Fprintf(writer, "Annotations:\t%s\n", v)
			} else {
				fmt.Fprintf(writer, " \t%s\n", v)
			}
		}

	}
}
