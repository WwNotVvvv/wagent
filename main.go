package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"wagent/internal/app"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wagent [flags] <task>\n")
		fmt.Fprintf(os.Stderr, "       wagent --interactive [flags]\n")
		fmt.Fprintf(os.Stderr, "       wagent key set|status|clear\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	mockFlag := flag.String("mock", "", "Path to MockLLM script (disables real LLM)")
	configFlag := flag.String("config", "", "Path to config file (optional; defaults to ./wagent.toml or built-in defaults)")
	interactiveFlag := flag.Bool("interactive", false, "Run in interactive REPL mode")
	colorFlag := flag.String("color", "auto", "Color mode: auto, always, never")
	flag.Parse()

	colorEnabled := app.ColorMode(*colorFlag)

	// Subcommand dispatch: key set/status/clear
	if flag.NArg() > 0 && flag.Arg(0) == "key" {
		handleKeyCommand(flag.Args()[1:])
		return
	}

	// Main flow: run task
	if flag.NArg() == 0 && !*interactiveFlag {
		flag.Usage()
		os.Exit(1)
	}
	task := strings.Join(flag.Args(), " ")

	// Config discovery: explicit --config -> strict load; implicit -> try wagent.toml -> defaults
	configExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configExplicit = true
		}
	})

	var cfg *app.Config
	var err error
	if configExplicit {
		cfg, err = app.LoadConfigStrict(*configFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", *configFlag, err)
			os.Exit(1)
		}
	} else {
		cfg, err = app.LoadConfigOrDefault("wagent.toml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
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

	tr, err := app.NewTraceRecorder(cfg, task, func(s string) string {
		return app.RedactAPIKey(s, apiKey)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create trace: %v\n", err)
	} else {
		harness.SetTraceRecorder(tr)
		defer tr.Flush()
	}

	harness.SetOnStep(func(ev app.StepEvent) {
		formatStepEvent(ev, colorEnabled)
	})

	if *interactiveFlag {
		runREPL(harness, colorEnabled)
		return
	}

	result, err := harness.Run(task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(app.Bold(colorEnabled, result))
}

func formatStepEvent(ev app.StepEvent, enabled bool) {
	prefix := fmt.Sprintf("[%d/%d]", ev.Step, ev.MaxSteps)
	switch ev.Phase {
	case app.StepEventAction:
		actionDesc := ev.Action.Type
		if ev.Action.Type == "run_command" {
			if argv, ok := ev.Action.Args["argv"]; ok {
				actionDesc = fmt.Sprintf("run_command: %v", argv)
			}
		}
		fmt.Println(app.Cyan(enabled, prefix+" ACTION "+actionDesc))
	case app.StepEventGuard:
		switch ev.Decision {
		case "allow":
			fmt.Println(app.Green(enabled, prefix+" ALLOW "+ev.Reason))
		case "deny":
			fmt.Println(app.Red(enabled, prefix+" DENY "+ev.Reason))
		case "ask":
			fmt.Println(app.Yellow(enabled, prefix+" ASK "+ev.Reason))
		}
	case app.StepEventResult:
		fmt.Println(app.Gray(enabled, prefix+" RESULT "+ev.Summary))
	case app.StepEventError:
		fmt.Println(app.Red(enabled, prefix+" ERROR "+ev.Error))
	}
}

func runREPL(harness *app.Harness, colorEnabled bool) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("wagent interactive mode. Type a task to run, or :help for commands.")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch line {
		case "exit", ":exit":
			fmt.Println("Goodbye.")
			return
		case ":help":
			fmt.Println("Commands:")
			fmt.Println("  <task>      Run a task")
			fmt.Println("  :help       Show this help")
			fmt.Println("  :reset      Clear conversation context")
			fmt.Println("  :exit       Exit interactive mode")
			fmt.Println("  exit        Exit interactive mode")
		case ":reset":
			harness.Reset()
			fmt.Println("Context cleared.")
		default:
			if strings.HasPrefix(line, ":") {
				fmt.Println("Unknown command. Type :help for help.")
				continue
			}
			result, err := harness.Run(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				fmt.Println(app.Bold(colorEnabled, result))
			}
		}
	}
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