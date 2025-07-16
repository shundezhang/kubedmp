package cli

import (
	// "bufio"
	"fmt"
	// "log"
	"os"
	// "path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

type PickItem func(string, string, string, string) []interface{}

func prettyPrintCronJobList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tSCHEDULE\tSUSPEND\tACTIVE\tLAST SCHEDULE\tAGE\tCONTAINERS\tIMAGES\tSELECTOR")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		job := item.(map[string]interface{})
		metadata := job["metadata"].(map[string]interface{})
		spec := job["spec"].(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		jobTemplate := spec["jobTemplate"].(map[string]interface{})
		jobTemplateSpec := jobTemplate["spec"].(map[string]interface{})
		template := jobTemplateSpec["template"].(map[string]interface{})
		templateSpec := template["spec"].(map[string]interface{})
		status := job["status"].(map[string]interface{})

		creationTimeStr := metadata["creationTimestamp"].(string)

		lastScheduleTimeStr := status["lastScheduleTime"].(string)
		lastSched := getAge(lastScheduleTimeStr)

		selectorStr := "<none>"
		selectorList := []string{}
		if selector, ok := spec["selector"].(map[string]interface{}); ok {
			if matchLabels, ok1 := selector["matchLabels"].(map[string]interface{}); ok1 {
				for k, v := range matchLabels {
					selectorList = append(selectorList, k+"="+v.(string))
				}
				selectorStr = strings.Join(selectorList, ",")
			}
		}
		age := getAge(creationTimeStr)
		imageList := []string{}
		containerList := []string{}
		containers := templateSpec["containers"].([]interface{})
		for _, item1 := range containers {
			cont := item1.(map[string]interface{})
			containerList = append(containerList, cont["name"].(string))
			imageList = append(imageList, cont["image"].(string))
		}

		active := 0
		if activeList, ok := status["active"].([]interface{}); ok {
			active = len(activeList)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%t\t%d\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], spec["schedule"], spec["suspend"], active, lastSched, age, strings.Join(containerList, ","), strings.Join(imageList, ","), selectorStr)
	}
	writer.Flush()

}

func prettyPrintJobList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tCOMPLETIONS\tDURATION\tAGE\tCONTAINERS\tIMAGES\tSELECTOR")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		job := item.(map[string]interface{})
		metadata := job["metadata"].(map[string]interface{})
		spec := job["spec"].(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		template := spec["template"].(map[string]interface{})
		templateSpec := template["spec"].(map[string]interface{})
		status := job["status"].(map[string]interface{})
		succeeded := "0"
		if status["succeeded"] != nil {
			succeeded = strconv.FormatInt(int64(status["succeeded"].(float64)), 10)
		}
		creationTimeStr := metadata["creationTimestamp"].(string)
		startTimeStr := status["startTime"].(string)
		duration := getAge(startTimeStr)

		if status["completionTime"] != nil {
			duration = getDuration(startTimeStr, status["completionTime"].(string))
		}
		selectorStr := "<none>"
		selectorList := []string{}
		if selector, ok := spec["selector"].(map[string]interface{}); ok {
			if matchLabels, ok1 := selector["matchLabels"].(map[string]interface{}); ok1 {
				for k, v := range matchLabels {
					selectorList = append(selectorList, k+"="+v.(string))
				}
				selectorStr = strings.Join(selectorList, ",")
			}
		}
		age := getAge(creationTimeStr)
		imageList := []string{}
		containerList := []string{}
		containers := templateSpec["containers"].([]interface{})
		for _, item1 := range containers {
			cont := item1.(map[string]interface{})
			containerList = append(containerList, cont["name"].(string))
			imageList = append(imageList, cont["image"].(string))
		}
		completions := "0"
		if spec["completions"] != nil {
			completions = strconv.FormatInt(int64((spec["completions"].(float64))), 10)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s/%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], succeeded, completions, duration, age, strings.Join(containerList, ","), strings.Join(imageList, ","), selectorStr)
	}
	writer.Flush()

}

