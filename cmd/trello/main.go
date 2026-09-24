// Command trello is a command-line client for the Trello REST API.
package main

import (
	"os"

	"github.com/nomadicworks/trello-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
