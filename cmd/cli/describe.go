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
			fmt.Printf("Please specify a type, e.g. node/no, pod, svc/service, deploy/deployment, daemonset, rs/replicaset, and an object name\n")
			return
		}
		queryType := args[0]
		objectName := args[1]

		if !contains([]string{"no", "node", "po", "pod", "svc", "service", "deploy", "deployment", "ds", "daemonset", "rs", "replicaset"}, queryType) {
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
	fmt.Fprintf(writer, "Priority:\t%s\n", strconv.FormatInt(int64(spec["priority"].(float64)), 10))
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
		describeContainerBasicInfoWithIndent(writer, cont, "  ")
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
		describeContainerResourcesWithIndent(writer, cont, "  ")
		describeContainerProbeWithIndent(writer, cont, "  ")
		describeEnvFromWithIndent(writer, cont, "  ")
		describeEnvVarsWithIndent(writer, cont, "  ")
		describeVolumeMountsWithIndent(writer, cont, "  ")
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
	describeVolumesWithIndent(writer, spec["volumes"], "")

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
	service := item.(map[string]interface{})
	metadata := service["metadata"].(map[string]interface{})
	status := service["status"].(map[string]interface{})
	spec := service["spec"].(map[string]interface{})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	describeLabels(writer, metadata["labels"])
	describeAnnotations(writer, metadata["annotations"])
	describeSelector(writer, spec["selector"], "Selector")

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
	deploy := item.(map[string]interface{})
	metadata := deploy["metadata"].(map[string]interface{})
	status := deploy["status"].(map[string]interface{})
	spec := deploy["spec"].(map[string]interface{})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	describeLabels(writer, metadata["labels"])
	describeAnnotations(writer, metadata["annotations"])
	if selector, ok := spec["selector"].(map[string]interface{}); ok {
		if _, ok1 := selector["matchLabels"]; ok1 {
			describeSelector(writer, selector["matchLabels"], "Selector")
		}
	}
	var updatedReplicas int64
	if _, ok := status["updatedReplicas"]; ok {
		updatedReplicas = (int64)(status["updatedReplicas"].(float64))
	}
	var replicas int64
	if _, ok := status["replicas"]; ok {
		replicas = (int64)(status["replicas"].(float64))
	}
	var availableReplicas int64
	if _, ok := status["availableReplicas"]; ok {
		availableReplicas = (int64)(status["availableReplicas"].(float64))
	}
	var unavailableReplicas int64
	if _, ok := status["unavailableReplicas"]; ok {
		unavailableReplicas = (int64)(status["unavailableReplicas"].(float64))
	}
	fmt.Fprintf(writer, "Replicas:\t%v desired | %v updated | %v total | %v available | %d unavailable\n", spec["replicas"], updatedReplicas, replicas, availableReplicas, unavailableReplicas)
	strategy := spec["strategy"].(map[string]interface{})
	fmt.Fprintf(writer, "StrategyType:\t%s\n", strategy["type"])
	var minReadySeconds int64
	if _, ok := spec["minReadySeconds"]; ok {
		minReadySeconds = (int64)(spec["minReadySeconds"].(float64))
	}
	fmt.Fprintf(writer, "MinReadySeconds:\t%v\n", minReadySeconds)
	if rollingUpdate, ok := strategy["rollingUpdate"].(map[string]interface{}); ok {
		fmt.Fprintf(writer, "RollingUpdateStrategy:\t%s max unavailable, %s max surge\n", rollingUpdate["maxUnavailable"], rollingUpdate["maxSurge"])
	}
	// pod template
	template := spec["template"].(map[string]interface{})
	describePodTemplate(writer, template)

	describeConditions(writer, status["conditions"])

	writer.Flush()
}

