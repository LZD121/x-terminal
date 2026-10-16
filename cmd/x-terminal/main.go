// Command x-terminal is a read-only x.com client for a terminal pane: it
// renders timelines, threads, search results, profiles and bookmark/like lists
// as plain text and never writes to the account.
//
// This file is only the process boundary. Argument handling, dispatch and exit
// codes live in internal/cli; every verb's body lives with the package that
// implements it, so nothing here decides what the program can do.
package main

import (
	"os"

	"github.com/LZD121/x-terminal/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