func prettyPrintStorageClassList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tPROVISIONER\tRECLAIMPOLICY\tVOLUMEBINDINGMODE\tALLOWVOLUMEEXPANSION\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		sc := item.(map[string]interface{})
		metadata := sc["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		allowVolumeExpansion := false
		if sc["allowVolumeExpansion"] != nil {
			allowVolumeExpansion = sc["allowVolumeExpansion"].(bool)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%t\t%s\n", metadata["name"], sc["provisioner"], sc["reclaimPolicy"], sc["volumeBindingMode"], allowVolumeExpansion, age)
	}
	writer.Flush()

}

func prettyPrintClusterRoleList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tCREATED AT")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		cr := item.(map[string]interface{})
		metadata := cr["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\n", metadata["name"], creationTimeStr)
	}
	writer.Flush()

}

func prettyPrintRoleList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tCREATED AT")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		cr := item.(map[string]interface{})
		metadata := cr["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata["namespace"], metadata["name"], creationTimeStr)
	}
	writer.Flush()

}

func prettyPrintClusterRoleBindingList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tROLE\tAGE\tUSERS\tGROUPS\tSERVICEACCOUNTS")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		crb := item.(map[string]interface{})
		metadata := crb["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		age := getAge(creationTimeStr)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		role := crb["roleRef"].(map[string]interface{})
		user := ""
		group := ""
		sa := ""
		if subjects, ok := crb["subjects"].([]interface{}); ok {
			for _, item1 := range subjects {
				sub := item1.(map[string]interface{})
				if sub["kind"] == "ServiceAccount" {
					sa += sub["namespace"].(string) + "/" + sub["name"].(string) + " "
				}
				if sub["kind"] == "Group" {
					group += sub["name"].(string) + " "
				}
				if sub["kind"] == "User" {
					user += sub["name"].(string) + " "
				}
			}
		}
		fmt.Fprintf(writer, "%s\t%s/%s\t%s\t%s\t%s\t%s\n", metadata["name"], role["kind"], role["name"], age, strings.Replace(strings.Trim(user, " "), " ", ",", -1), strings.Replace(strings.Trim(group, " "), " ", ",", -1), strings.Replace(strings.Trim(sa, " "), " ", ",", -1))
	}
	writer.Flush()

}

func prettyPrintRoleBindingList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tROLE\tAGE\tUSERS\tGROUPS\tSERVICEACCOUNTS")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		crb := item.(map[string]interface{})
		metadata := crb["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		age := getAge(creationTimeStr)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		role := crb["roleRef"].(map[string]interface{})
		user := ""
		group := ""
		sa := ""
		if subjects, ok := crb["subjects"].([]interface{}); ok {
			for _, item1 := range subjects {
				sub := item1.(map[string]interface{})
				if sub["kind"] == "ServiceAccount" {
					sa += sub["namespace"].(string) + "/" + sub["name"].(string) + " "
				}
				if sub["kind"] == "Group" {
					group += sub["name"].(string) + " "
				}
				if sub["kind"] == "User" {
					user += sub["name"].(string) + " "
				}
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%s/%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], role["kind"], role["name"], age, strings.Replace(strings.Trim(user, " "), " ", ",", -1), strings.Replace(strings.Trim(group, " "), " ", ",", -1), strings.Replace(strings.Trim(sa, " "), " ", ",", -1))
	}
	writer.Flush()

}

