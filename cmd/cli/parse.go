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
)

type PickItem func(string, string, string, string) []interface{}

func prettyPrintNodeList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tSTATUS\tROLES\tAGE\tVERSION\tINTERNAL-IP\tEXTERNAL-IP\tOS-IMAGE\tKERNEL-VERSION\tCONTAINER-RUNTIME")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		node := item.(map[string]interface{})
		metadata := node["metadata"].(map[string]interface{})
		nodeName := metadata["name"].(string)

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
		for r, _ := range labels {
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
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", nodeName, strings.Trim(state, " "), role, age, kubletVersion, ipaddress, extip, osImage, kernelVersion, containerRuntimeVersion)
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
		metadata := svc["metadata"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		extip := "<none>"
		ports := spec["ports"].([]interface{})
		portList := []string{}
		for _, item1 := range ports {
			port := item1.(map[string]interface{})
			portList = append(portList, strconv.FormatInt(int64((port["port"].(float64))), 10)+"/"+port["protocol"].(string))
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], spec["type"], spec["clusterIP"], extip, strings.Join(portList, ","), age)
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
		if status["replicas"].(float64) > 0 {
			replica = strconv.FormatInt(int64((status["replicas"].(float64))), 10)
			ready = strconv.FormatInt(int64((status["readyReplicas"].(float64))), 10)
			// update := strconv.FormatInt(int64((status["updatedReplicas"].(float64))), 10)
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
		if status["readyReplicas"].(float64) > 0 {
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
		ready := strconv.FormatInt(int64((status["numberReady"].(float64))), 10)
		current := strconv.FormatInt(int64((status["currentNumberScheduled"].(float64))), 10)
		update := strconv.FormatInt(int64((status["updatedNumberScheduled"].(float64))), 10)
		avail := strconv.FormatInt(int64((status["numberAvailable"].(float64))), 10)
		desire := strconv.FormatInt(int64((status["desiredNumberScheduled"].(float64))), 10)
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
		if ageTime.Hours() > 24 {
			age = strconv.FormatInt(int64(ageTime.Hours()/24), 10) + "d"
		} else if ageTime.Hours() < 24 && ageTime.Hours() > 0 {
			age = strconv.FormatInt(int64(ageTime.Hours()), 10) + "h"
		} else if ageTime.Hours() < 0 && ageTime.Minutes() > 0 {
			age = strconv.FormatInt(int64(ageTime.Minutes()), 10) + "m"
		} else if ageTime.Minutes() < 0 && ageTime.Seconds() > 0 {
			age = strconv.FormatInt(int64(ageTime.Seconds()), 10) + "s"
		}
	} else {
		fmt.Println("Error:", err)
	}
	return age
}
