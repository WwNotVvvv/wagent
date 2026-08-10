package app

import (
	"fmt"
	"runtime"
)

func testCommandEcho(text string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd.exe", "/c", "echo " + text}
	}
	return []string{"sh", "-c", "printf '%s\\n' \"$1\"", "wagent-test", text}
}

func testCommandExit(code int) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd.exe", "/c", "exit", fmt.Sprint(code)}
	}
	return []string{"sh", "-c", fmt.Sprintf("exit %d", code)}
}

func testCommandStderr(text string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd.exe", "/c", "echo", text, ">&2"}
	}
	return []string{"sh", "-c", "printf '%s\\n' \"$1\" >&2", "wagent-test", text}
}

func testCommandWorkingDirectory() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd.exe", "/c", "cd"}
	}
	return []string{"pwd"}
}

func toAnySlice(values []string) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}