func prettyPrintNodeList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tSTATUS\tROLES\tAGE\tVERSION\tINTERNAL-IP\tEXTERNAL-IP\tOS-IMAGE\tKERNEL-VERSION\tCONTAINER-RUNTIME")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		node := item.(map[string]interface{})
		metadata := node["metadata"].(map[string]interface{})
		nodeName := metadata["name"].(string)
		spec := node["spec"].(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		status := node["status"].(map[string]interface{})
		addresses := status["addresses"].([]interface{})
		conditions := status["conditions"].([]interface{})
		nodeInfo := status["nodeInfo"].(map[string]interface{})
		kubletVersion := nodeInfo["kubeletVersion"].(string)
		kernelVersion := nodeInfo["kernelVersion"].(string)
		osImage := nodeInfo["osImage"].(string)
		containerRuntimeVersion := nodeInfo["containerRuntimeVersion"].(string)
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		labels := metadata["labels"].(map[string]interface{})
		role := "<none>"
		roles := []string{}
		for r := range labels {
			if strings.HasPrefix(r, "node-role.kubernetes.io/") {
				roles = append(roles, strings.Split(r, "/")[1])
			}
		}
		if len(roles) > 0 {
			role = strings.Join(roles, ",")
		}
		// var hostname string
		var ipaddress string
		extip := "<none>"
		for _, address := range addresses {
			// fmt.Println("address: ", address)
			add := address.(map[string]interface{})
			if add["type"] == "InternalIP" {
				ipaddress = add["address"].(string)
			} else if add["type"] == "ExternalIP" {
				extip = add["address"].(string)
				// } else if add["type"] == "Hostname" {
				// 	hostname = add["address"].(string)
			}
		}
		var state string
		for _, condition := range conditions {
			cond := condition.(map[string]interface{})
			if cond["status"] == "True" {
				state += cond["type"].(string) + " "
			}
		}
		if spec["unschedulable"] != nil && spec["unschedulable"].(bool) {
			state += "SchedulingDisabled "
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", nodeName, strings.Replace(strings.Trim(state, " "), " ", ",", -1), role, age, kubletVersion, ipaddress, extip, osImage, kernelVersion, containerRuntimeVersion)
	}
	writer.Flush()
}

func prettyPrintPodList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		metadata := pod["metadata"].(map[string]interface{})
		status := pod["status"].(map[string]interface{})
		spec := pod["spec"].(map[string]interface{})
		// hostIP, ok := status["hostIP"]
		// if !ok {
		// 	hostIP = " "
		// }
		nodeName, ok := spec["nodeName"]
		if !ok {
			nodeName = " "
		}
		podIP, ok := status["podIP"]
		if !ok {
			podIP = "None"
		}
		restartCount := "0"
		age := "0s"
		ready := 0
		containerStatuses, ok := status["containerStatuses"].([]interface{})
		if ok {
			for _, item1 := range containerStatuses {
				cStatus := item1.(map[string]interface{})
				if cStatus["ready"] == true {
					ready++
				}
			}
			if len(containerStatuses) > 0 {
				firstStatus := containerStatuses[0].(map[string]interface{})
				restartCount = strconv.FormatInt(int64((firstStatus["restartCount"].(float64))), 10)
			}
			startTimeStr := status["startTime"].(string)
			age = getAge(startTimeStr)
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], strconv.Itoa(ready)+"/"+strconv.Itoa(len(containerStatuses)), status["phase"], restartCount, age, podIP, nodeName)
	}
	writer.Flush()
}

