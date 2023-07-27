package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	cont = "container"
)

var logsCmd = &cobra.Command{
	Use:                   "logs POD_NAME [-n NAMESPACE] [-c CONTAINER_NAME]",
	DisableFlagsInUseLine: true,
	Short:                 "Print the logs for a container in a pod",
	Long: `Print the logs for a container in a pod or specified resource.
If the pod has more than one container, and a container name is not specified, logs of all containers will be printed out.`,
	Example: `  # Return logs from pod nginx with all containers
  kubedmp logs nginx
  
  # Return logs of ruby container logs from pod web-1
  kubectl logs web-1 -c ruby`,
	Run: func(cmd *cobra.Command, args []string) {
		dumpFile, err := cmd.Flags().GetString(dumpFile)
		if err != nil {
			log.Fatalf("Please provide a dump file\n")
			return
		}

		if len(args) == 0 {
			log.Fatalf("Please provide a pod name\n")
			return
		}
		podName := args[0]
		namespace, err := cmd.Flags().GetString(ns)
		if err != nil {
			log.Fatalf("Error parsing namespace\n")
			return
		}
		container, err := cmd.Flags().GetString(cont)
		if err != nil {
			log.Fatalf("Error parsing container\n")
			return
		} // fmt.Printf("parsing dump file %s\n", dumpFile)
		f, err := os.Open(dumpFile)
		if err != nil {
			log.Fatalf("Error to read [file=%v]: %v", dumpFile, err.Error())
		}

		finishedCh := make(chan bool, 1)
		buff := make(chan string, 100)
		marker := namespace + "/" + podName
		if len(container) > 0 {
			marker = "container " + container + " of pod " + namespace + "/" + podName
		}
		go scanFile(f, marker, buff, finishedCh)
		defer func() {
			f.Close()
		}()
		for {
			lastLine, ok := <-buff

			if ok == false {
				break
			}
			fmt.Println(lastLine)
		}

	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringP(ns, "n", "default", "namespace of the pod")
	logsCmd.Flags().StringP(cont, "c", "", "container")
}

func scanFile(f *os.File, marker string, buff chan string, finishedCh chan bool) {
	scanner := bufio.NewScanner(f)
	canPrint := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "==== START logs for container") && strings.Contains(line, marker) {
			canPrint = true
		}
		if canPrint {
			buff <- line
			// fmt.Println(line)
		}
		if strings.HasPrefix(line, "==== END logs for container") && strings.Contains(line, marker) {
			canPrint = false
		}
	}
	close(buff)
}
