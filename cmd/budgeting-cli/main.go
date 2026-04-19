// Command budgeting-cli is an HTTP client for the budgeting API, intended for
// use by scripts and AI agents. It reads the server URL and API token from
// env vars BUDGETING_URL and BUDGETING_TOKEN (overridable per-command via flags).
// Output is compact JSON by default; --pretty emits indented JSON.
package main

import (
	"fmt"
	"os"

	"github.com/manolis/budgeting/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "version" || command == "--version" || command == "-v" {
		fmt.Printf("budgeting-cli version %s\n", version.Get())
		return
	}
	if command == "help" || command == "--help" || command == "-h" {
		printUsage()
		return
	}

	args := os.Args[2:]

	switch command {
	case "me":
		runMe(args)
	case "actions":
		runActions(args)
	case "categories":
		runCategories(args)
	case "charts":
		runCharts(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: budgeting-cli <command> [subcommand] [flags]

Global environment variables:
  BUDGETING_URL     Base URL of the budgeting server (e.g. http://localhost:8080)
  BUDGETING_TOKEN   API token generated from the web UI (bdg_...)

Per-command flags (accepted by all subcommands):
  --url <url>       Override BUDGETING_URL
  --token <token>   Override BUDGETING_TOKEN
  --pretty          Indent JSON output

Commands:
  me
  actions list       [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--type income|expense]
                     [--category ID] [--user NAME] [--search TEXT] [--limit N] [--offset N]
  actions create     --type --date --description --amount --category
  actions update ID  [--type] [--date] [--description] [--amount] [--category]
  actions delete ID
  categories list    [--type income|expense]
  categories create  --description --type
  categories update ID [--description] [--type]
  categories delete ID
  charts monthly     [--year YYYY]
  charts categories  [--year YYYY] [--month M]
  version`)
}