func prettyPrintServiceList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tTYPE\tCLUSTER-IP\tEXTERNAL-IP\tPORT(S)\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		svc := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := svc["spec"].(map[string]interface{})
		status := svc["status"].(map[string]interface{})
		metadata := svc["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		extip := "<unknown>"
		result := sets.NewString()
		switch spec["type"] {
		case "ClusterIP":
			if spec["externalIPs"] != nil && len(spec["externalIPs"].([]interface{})) > 0 {
				extIPs := spec["externalIPs"].([]interface{})
				for _, ip := range extIPs {
					result.Insert(ip.(string))
				}
				extip = strings.Join(result.List(), ",")
			} else {
				extip = "<none>"
			}
		case "NodePort":
			if spec["externalIPs"] != nil && len(spec["externalIPs"].([]interface{})) > 0 {
				extIPs := spec["externalIPs"].([]interface{})
				for _, ip := range extIPs {
					result.Insert(ip.(string))
				}
				extip = strings.Join(result.List(), ",")
			} else {
				extip = "<none>"
			}
		case "LoadBalancer":
			if status["loadBalancer"] != nil {
				lb := status["loadBalancer"].(map[string]interface{})
				if lb["ingress"] != nil && len(lb["ingress"].([]interface{})) > 0 {
					extIPs := lb["ingress"].([]interface{})
					for _, ing := range extIPs {
						ingress := ing.(map[string]interface{})
						if ingress["ip"] != nil {
							result.Insert(ingress["ip"].(string))
						} else if ingress["hostname"] != nil {
							result.Insert(ingress["hostname"].(string))
						}
					}
				}
			}
			if spec["externalIPs"] != nil && len(spec["externalIPs"].([]interface{})) > 0 {
				extIPs := spec["externalIPs"].([]interface{})
				for _, ip := range extIPs {
					result.Insert(ip.(string))
				}
			}
			if len(result) > 0 {
				if len(result.List()) > 2 {
					extip = strings.Join(result.List()[0:2], ",") + "..."
				} else {
					extip = strings.Join(result.List(), ",")
				}
			} else {
				extip = "<pending>"
			}
		case "ExternalName":
			extip = spec["externalName"].(string)
		}
		portString := "<none>"
		if spec["ports"] != nil && len(spec["ports"].([]interface{})) > 0 {
			ports := spec["ports"].([]interface{})
			portList := []string{}
			for _, item1 := range ports {
				port := item1.(map[string]interface{})
				// fmt.Println(port)
				// fmt.Println(strconv.FormatInt(int64(port["port"].(float64)), 10))
				// fmt.Println(port["protocol"].(string))
				portList = append(portList, strconv.FormatInt(int64(port["port"].(float64)), 10)+"/"+port["protocol"].(string))
			}
			portString = strings.Join(portList, ",")
		}
		clusterIP := "<none>"
		if spec["clusterIP"] != nil {
			clusterIP = spec["clusterIP"].(string)
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], spec["type"], clusterIP, extip, portString, age)
	}
	writer.Flush()
}

func prettyPrintDeploymentList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		deploy := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := deploy["spec"].(map[string]interface{})
		metadata := deploy["metadata"].(map[string]interface{})
		status := deploy["status"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		ready := "0"
		if status["readyReplicas"] != nil {
			ready = strconv.FormatInt(int64((status["readyReplicas"].(float64))), 10)
		}
		replica := strconv.FormatInt(int64((spec["replicas"].(float64))), 10)
		update := "0"
		if status["updatedReplicas"] != nil {
			update = strconv.FormatInt(int64((status["updatedReplicas"].(float64))), 10)
		}
		avail := "0"
		if status["availableReplicas"] != nil {
			avail = strconv.FormatInt(int64((status["availableReplicas"].(float64))), 10)
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], ready+"/"+replica, update, avail, age)
	}
	writer.Flush()
}

func prettyPrintReplicaSetList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tDESIRED\tCURRENT\tREADY\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		rs := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		// spec := deploy["spec"].(map[string]interface{})
		metadata := rs["metadata"].(map[string]interface{})
		status := rs["status"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		// fmt.Println("name: ", metadata["name"])
		// fmt.Println("ready: ", status["readyReplicas"])
		replica := "0"
		ready := "0"
		avail := "0"
		if status["replicas"] != nil && status["replicas"].(float64) > 0 {
			replica = strconv.FormatInt(int64((status["replicas"].(float64))), 10)
		}
		if status["readyReplicas"] != nil && status["readyReplicas"].(float64) > 0 {
			ready = strconv.FormatInt(int64((status["readyReplicas"].(float64))), 10)
		}
		if status["availableReplicas"] != nil && status["availableReplicas"].(float64) > 0 {
			avail = strconv.FormatInt(int64((status["availableReplicas"].(float64))), 10)
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], replica, avail, ready, age)
	}
	writer.Flush()
}