func describeDaemonSet(item interface{}) {
	daemon := item.(map[string]interface{})
	metadata := daemon["metadata"].(map[string]interface{})
	status := daemon["status"].(map[string]interface{})
	spec := daemon["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	if selector, ok := spec["selector"].(map[string]interface{}); ok {
		if _, ok1 := selector["matchLabels"]; ok1 {
			describeSelector(writer, selector["matchLabels"], "Selector")
		}
	}
	if nodeSelector, ok := templateSpec["nodeSelector"].(map[string]interface{}); ok {
		describeSelector(writer, nodeSelector, "Node-Selector")
	}
	// fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	describeLabels(writer, metadata["labels"])
	describeAnnotations(writer, metadata["annotations"])
	fmt.Fprintf(writer, "Desired Number of Nodes Scheduled: %v\n", status["desiredNumberScheduled"])
	fmt.Fprintf(writer, "Current Number of Nodes Scheduled: %v\n", status["currentNumberScheduled"])
	fmt.Fprintf(writer, "Number of Nodes Scheduled with Up-to-date Pods: %v\n", status["updatedNumberScheduled"])
	fmt.Fprintf(writer, "Number of Nodes Scheduled with Available Pods: %v\n", status["numberAvailable"])
	fmt.Fprintf(writer, "Number of Nodes Misscheduled: %v\n", status["numberMisscheduled"])
	// fmt.Fprintf(writer, "Pods Status:\t%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
	// pod template
	describePodTemplate(writer, template)
	describeVolumesWithIndent(writer, templateSpec["volumes"], "  ")
	writer.Flush()
}

func describeReplicaSet(item interface{}) {
	rs := item.(map[string]interface{})
	metadata := rs["metadata"].(map[string]interface{})
	status := rs["status"].(map[string]interface{})
	spec := rs["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintf(writer, "Name:\t%s\n", metadata["name"])
	fmt.Fprintf(writer, "Namespace:\t%s\n", metadata["namespace"])
	if selector, ok := spec["selector"].(map[string]interface{}); ok {
		if _, ok1 := selector["matchLabels"]; ok1 {
			describeSelector(writer, selector["matchLabels"], "Selector")
		}
	}
	describeLabels(writer, metadata["labels"])
	describeAnnotations(writer, metadata["annotations"])

	// if controlledBy := printController(rs); len(controlledBy) > 0 {
	// 	w.Write(LEVEL_0, "Controlled By:\t%s\n", controlledBy)
	// }
	fmt.Fprintf(writer, "Replicas:\t%v current / %v desired\n", status["replicas"], spec["replicas"])
	// fmt.Fprintf(writer, "Pods Status:\t")
	// if getPodErr != nil {
	// 	fmt.Fprintf(writer, "error in fetching pods: %s\n", getPodErr)
	// } else {
	// 	w.Write(LEVEL_0, "%d Running / %d Waiting / %d Succeeded / %d Failed\n", running, waiting, succeeded, failed)
	// }
	describePodTemplate(writer, template)
	describeVolumesWithIndent(writer, templateSpec["volumes"], "  ")
	describeConditions(writer, status["conditions"])
	writer.Flush()
}

func describeLabels(writer *tabwriter.Writer, labels interface{}) {
	describeLabelsWithIndent(writer, labels, "")
}

func describeLabelsWithIndent(writer *tabwriter.Writer, labels interface{}, indent string) {
	labelList := []string{}
	if labels != nil {
		labelMap := labels.(map[string]interface{})
		for k, v := range labelMap {
			labelList = append(labelList, k+"="+v.(string))
		}
	}
	if len(labelList) == 0 {
		fmt.Fprintf(writer, "%sLabels:\t<none>\n", indent)
	} else {
		for i, v := range labelList {
			if i == 0 {
				fmt.Fprintf(writer, "%sLabels:\t%s\n", indent, v)
			} else {
				fmt.Fprintf(writer, " %s\t%s\n", indent, v)
			}
		}
	}
}

func describeAnnotations(writer *tabwriter.Writer, annotations interface{}) {
	describeAnnotationsWithIndent(writer, annotations, "")
}

func describeAnnotationsWithIndent(writer *tabwriter.Writer, annotations interface{}, indent string) {
	annotationList := []string{}
	if annotations != nil {
		lannotationMap := annotations.(map[string]interface{})
		for k, v := range lannotationMap {
			if k != "kubectl.kubernetes.io/last-applied-configuration" {
				annotationList = append(annotationList, k+": "+v.(string))
			}
		}
	}
	if len(annotationList) == 0 {
		fmt.Fprintf(writer, "%sAnnotations:\t<none>\n", indent)
	} else {
		for i, v := range annotationList {
			if i == 0 {
				fmt.Fprintf(writer, "%sAnnotations:\t%s\n", indent, v)
			} else {
				fmt.Fprintf(writer, " %s\t%s\n", indent, v)
			}
		}

	}
}

func describeContainerBasicInfoWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	fmt.Fprintf(writer, "%s%s:\n", indent, cont["name"])
	fmt.Fprintf(writer, "  %sImage:\t%s\n", indent, cont["image"])
	portList := []string{}
	hostPortList := []string{}
	if ports, ok := cont["ports"].([]interface{}); ok {
		for _, port := range ports {
			portMap := port.(map[string]interface{})
			if containerPort, ok1 := portMap["containerPort"]; ok1 {
				portList = append(portList, fmt.Sprintf("%v/%s", containerPort, portMap["protocol"]))
			} else {
				portList = append(portList, fmt.Sprintf("0/%s", portMap["protocol"]))
			}
			if hostPort, ok1 := portMap["hostPort"]; ok1 {
				hostPortList = append(hostPortList, fmt.Sprintf("%v/%s", hostPort, portMap["protocol"]))
			} else {
				hostPortList = append(hostPortList, fmt.Sprintf("0/%s", portMap["protocol"]))
			}
		}
		fmt.Fprintf(writer, "  %sPort:\t%s\n", indent, strings.Join(portList, ", "))
		fmt.Fprintf(writer, "  %sHost Port:\t%s\n", indent, strings.Join(hostPortList, ", "))
	}
}

func describeContainerCommandWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	if command, ok := cont["command"].([]interface{}); ok {
		fmt.Fprintf(writer, "  %sCommand:\n", indent)
		for _, item2 := range command {
			fmt.Fprintf(writer, "    %s%s\n", indent, item2)
		}
	}
	if args, ok := cont["args"].([]interface{}); ok {
		fmt.Fprintf(writer, "  %sArgs:\n", indent)
		for _, item2 := range args {
			fmt.Fprintf(writer, "    %s%s\n", indent, item2)
		}
	}
}

func describeContainerResourcesWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	resources := cont["resources"].(map[string]interface{})
	if limits, ok4 := resources["limits"].(map[string]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sLimits:\n", indent)
		for k, v := range limits {
			fmt.Fprintf(writer, "    %s%s:\t%s\n", indent, k, v)
		}
	}
	if reqs, ok4 := resources["requests"].(map[string]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sRequests:\n", indent)
		for k, v := range reqs {
			fmt.Fprintf(writer, "    %s%s:\t%s\n", indent, k, v)
		}
	}
}

func describeContainerProbeWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	if livenessProbe, ok4 := cont["livenessProbe"].(map[string]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sLiveness:\t%s", indent, describeProbe(livenessProbe))
	}
	if readinessProbe, ok4 := cont["readinessProbe"].(map[string]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sReadiness:\t%s", indent, describeProbe(readinessProbe))
	}
	if startupProbe, ok4 := cont["startupProbe"].(map[string]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sStartup:\t%s", indent, describeProbe(startupProbe))
	}
}

