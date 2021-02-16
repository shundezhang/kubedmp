package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func prettyPrint(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	json.Unmarshal([]byte(buffer), &result)
	fmt.Println("Kind: ", result["kind"])
	fmt.Println("================================================")
	// fmt.Println("items: ", reflect.TypeOf(result["items"]).String())
	if len(result["items"].([]interface{})) == 0 {
		fmt.Println("No Items")
	} else {
		switch result["kind"] {
		case "NodeList":
			prettyPrintNodeList(result["items"].([]interface{}))
		case "PodList":
			prettyPrintPodList(result["items"].([]interface{}))
		case "ServiceList":
			prettyPrintServiceList(result["items"].([]interface{}))
		case "DeploymentList":
			prettyPrintDeploymentList(result["items"].([]interface{}))
		}
	}
	fmt.Println()
}

func prettyPrintNodeList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Hostname\tIPAddress\tStatus\tKernal Version\tOS Image\tContainer Runtime")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		node := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		status := node["status"].(map[string]interface{})
		addresses := status["addresses"].([]interface{})
		conditions := status["conditions"].([]interface{})
		nodeInfo := status["nodeInfo"].(map[string]interface{})
		kernelVersion := nodeInfo["kernelVersion"].(string)
		osImage := nodeInfo["osImage"].(string)
		containerRuntimeVersion := nodeInfo["containerRuntimeVersion"].(string)
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		var hostname string
		var ipaddress string
		for _, address := range addresses {
			// fmt.Println("address: ", address)
			add := address.(map[string]interface{})
			if add["type"] == "Hostname" {
				hostname = add["address"].(string)
			} else if add["type"] == "InternalIP" {
				ipaddress = add["address"].(string)
			}
		}
		var state string
		for _, condition := range conditions {
			cond := condition.(map[string]interface{})
			if cond["status"] == "True" {
				state += cond["type"].(string) + " "
			}
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", hostname, ipaddress, strings.Trim(state, " "), kernelVersion, osImage, containerRuntimeVersion)
	}
	writer.Flush()
}

func prettyPrintPodList(items []interface{}) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tHostIP\tPodIP\tStatus")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		status := pod["status"].(map[string]interface{})
		metadata := pod["metadata"].(map[string]interface{})
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", metadata["name"], metadata["namespace"], status["hostIP"], status["podIP"], status["phase"])
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

func main() {
	var buffer string
	var inject bool
	//scanner := bufio.NewScanner(os.Stdin)
	r := bufio.NewReaderSize(os.Stdin, 400*1024)
	// for scanner.Scan() {
	// 	line := scanner.Text()
	buf, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		line := string(buf)
		if line == "{" {
			buffer = line
			inject = true
		} else if line == "}" {
			buffer += line
			inject = false
			prettyPrint(buffer)
		} else if inject {
			buffer += line
		}
		buf, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		fmt.Println("buffer size to small")
		return
	}
	// if err := scanner.Err(); err != nil {
	// 	log.Println(err)
	// }
}