func prettyPrintStatefulSetList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tREADY\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		sts := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := sts["spec"].(map[string]interface{})
		metadata := sts["metadata"].(map[string]interface{})
		status := sts["status"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		// fmt.Println("name: ", metadata["name"])
		// fmt.Println("ready: ", status["readyReplicas"])
		replica := strconv.FormatInt(int64((spec["replicas"].(float64))), 10)
		ready := "0"
		if status["readyReplicas"] != nil && status["readyReplicas"].(float64) > 0 {
			ready = strconv.FormatInt(int64((status["readyReplicas"].(float64))), 10)
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%v/%v\t%s\n", metadata["namespace"], metadata["name"], ready, replica, age)
	}
	writer.Flush()
}

func prettyPrintDaemonSetList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tDESIRED\tCURRENT\tREADY\tUP-TO-DATE\tAVAILABLE\tNODE SELECTOR\tAGE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		daemon := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := daemon["spec"].(map[string]interface{})
		template := spec["template"].(map[string]interface{})
		templateSpec := template["spec"].(map[string]interface{})
		metadata := daemon["metadata"].(map[string]interface{})
		status := daemon["status"].(map[string]interface{})
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		ready := "0"
		if status["numberReady"] != nil && status["numberReady"].(float64) > 0 {
			strconv.FormatInt(int64((status["numberReady"].(float64))), 10)
		}
		current := "0"
		if status["currentNumberScheduled"] != nil && status["currentNumberScheduled"].(float64) > 0 {
			strconv.FormatInt(int64((status["currentNumberScheduled"].(float64))), 10)
		}
		update := "0"
		if status["updatedNumberScheduled"] != nil && status["updatedNumberScheduled"].(float64) > 0 {
			strconv.FormatInt(int64((status["updatedNumberScheduled"].(float64))), 10)
		}
		avail := "0"
		if status["numberAvailable"] != nil && status["numberAvailable"].(float64) > 0 {
			avail = strconv.FormatInt(int64((status["numberAvailable"].(float64))), 10)
		}
		desire := "0"
		if status["desiredNumberScheduled"] != nil && status["desiredNumberScheduled"].(float64) > 0 {
			strconv.FormatInt(int64((status["desiredNumberScheduled"].(float64))), 10)
		}
		// fmt.Println("name: ", metadata["name"])
		nodeSelectorList := []string{}
		if _, ok := templateSpec["nodeSelector"]; ok {
			// fmt.Println("nodeSelector: ", templateSpec["nodeSelector"])
			nodeSelector := templateSpec["nodeSelector"].(map[string]interface{})
			for k, v := range nodeSelector {
				nodeSelectorList = append(nodeSelectorList, k+"="+v.(string))
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], desire, current, ready, update, avail, strings.Join(nodeSelectorList, ","), age)
	}
	writer.Flush()
}

