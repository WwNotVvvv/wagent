package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type HITL struct{}

func (h *HITL) Prompt(a Action, timeout time.Duration) bool {
	argvRaw, _ := a.Args["argv"]
	cmdStr := fmt.Sprint(argvRaw)
	fmt.Printf("[HITL] Action: %s %s\n", a.Type, cmdStr)
	fmt.Printf("[HITL] Allow? (y/N, %v timeout): ", timeout)

	result := make(chan bool, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			result <- false
			return
		}
		result <- parseHITLInput(input)
	}()

	select {
	case approved := <-result:
		if approved {
			fmt.Println("[HITL] Approved")
		} else {
			fmt.Println("[HITL] Rejected")
		}
		return approved
	case <-time.After(timeout):
		fmt.Println("[HITL] Timeout — rejected")
		return false
	}
}

func parseHITLInput(input string) bool {
	cleaned := strings.TrimSpace(input)
	return cleaned == "y" || cleaned == "Y"
}