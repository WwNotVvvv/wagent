package app

import (
	"os"

	"golang.org/x/term"
)

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiGray   = "\033[90m"
	ansiBold   = "\033[1m"
)

func Colorize(enabled bool, text string, code string) string {
	if !enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}

func Green(enabled bool, text string) string  { return Colorize(enabled, text, ansiGreen) }
func Red(enabled bool, text string) string    { return Colorize(enabled, text, ansiRed) }
func Cyan(enabled bool, text string) string   { return Colorize(enabled, text, ansiCyan) }
func Yellow(enabled bool, text string) string { return Colorize(enabled, text, ansiYellow) }
func Gray(enabled bool, text string) string   { return Colorize(enabled, text, ansiGray) }
func Bold(enabled bool, text string) string   { return Colorize(enabled, text, ansiBold) }

func ColorMode(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		if os.Getenv("NO_COLOR") != "" {
			return false
		}
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}