package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		// dumpFile, err := cmd.Flags().GetString(dumpFileFlag)
		// if err != nil {
		// 	log.Fatalf("Please provide a dump file\n")
		// 	return
		// }

		if len(args) == 0 {
			log.Fatalf("Please provide a pod name.\n")
			return
		}
		podName := args[0]

		// container, err := cmd.Flags().GetString(cont)
		// if err != nil {
		// 	log.Fatalf("Error parsing container\n")
		// 	return
		// }
		// fmt.Printf("parsing dump file %s\n", dumpFile)
		logFile := ""
		marker := resNamespace + "/" + podName
		if len(resContainer) > 0 {
			marker = "container " + resContainer + " of pod " + resNamespace + "/" + podName
		}

		if len(dumpDir) > 0 {
			dumpDirPath, _ := filepath.Abs(dumpDir)
			// fmt.Println("fullPath: ", dumpDirPath)
			if strings.Contains(dumpDirPath, "sos_commands") && strings.Contains(dumpDirPath, "kubernetes") {
				marker = ""
				podFiles, err1 := os.ReadDir(filepath.Join(dumpDir, "pods"))
				if err1 != nil {
					log.Fatalf("Error to open [dir=%v]: %v", dumpDir, err1.Error())
				}
				for _, podFile := range podFiles {
					// fmt.Println("podFile: ", podFile.Name())
					if podFile.IsDir() {
						continue
					}
					// fmt.Println("search string: ", "_--namespace_"+resNamespace+"_logs_"+podName+"_-c_")
					// fmt.Println("in name: ", strings.Contains(podFile.Name(), "_--namespace_"+resNamespace+"_logs_"+podName+"_-c_"))
					if len(resContainer) > 0 && strings.HasSuffix(podFile.Name(), "_--namespace_"+resNamespace+"_logs_"+podName+"_-c_"+resContainer) {
						logFile = filepath.Join(dumpDir, "pods", podFile.Name())
						break
					} else if len(resContainer) == 0 && strings.Contains(podFile.Name(), "_--namespace_"+resNamespace+"_logs_"+podName+"_-c_") {
						// fmt.Println("podFile: ", 2)
						if len(logFile) > 0 {
							log.Fatalf("Please specify a container name since this pod %s/%s has more than one containers.", resNamespace, podName)
							break
						}
						logFile = filepath.Join(dumpDir, "pods", podFile.Name())
					}
				}
			} else {
				logFile = filepath.Join(dumpDir, resNamespace, podName, "logs.txt")
			}
		} else {
			logFile = dumpFile
		}
		if len(logFile) == 0 {
			log.Fatalf("No log is found for pod %s/%s.", resNamespace, podName)
		}
		f, err := os.Open(logFile)
		if err != nil {
			log.Fatalf("Error to read [file=%v]: %v", logFile, err.Error())
		}

		finishedCh := make(chan bool, 1)
		buff := make(chan string, 100)
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
	logsCmd.Flags().StringVarP(&resNamespace, ns, "n", "default", "namespace of the pod")
	logsCmd.Flags().StringVarP(&resContainer, cont, "c", "", "container")
	logsCmd.PersistentFlags().StringVarP(&dumpFile, dumpFileFlag, "f", "./cluster-info.dump", "Path to dump file")
	logsCmd.PersistentFlags().StringVarP(&dumpDir, dumpDirFlag, "d", "", "Path to dump directory")
}

func scanFile(f *os.File, marker string, buff chan string, finishedCh chan bool) {
	scanner := bufio.NewScanner(f)
	canPrint := false
	for scanner.Scan() {
		line := scanner.Text()
		if len(marker) == 0 || (strings.HasPrefix(line, "==== START logs for container") && strings.Contains(line, marker)) {
			canPrint = true
		}
		if canPrint {
			buff <- line
			// fmt.Println(line)
		}
		if len(marker) > 0 && strings.HasPrefix(line, "==== END logs for container") && strings.Contains(line, marker) {
			canPrint = false
		}
	}
	close(buff)
}
