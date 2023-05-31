package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type process func(string, string, string, string)

func prettyPrintNodeList(items []interface{}, itemName string) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tSTATUS\tROLES\tAGE\tVERSION\tINTERNAL-IP\tEXTERNAL-IP\tOS-IMAGE\tKERNEL-VERSION\tCONTAINER-RUNTIME")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		node := item.(map[string]interface{})
		metadata := node["metadata"].(map[string]interface{})
		nodeName := metadata["name"].(string)
		if itemName != "" && nodeName != itemName {
			continue
		}
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

func prettyPrintPodList(items []interface{}, namespace string, podName string) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		status := pod["status"].(map[string]interface{})
		metadata := pod["metadata"].(map[string]interface{})
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
			podIP = " "
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
			creationTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr)
			// fmt.Println("creationTimeStr: ", creationTimeStr)
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
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", metadata["namespace"], metadata["name"], strconv.Itoa(ready)+"/"+strconv.Itoa(len(containerStatuses)), status["phase"], restartCount, age, podIP, nodeName)
	}
	writer.Flush()
}

func prettyPrintServiceList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tType\tClusterIP")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		svc := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := svc["spec"].(map[string]interface{})
		metadata := svc["metadata"].(map[string]interface{})
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", metadata["name"], metadata["namespace"], spec["type"], spec["clusterIP"])
	}
	writer.Flush()
}

func prettyPrintDeploymentList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tDesired\tCurrent")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		deploy := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		spec := deploy["spec"].(map[string]interface{})
		metadata := deploy["metadata"].(map[string]interface{})
		status := deploy["status"].(map[string]interface{})
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%.0f\t%.0f\n", metadata["name"], metadata["namespace"], spec["replicas"], status["replicas"])
	}
	writer.Flush()
}

func prettyPrintDaemonSetList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tDesired\tCurrent")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		daemon := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		// spec := daemon["spec"].(map[string]interface{})
		metadata := daemon["metadata"].(map[string]interface{})
		status := daemon["status"].(map[string]interface{})
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%.0f\t%.0f\n", metadata["name"], metadata["namespace"], status["desiredNumberScheduled"], status["numberReady"])
	}
	writer.Flush()
}

func prettyPrintEventList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	for _, item := range items {
		event := item.(map[string]interface{})
		metadata := event["metadata"].(map[string]interface{})
		source := event["source"].(map[string]interface{})
		fmt.Fprintf(writer, "name:\t%s\n", metadata["name"])
		fmt.Fprintf(writer, "namespace:\t%s\n", metadata["namespace"])
		fmt.Fprintf(writer, "reason:\t%s\n", event["reason"])
		fmt.Fprintf(writer, "message:\t%s\n", event["message"])
		fmt.Fprintf(writer, "component:\t%s\n", source["component"])
		fmt.Fprintf(writer, "host:\t%s\n", source["host"])
		fmt.Fprintf(writer, "\n")
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
	}
	writer.Flush()
}

func show(dumpFilename string, fn process, queryType, namespace, objectName string) {
	var buffer string
	var inject bool

	fmt.Printf("In parse.show: parsing dump file %s\n", dumpFilename)
	//scanner := bufio.NewScanner(os.Stdin)
	// r := bufio.NewReaderSize(os.Stdin, 500*1024)

	f, err := os.Open(dumpFilename)
	if err != nil {
		log.Fatalf("Error to read [file=%v]: %v", dumpFilename, err.Error())
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "{" {
			buffer = line
			inject = true
		} else if line == "}" {
			buffer += line
			inject = false
			fn(buffer, queryType, namespace, objectName)
			buffer = ""
		} else if inject {
			buffer += line
		}
	}

	f.Close()
	// if err := scanner.Err(); err != nil {
	// 	log.Println(err)
	// }
}