func prettyPrintEventList(items []interface{}) {
	sort.Slice(items, func(i, j int) bool {
		// fmt.Println("items[i].(map[string]interface{}): ", items[i].(map[string]interface{}))
		// fmt.Println("items[j].(map[string]interface{}): ", items[j].(map[string]interface{}))
		return getEventTime(items[i].(map[string]interface{})) > getEventTime(items[j].(map[string]interface{}))
	})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tLAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE")
	for _, item := range items {
		event := item.(map[string]interface{})
		metadata := event["metadata"].(map[string]interface{})
		involvedObject := event["involvedObject"].(map[string]interface{})
		// source := event["source"].(map[string]interface{})

		// fmt.Println("lastTimestampStr: ", lastTimestampStr)

		age := getAge(getEventTime(event))

		message := ""
		if event["message"] != nil {
			message = strings.TrimSpace(event["message"].(string))
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s/%s\t%s\n", metadata["namespace"], age, event["type"], event["reason"], strings.ToLower(involvedObject["kind"].(string)), involvedObject["name"].(string), message)

		// fmt.Println("item: ", reflect.TypeOf(item).String())
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
	}
	writer.Flush()
}

func prettyPrintPersistentVolumeList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tCAPACITY\tACCESS MODES\tRECLAIM POLICY\tSTATUS\tCLAIM\tSTORAGECLASS\tREASON\tAGE\tVOLUMEMODE")
	for _, item := range items {
		pv := item.(map[string]interface{})
		metadata := pv["metadata"].(map[string]interface{})
		spec := pv["spec"].(map[string]interface{})
		status := pv["status"].(map[string]interface{})
		capacity := spec["capacity"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		claim := ""
		if claimRef, ok1 := spec["claimRef"].(map[string]interface{}); ok1 {
			claim = claimRef["namespace"].(string) + "/" + claimRef["name"].(string)
		}
		accessMode := ""
		for i, m := range spec["accessModes"].([]interface{}) {
			if i > 0 {
				accessMode += ","
			}
			accessMode += m.(string)
		}
		reason := ""
		if status["reason"] != nil {
			reason = status["reason"].(string)
		}
		phase := ""
		if status["phase"] != nil {
			phase = status["phase"].(string)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["name"], capacity["storage"], accessMode, spec["persistentVolumeReclaimPolicy"], phase, claim, spec["storageClassName"], reason, age, spec["volumeMode"])
	}
	writer.Flush()
}

func prettyPrintPersistentVolumeClaimList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tSTATUS\tVOLUME\tCAPACITY\tACCESS MODES\tSTORAGECLASS\tAGE\tVOLUMEMODE")
	for _, item := range items {
		pvc := item.(map[string]interface{})
		metadata := pvc["metadata"].(map[string]interface{})
		spec := pvc["spec"].(map[string]interface{})
		status := pvc["status"].(map[string]interface{})
		capacity := status["capacity"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		accessMode := ""
		for i, m := range spec["accessModes"].([]interface{}) {
			if i > 0 {
				accessMode += ","
			}
			accessMode += m.(string)
		}

		phase := ""
		if status["phase"] != nil {
			phase = status["phase"].(string)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], phase, spec["volumeName"], capacity["storage"], accessMode, spec["storageClassName"], age, spec["volumeMode"])
	}
	writer.Flush()
}

func prettyPrintSecretList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tTYPE\tDATA\tAGE")
	for _, item := range items {
		secret := item.(map[string]interface{})
		metadata := secret["metadata"].(map[string]interface{})
		dataNum := 0
		if data, ok := secret["data"].(map[string]interface{}); ok {
			dataNum = len(data)
		}

		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%v\t%v\t%s\n", metadata["namespace"], metadata["name"], secret["type"], dataNum, age)
	}
	writer.Flush()
}

func prettyPrintConfigMapList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tDATA\tAGE")
	for _, item := range items {
		cm := item.(map[string]interface{})
		metadata := cm["metadata"].(map[string]interface{})
		dataNum := 0
		if data, ok := cm["data"].(map[string]interface{}); ok {
			dataNum = len(data)
		}

		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%v\t%s\n", metadata["namespace"], metadata["name"], dataNum, age)
	}
	writer.Flush()
}

func prettyPrintServiceAccountList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tSECRETS\tAGE")
	for _, item := range items {
		sa := item.(map[string]interface{})
		metadata := sa["metadata"].(map[string]interface{})
		dataNum := 0
		if secrets, ok := sa["secrets"].([]interface{}); ok {
			dataNum = len(secrets)
		}

		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%v\t%s\n", metadata["namespace"], metadata["name"], dataNum, age)
	}
	writer.Flush()
}

func prettyPrintEndpointsList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tENDPOINTS\tAGE")
	for _, item := range items {
		ep := item.(map[string]interface{})
		metadata := ep["metadata"].(map[string]interface{})
		eps := "<none>"
		if subsets, ok := ep["subsets"].([]interface{}); ok {
			eps = ""
			for _, item1 := range subsets {
				subset := item1.(map[string]interface{})
				adds, ok1 := subset["addresses"].([]interface{})
				if !ok1 {
					adds, _ = subset["notReadyAddresses"].([]interface{})
				}
				ports := subset["ports"].([]interface{})
				for _, item2 := range adds {
					add := item2.(map[string]interface{})
					for _, item3 := range ports {
						port := item3.(map[string]interface{})
						eps += add["ip"].(string) + ":" + strconv.FormatInt(int64((port["port"].(float64))), 10) + " "
					}
				}
			}
		}

		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%v\t%s\n", metadata["namespace"], metadata["name"], strings.Replace(strings.Trim(eps, " "), " ", ",", -1), age)
	}
	writer.Flush()
}

