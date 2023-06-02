package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
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
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		metadata := pod["metadata"].(map[string]interface{})
		// fmt.Printf("object ns %s pod %s \n", metadata["namespace"], metadata["name"])
		if (queryType == "no" || queryType == "node") && objectName == metadata["name"] {
			describeNode(item)
		} else if namespace == metadata["namespace"] && objectName == metadata["name"] {
			fmt.Println("item: ", item)
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
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tSTATUS\tROLES\tAGE\tVERSION\tINTERNAL-IP\tEXTERNAL-IP\tOS-IMAGE\tKERNEL-VERSION\tCONTAINER-RUNTIME")
	fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", nodeName, strings.Trim(state, " "), role, age, kubletVersion, ipaddress, extip, osImage, kernelVersion, containerRuntimeVersion)

	writer.Flush()
}
