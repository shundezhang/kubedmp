package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
		// spec := deploy["spec"].(map[string]interface{})
		metadata := deploy["metadata"].(map[string]interface{})
		status := deploy["status"].(map[string]interface{})
		creationTimeStr := metadata["creationTimestamp"].(string)
		// fmt.Println("creationTimeStr: ", creationTimeStr)
		age := getAge(creationTimeStr)
		ready := strconv.FormatInt(int64((status["readyReplicas"].(float64))), 10)
		replica := strconv.FormatInt(int64((status["replicas"].(float64))), 10)
		update := strconv.FormatInt(int64((status["updatedReplicas"].(float64))), 10)
		avail := strconv.FormatInt(int64((status["availableReplicas"].(float64))), 10)
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
		deploy := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		// spec := deploy["spec"].(map[string]interface{})
		metadata := deploy["metadata"].(map[string]interface{})
		status := deploy["status"].(map[string]interface{})
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
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["name"], metadata["namespace"], desire, current, ready, update, avail, strings.Join(nodeSelectorList, ","), age)
	}
	writer.Flush()
}

func prettyPrintEventList(items []interface{}) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].(map[string]interface{})["lastTimestamp"].(string) > items[j].(map[string]interface{})["lastTimestamp"].(string)
	})
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tLAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE")
	for _, item := range items {
		event := item.(map[string]interface{})
		metadata := event["metadata"].(map[string]interface{})
		involvedObject := event["involvedObject"].(map[string]interface{})
		// source := event["source"].(map[string]interface{})
		lastTimestampStr := event["lastTimestamp"].(string)
		// fmt.Println("lastTimestampStr: ", lastTimestampStr)
		age := getAge(lastTimestampStr)
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], age, event["type"], event["reason"], strings.ToLower(involvedObject["kind"].(string))+"/"+involvedObject["name"].(string), event["message"])

		// fmt.Println("item: ", reflect.TypeOf(item).String())
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
	}
	writer.Flush()
}

func show(dumpFilename string, pi PickItem, queryType, namespace, objectName string) {
	var buffer string
	var inject bool

	// fmt.Printf("In parse.show: parsing dump file %s\n", dumpFilename)
	//scanner := bufio.NewScanner(os.Stdin)
	// r := bufio.NewReaderSize(os.Stdin, 500*1024)

	f, err := os.Open(dumpFilename)
	if err != nil {
		log.Fatalf("Error to read [file=%v]: %v", dumpFilename, err.Error())
	}

	scanner := bufio.NewScanner(f)
	items := []interface{}{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "{" {
			buffer = line
			inject = true
		} else if line == "}" {
			buffer += line
			inject = false
			for _, item := range pi(buffer, queryType, namespace, objectName) {
				items = append(items, item)
			}
			buffer = ""
		} else if inject {
			buffer += line
		}
	}

	f.Close()
	// fmt.Println(items)
	switch queryType {
	case "no", "node":
		prettyPrintNodeList(items)
	case "po", "pod":
		prettyPrintPodList(items)
	case "svc", "service":
		prettyPrintServiceList(items)
	case "deploy", "deployment":
		prettyPrintDeploymentList(items)
	case "ds", "daemonset":
		prettyPrintDaemonSetList(items)
	case "rs", "replicaset":
		prettyPrintReplicaSetList(items)
	case "event":
		prettyPrintEventList(items)
	}

	// if err := scanner.Err(); err != nil {
	// 	log.Println(err)
	// }
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