func prettyPrintIngressList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tCLASS\tHOSTS\tADDRESS\tPORTS\tAGE")
	for _, item := range items {
		ing := item.(map[string]interface{})
		metadata := ing["metadata"].(map[string]interface{})
		spec := ing["spec"].(map[string]interface{})
		status := ing["status"].(map[string]interface{})
		host := "*"
		port := "80"
		add := ""
		if rules, ok := spec["rules"].([]interface{}); ok {
			for _, item := range rules {
				rule := item.(map[string]interface{})
				if hostStr, ok1 := rule["host"]; ok1 {
					host = hostStr.(string)
				}
			}
		}

		if loadBalancer, ok := status["loadBalancer"].(map[string]interface{}); ok {
			if ingress, ok1 := loadBalancer["ingress"].([]interface{}); ok1 {
				for _, item := range ingress {
					address := item.(map[string]interface{})
					if ip, ok2 := address["ip"].(string); ok2 {
						add = ip
					}
					if ports, ok2 := address["ports"].(string); ok2 {
						port = ports
					}
				}
			}
		}
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)

		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], spec["ingressClassName"], host, add, port, age)
	}
	writer.Flush()
}

func getEventTime(event map[string]interface{}) string {
	if series, ok := event["series"].(map[string]interface{}); ok {
		return series["lastObservedTime"].(string)
	}
	if lastTimestamp, ok := event["lastTimestamp"].(string); ok {
		return lastTimestamp
	}
	return event["eventTime"].(string)
}

func getAge(creationTimeStr string) string {
	creationTime, err := time.Parse("2006-01-02T15:04:05Z", creationTimeStr)
	// fmt.Println("creationTimeStr: ", creationTimeStr)
	age := "0s"
	if err == nil {
		ageTime := time.Now().Sub(creationTime)
		// fmt.Println("ageTime: ", ageTime)
		return getDisplayTime(ageTime)
	} else {
		fmt.Println("Error:", err)
	}
	return age
}

func getDuration(startTimeStr string, completionTimeStr string) string {
	startTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr)
	completionTime, err1 := time.Parse("2006-01-02T15:04:05Z", completionTimeStr)
	// fmt.Println("creationTimeStr: ", startTimeStr)
	// fmt.Println("completionTimeStr: ", completionTimeStr)
	duration := "0s"
	if err == nil && err1 == nil {
		durationTime := completionTime.Sub(startTime)
		return getDisplayTime(durationTime)
	} else {
		fmt.Println("Error:", err, err1)
	}
	return duration
}

func getDisplayTime(durationTime time.Duration) string {
	duration := "0s"
	// fmt.Println("durationTime: ", durationTime.Hours())
	// fmt.Println("durationTime: ", durationTime.Minutes())
	// fmt.Println("durationTime: ", durationTime.Seconds())
	if durationTime.Hours() > 24 {
		duration = strconv.FormatInt(int64(durationTime.Hours()/24), 10) + "d"
	} else if durationTime.Hours() < 24 && durationTime.Hours() > 1 {
		duration = strconv.FormatInt(int64(durationTime.Hours()), 10) + "h"
	} else if durationTime.Hours() <= 1 && durationTime.Minutes() >= 2 {
		duration = strconv.FormatInt(int64(durationTime.Minutes()), 10) + "m"
	} else if durationTime.Minutes() < 2 && durationTime.Seconds() > 0 {
		duration = strconv.FormatInt(int64(durationTime.Seconds()), 10) + "s"
	}
	return duration
}
