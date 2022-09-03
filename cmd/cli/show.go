package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func prettyPrint(buffer string) {
	var result map[string]interface{}
	// fmt.Println(buffer)
	err := json.Unmarshal([]byte(buffer), &result)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(buffer)
		return
	}
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
		case "DaemonSetList":
			prettyPrintDaemonSetList(result["items"].([]interface{}))
		case "EventList":
			prettyPrintEventList(result["items"].([]interface{}))
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
	fmt.Fprintln(writer, "Name\tNamespace\tHostIP\tPodIP\tStatus\tReason")
	for _, item := range items {
		// fmt.Println("item: ", reflect.TypeOf(item).String())
		pod := item.(map[string]interface{})
		// fmt.Println("item: ", reflect.TypeOf(node["status"]).String())
		status := pod["status"].(map[string]interface{})
		metadata := pod["metadata"].(map[string]interface{})
		hostIP, ok := status["hostIP"]
		if !ok {
			hostIP = " "
		}
		podIP, ok := status["podIP"]
		if !ok {
			podIP = " "
		}
		reason, ok := status["reason"]
		var message interface{}
		if !ok {
			reason = " "
		} else {
			message, ok = status["message"]
			if !ok {
				message = " "
			}
		}
		// address := item.(map[string]interface{})["status"]["addresses"].(map[string]interface{})
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s-%s\n", metadata["name"], metadata["namespace"], hostIP, podIP, status["phase"], reason, message)
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

func show(dumpFilename string) {
	var buffer string
	var inject bool

	fmt.Printf("parsing dump file %s\n", dumpFilename)
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
			prettyPrint(buffer)
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

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show details in cluster info dump",
	Long:  `show details in cluster info dump`,
	Run: func(cmd *cobra.Command, args []string) {
		dumpFile, err := cmd.Flags().GetString(dumpFile)
		if err != nil {
			log.Fatalf("Please provide a dump file\n")
			return
		}
		show(dumpFile)
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
