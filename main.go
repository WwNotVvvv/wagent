package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"wagent/internal/app"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wagent [flags] <task>\n")
		fmt.Fprintf(os.Stderr, "       wagent key set|status|clear\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	mockFlag := flag.String("mock", "", "Path to MockLLM script (disables real LLM)")
	configFlag := flag.String("config", "wagent.toml", "Path to config file")
	flag.Parse()

	// Subcommand dispatch: key set/status/clear
	if flag.NArg() > 0 && flag.Arg(0) == "key" {
		handleKeyCommand(flag.Args()[1:])
		return
	}

	// Main flow: run task
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	task := strings.Join(flag.Args(), " ")

cfg, err := app.LoadConfigOrDefault(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var llm app.LLM
	var apiKey string
	if *mockFlag != "" {
		m := app.NewMockLLM()
		if err := m.LoadScript(*mockFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading mock script: %v\n", err)
			os.Exit(1)
		}
		llm = m
	} else {
		creds := app.NewCredentialStore()
		key, err := creds.Get()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Set WAGENT_API_KEY or run 'wagent key set'\n")
			os.Exit(1)
		}
		apiKey = key
		llm = app.NewOpenAILLM(apiKey, cfg.LLM.Model, cfg.LLM.BaseURL)
	}

	harness := app.NewHarness(cfg, llm)

	tr, err := app.NewTraceRecorder(cfg, task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create trace: %v\n", err)
	} else {
		if apiKey != "" {
			tr.SetRedactFunc(func(s string) string {
				return app.RedactAPIKey(s, apiKey)
			})
		}
		harness.SetTraceRecorder(tr)
		defer tr.Flush()
	}

	result, err := harness.Run(task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}

func handleKeyCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: wagent key set|status|clear\n")
		os.Exit(1)
	}
	creds := app.NewCredentialStore()
	switch args[0] {
	case "set":
		key, err := creds.InteractivePrompt()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := creds.Set(key); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving key: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("API Key saved to keychain.")
	case "status":
		ok, err := creds.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if ok {
			fmt.Println("API Key is configured.")
		} else {
			fmt.Println("API Key is not configured.")
		}
	case "clear":
		if err := creds.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("API Key cleared.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown key subcommand: %s\n", args[0])
		os.Exit(1)
	}
}