package cli

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "show log of a pod",
	Long:  `show log of a pod`,
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
		fmt.Printf("parsing dump file %s\n", dumpFile)
		f, err := os.Open(dumpFile)
		if err != nil {
			log.Fatalf("Error to read [file=%v]: %v", dumpFile, err.Error())
		}

		finishedCh := make(chan bool, 1)
		buff := make(chan string, 100)
		go scanFile(f, args[0], buff, finishedCh)
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
	rootCmd.AddCommand(logCmd)
}

func scanFile(f *os.File, podName string, buff chan string, finishedCh chan bool) {
	scanner := bufio.NewScanner(f)
	canPrint := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "==== START logs for container") && strings.Contains(line, podName) {
			canPrint = true
		}
		if canPrint {
			buff <- line
			// fmt.Println(line)
		}
		if strings.HasPrefix(line, "==== END logs for container") && strings.Contains(line, podName) {
			canPrint = false
		}
	}
	close(buff)
}
