package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: changescope analyze [path]")
		os.Exit(1)
	}

	switch args[0] {
	case "analyze":
		path := "."
		if len(args) >= 2 {
			path = args[1]
		}
		// TODO(sub-task-6): wire differ → parser → searcher → report
		fmt.Printf("ChangeScope — analyzing %s\n", path)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}