func describeProbe(probe map[string]interface{}) string {
	attrs := fmt.Sprintf("delay=%vs timeout=%vs period=%vs #success=%v #failure=%v", probe["initialDelaySeconds"], probe["timeoutSeconds"], probe["periodSeconds"], probe["successThreshold"], probe["failureThreshold"])
	if _, ok5 := probe["exec"]; ok5 {
		probe := probe["exec"].(map[string]interface{})
		return fmt.Sprintf("exec %v %s\n", probe["command"], attrs)
	} else if _, ok5 := probe["httpGet"]; ok5 {
		probe := probe["httpGet"].(map[string]interface{})
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
		return fmt.Sprintf("http-get %s %s\n", url.String(), attrs)
	} else if _, ok5 := probe["tcpSocket"]; ok5 {
		probe := probe["tcpSocket"].(map[string]interface{})
		return fmt.Sprintf("tcp-socket %s:%s %s\n", probe["host"], probe["port"], attrs)
	} else if _, ok5 := probe["grpc"]; ok5 {
		probe := probe["grpc"].(map[string]interface{})
		return fmt.Sprintf("grpc <pod>:%d %s %s\n", probe["port"], probe["service"], attrs)
	} else {
		return fmt.Sprintf("unknown %s\n", attrs)
	}

}
func describeEnvFromWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	if envFroms, ok := cont["envFrom"].([]interface{}); ok {
		if len(envFroms) > 0 {
			fmt.Fprintf(writer, "  %sEnvironment Variables from:\n", indent)
			for _, envFrom := range envFroms {
				envFromMap := envFrom.(map[string]interface{})
				from := ""
				name := ""
				optional := false
				if configMapRef, ok1 := envFromMap["ConfigMapRef"].(map[string]interface{}); ok1 {
					from = "ConfigMap"
					name = configMapRef["name"].(string)
					optional = configMapRef["optional"].(bool)
				} else if secretRef, ok1 := envFromMap["secretRef"].(map[string]interface{}); ok1 {
					from = "Secret"
					name = secretRef["name"].(string)
					optional = secretRef["optional"].(bool)
				}
				if prefix, ok1 := envFromMap["prefix"].(string); ok1 && len(prefix) > 0 {
					fmt.Fprintf(writer, "  %s%s\t%s with prefix '%s'\tOptional: %t\n", indent, name, from, prefix, optional)
				} else {
					fmt.Fprintf(writer, "  %s%s\t%s\tOptional: %t\n", indent, name, from, optional)
				}
			}
		} else {
			fmt.Fprintf(writer, "  %sEnvironment Variables from:\t<none>\n", indent)
		}
	}
}
func describeEnvVarsWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	if envs, ok4 := cont["env"].([]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sEnvironment:\n", indent)
		for _, env := range envs {
			envMap := env.(map[string]interface{})
			if valueFrom, ok5 := envMap["valueFrom"].(map[string]interface{}); ok5 {
				if fieldRef, ok6 := valueFrom["fieldRef"].(map[string]interface{}); ok6 {
					fmt.Fprintf(writer, "    %s%s:\t (%s:%s)\n", indent, envMap["name"], fieldRef["apiVersion"], fieldRef["fieldPath"])
				} else if resourceFieldRef, ok6 := valueFrom["resourceFieldRef"].(map[string]interface{}); ok6 {
					fmt.Fprintf(writer, "    %s%s:\t%s (%s)\n", indent, envMap["name"], resourceFieldRef["containerName"], resourceFieldRef["resource"])
				} else if secretKeyRef, ok6 := valueFrom["secretKeyRef"].(map[string]interface{}); ok6 {
					optional := false
					if _, ok7 := secretKeyRef["optional"]; ok7 {
						optional = secretKeyRef["optional"].(bool)
					}
					fmt.Fprintf(writer, "    %s%s:\t<set to the key '%s' in secret '%s'>\tOptional: %t\n", indent, envMap["name"], secretKeyRef["key"], secretKeyRef["name"], optional)
				} else if configMapKeyRef, ok6 := valueFrom["configMapKeyRef"].(map[string]interface{}); ok6 {
					optional := false
					if _, ok7 := configMapKeyRef["optional"]; ok7 {
						optional = configMapKeyRef["optional"].(bool)
					}
					fmt.Fprintf(writer, "    %s%s:\t<set to the key '%s' of config map '%s'>\tOptional: %t\n", indent, envMap["name"], configMapKeyRef["key"], configMapKeyRef["name"], optional)
				}
			} else {
				for i, s := range strings.Split(envMap["value"].(string), "\n") {
					if i == 0 {
						fmt.Fprintf(writer, "    %s%s:\t%s\n", indent, envMap["name"], s)
					} else {
						fmt.Fprintf(writer, "    %s\t%s\n", indent, s)
					}
				}
			}
		}
	} else {
		fmt.Fprintf(writer, "  %sEnvironment:\t<none>\n", indent)
	}
}

func describeVolumeMountsWithIndent(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	if volumeMounts, ok4 := cont["volumeMounts"].([]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sMounts:\n", indent)
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
			fmt.Fprintf(writer, "    %s%s from %s (%s)\n", indent, mount["mountPath"], mount["name"], strings.Join(flags, ","))
		}
	} else {
		fmt.Fprintf(writer, "  %sMounts:\t<none>\n", indent)
	}
	if volumeDevices, ok4 := cont["volumeDevices"].([]interface{}); ok4 {
		fmt.Fprintf(writer, "  %sDevices:\n", indent)
		for _, item4 := range volumeDevices {
			device := item4.(map[string]interface{})
			fmt.Fprintf(writer, "    %s%s from %s\n", indent, device["devicePath"], device["name"])
		}
	}
}

func describeVolumesWithIndent(writer *tabwriter.Writer, items interface{}, indent string) {
	if volumes, ok1 := items.([]interface{}); ok1 {
		fmt.Fprintf(writer, "%sVolumes:\n", indent)
		for _, item1 := range volumes {
			vol := item1.(map[string]interface{})
			fmt.Fprintf(writer, "  %s%v:\n", vol["name"], indent)
			if hostPath, ok2 := vol["hostPath"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tHostPath (bare host directory volume)\n", indent)
				fmt.Fprintf(writer, "    %sPath:\t%s\n", indent, hostPath["path"])
				if hostPathType, ok3 := hostPath["type"]; ok3 {
					fmt.Fprintf(writer, "    %sHostPathType:\t%s\n", indent, hostPathType)
				}
			} else if emptyDir, ok2 := vol["emptyDir"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tEmptyDir (a temporary directory that shares a pod's lifetime)\n", indent)
				var sizeLimitStr string
				fmt.Fprintf(writer, "    %sMedium:", indent)
				if medium, ok3 := emptyDir["medium"].(string); ok3 {
					fmt.Fprintf(writer, "\t%s", medium)
				}
				fmt.Fprintf(writer, "\n")
				if sizeLimit, ok3 := emptyDir["sizeLimit"].(float64); ok3 && sizeLimit > 0 {
					sizeLimitStr = fmt.Sprintf("%v", sizeLimit)
				}
				fmt.Fprintf(writer, "    %sSizeLimit:\t%s\n", indent, sizeLimitStr)
			} else if secret, ok2 := vol["secret"].(map[string]interface{}); ok2 {
				optional := false
				if _, ok3 := secret["optional"]; ok3 {
					optional = secret["optional"].(bool)
				}
				fmt.Fprintf(writer, "    %sType:\tSecret (a volume populated by a Secret)\n", indent)
				fmt.Fprintf(writer, "    %sSecretName:\t%v\n", indent, secret["secretName"])
				fmt.Fprintf(writer, "    %sOptional:\t%v\n", indent, optional)
			} else if configMap, ok2 := vol["configMap"].(map[string]interface{}); ok2 {
				optional := false
				if _, ok3 := configMap["optional"]; ok3 {
					optional = configMap["optional"].(bool)
				}
				fmt.Fprintf(writer, "    %sType:\tConfigMap (a volume populated by a ConfigMap)\n", indent)
				fmt.Fprintf(writer, "    %sName:\t%v\n", indent, configMap["name"])
				fmt.Fprintf(writer, "    %sOptional:\t%v\n", indent, optional)
			} else if nfs, ok2 := vol["nfs"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tNFS (an NFS mount that lasts the lifetime of a pod)\n", indent)
				fmt.Fprintf(writer, "    %sServer:\t%v\n", indent, nfs["server"])
				fmt.Fprintf(writer, "    %sPath:\t%v\n", indent, nfs["path"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, nfs["readOnly"])
			} else if iscsi, ok2 := vol["iscsi"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tISCSI (an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod)\n", indent)
				fmt.Fprintf(writer, "    %sTargetPortal:\t%v\n", indent, iscsi["portals"])
				fmt.Fprintf(writer, "    %sIQN:\t%v\n", indent, iscsi["iqn"])
				fmt.Fprintf(writer, "    %sLun:\t%v\n", indent, iscsi["lun"])
				fmt.Fprintf(writer, "    %sISCSIInterface:\t%v\n", indent, iscsi["iscsiInterface"])
				fmt.Fprintf(writer, "    %sFSType:\t%v\n", indent, iscsi["fsType"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, iscsi["readOnly"])
				fmt.Fprintf(writer, "    %sPortals:\t%v\n", indent, iscsi["targetPortal"])
				fmt.Fprintf(writer, "    %sDiscoveryCHAPAuth:\t%v\n", indent, iscsi["chapAuthDiscovery"])
				fmt.Fprintf(writer, "    %sSessionCHAPAuth:\t%v\n", indent, iscsi["chapAuthSession"])
				fmt.Fprintf(writer, "    %sSecretRef:\t%v\n", indent, iscsi["server"])
				if _, ok3 := iscsi["initiatorName"]; ok3 {
					fmt.Fprintf(writer, "    %sInitiatorName:\t%v\n", indent, iscsi["initiatorName"])
				}
			} else if pvc, ok2 := vol["persistentVolumeClaim"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tPersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)\n", indent)
				fmt.Fprintf(writer, "    %sClaimName:\t%v\n", indent, pvc["claimName"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, pvc["readOnly"])
				// } else if ephemeral, ok2 := vol["ephemeral"].(map[string]interface{}); ok2 {
				// 	fmt.Fprintf(writer, "    Type:\tEphemeralVolume (an inline specification for a volume that gets created and deleted with the pod)\n")
				// 	fmt.Fprintf(writer, "    ClaimName:\t%v\n", pvc["claimName"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", pvc["readOnly"])
			} else if rbd, ok2 := vol["rbd"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tRBD (a Rados Block Device mount on the host that shares a pod's lifetime)\n", indent)
				fmt.Fprintf(writer, "    %sCephMonitors:\t%v\n", indent, rbd["monitors"])
				fmt.Fprintf(writer, "    %sRBDImage:\t%v\n", indent, rbd["image"])
				fmt.Fprintf(writer, "    %sFSType:\t%v\n", indent, rbd["fsType"])
				fmt.Fprintf(writer, "    %sRBDPool:\t%v\n", indent, rbd["pool"])
				fmt.Fprintf(writer, "    %sRadosUser:\t%v\n", indent, rbd["user"])
				fmt.Fprintf(writer, "    %sKeyring:\t%v\n", indent, rbd["keyring"])
				fmt.Fprintf(writer, "    %sSecretRef:\t%v\n", indent, rbd["secretRef"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, rbd["readOnly"])
			} else if vsphereVolume, ok2 := vol["vsphereVolume"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tvSphereVolume (a Persistent Disk resource in vSphere)\n", indent)
				fmt.Fprintf(writer, "    %sVolumePath:\t%v\n", indent, vsphereVolume["volumePath"])
				fmt.Fprintf(writer, "    %sFSType:\t%v\n", indent, vsphereVolume["fsType"])
				fmt.Fprintf(writer, "    %sStoragePolicyName:\t%v\n", indent, vsphereVolume["storagePolicyName"])
			} else if cinder, ok2 := vol["cinder"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tCinder (a Persistent Disk resource in OpenStack)\n", indent)
				fmt.Fprintf(writer, "    %sVolumeID:\t%v\n", indent, cinder["volumeID"])
				fmt.Fprintf(writer, "    %sFSType:\t%v\n", indent, cinder["fsType"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, cinder["readOnly"])
				fmt.Fprintf(writer, "    %sSecretRef:\t%v\n", indent, cinder["secretRef"])
			} else if cephfs, ok2 := vol["cephfs"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tCephFS (a CephFS mount on the host that shares a pod's lifetime)\n", indent)
				fmt.Fprintf(writer, "    %sMonitors:\t%v\n", indent, cephfs["monitors"])
				fmt.Fprintf(writer, "    %sPath:\t%v\n", indent, cephfs["path"])
				fmt.Fprintf(writer, "    %sUser:\t%v\n", indent, cephfs["user"])
				fmt.Fprintf(writer, "    %sSecretFile:\t%v\n", indent, cephfs["secretFile"])
				fmt.Fprintf(writer, "    %sSecretRef:\t%v\n", indent, cephfs["secretRef"])
				fmt.Fprintf(writer, "    %sReadOnly:\t%v\n", indent, cephfs["readOnly"])
			} else if projected, ok2 := vol["projected"].(map[string]interface{}); ok2 {
				fmt.Fprintf(writer, "    %sType:\tProjected (a volume that contains injected data from multiple sources)\n", indent)
				sources := projected["sources"].([]interface{})
				for _, item2 := range sources {
					source := item2.(map[string]interface{})
					if pSecret, ok3 := source["secret"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    %sSecretName:\t%v\n", indent, pSecret["name"])
						fmt.Fprintf(writer, "    %sSecretOptionalName:\t%v\n", indent, pSecret["optional"])
					} else if _, ok3 := source["downwardAPI"]; ok3 {
						fmt.Fprintf(writer, "    %sDownwardAPI:\ttrue\n", indent)
					} else if pConfigMap, ok3 := source["configMap"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    %sConfigMapName:\t%v\n", indent, pConfigMap["name"])
						fmt.Fprintf(writer, "    %sConfigMapOptional:\t%v\n", indent, pConfigMap["optional"])
					} else if pServiceAccountToken, ok3 := source["serviceAccountToken"].(map[string]interface{}); ok3 {
						fmt.Fprintf(writer, "    %sTokenExpirationSeconds:\t%v\n", indent, pServiceAccountToken["expirationSeconds"])
					}
				}
				// } else if csi, ok2 := vol["csi"].(map[string]interface{}); ok2 {
				// 	fmt.Fprintf(writer, "    Type:\tCSI (a Container Storage Interface (CSI) volume source)\n")
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
				// 	fmt.Fprintf(writer, "    ReadOnly:\t%v\n", cephfs["readOnly"])
			} else {
				fmt.Fprintf(writer, "    %s<unknown>\n", indent)
			}
		}
	} else {
		fmt.Fprintf(writer, "%sVolumes:\t<none>\n", indent)
	}
}

func describeSelector(writer *tabwriter.Writer, selectors interface{}, displayName string) {
	if selectors != nil {
		selectorMap := selectors.(map[string]interface{})
		idx := 0
		for k, v := range selectorMap {
			if idx == 0 {
				fmt.Fprintf(writer, "%s:\t%s=%s", displayName, k, v)
			} else {
				fmt.Fprintf(writer, ",%s=%s", k, v)
			}
			idx++
		}
		fmt.Fprintf(writer, "\n")
	}
}

func describePodTemplate(writer *tabwriter.Writer, template map[string]interface{}) {
	templateMetadata := template["metadata"].(map[string]interface{})
	fmt.Fprintf(writer, "Pod Template:\n")
	describeLabelsWithIndent(writer, templateMetadata["labels"], "  ")
	describeAnnotationsWithIndent(writer, templateMetadata["annotations"], "  ")
	templateSpec := template["spec"].(map[string]interface{})
	if serviceAccount, ok := templateSpec["serviceAccount"]; ok {
		fmt.Fprintf(writer, "  Service Account:\t%s\n", serviceAccount)
	}
	if initContainers, ok := templateSpec["initContainers"].([]interface{}); ok {
		fmt.Fprintf(writer, "  Init Containers:\n")
		for _, cont := range initContainers {
			describeContainer(writer, cont.(map[string]interface{}), "    ")
		}
	}
	fmt.Fprintf(writer, "  Containers:\n")
	for _, cont := range templateSpec["containers"].([]interface{}) {
		describeContainer(writer, cont.(map[string]interface{}), "    ")
	}
	// describeTopologySpreadConstraints(template.Spec.TopologySpreadConstraints, w, "  ")
	if priorityClassName, ok := templateSpec["priorityClassName"]; ok {
		fmt.Fprintf(writer, "  Priority Class Name:\t%s\n", priorityClassName)
	}
}

func describeContainer(writer *tabwriter.Writer, cont map[string]interface{}, indent string) {
	describeContainerBasicInfoWithIndent(writer, cont, indent)
	describeContainerCommandWithIndent(writer, cont, indent)
	describeContainerResourcesWithIndent(writer, cont, indent)
	describeContainerProbeWithIndent(writer, cont, indent)
	describeEnvFromWithIndent(writer, cont, indent)
	describeEnvVarsWithIndent(writer, cont, indent)
	describeVolumeMountsWithIndent(writer, cont, indent)
}

func describeConditions(writer *tabwriter.Writer, items interface{}) {
	if conditions, ok := items.([]interface{}); ok {
		fmt.Fprintf(writer, "Conditions:\n  Type\tStatus\tReason\n")
		fmt.Fprintf(writer, "  ----\t------\t------\n")
		for _, cond := range conditions {
			condMap := cond.(map[string]interface{})
			fmt.Fprintf(writer, "  %v \t%v\t%v\n", condMap["type"], condMap["status"], condMap["reason"])
		}
	}
}