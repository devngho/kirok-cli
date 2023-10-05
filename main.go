package main

import (
	"os"
)

type Command struct {
	name        string
	description string
	execute     func(args []string)
}

var commands = []Command{
	HelpCommand,
	InitCommand,
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		HelpCommand.execute(args)
		return
	}

	for _, command := range commands {
		if command.name == args[0] {
			command.execute(args[1:])
			return
		}
	}

	println("Unknown command: " + args[0])
	println()
	HelpCommand.execute(args)
}

var HelpCommand = Command{
	name:        "help",
	description: "Prints this help message.",
	execute: func(args []string) {
		println("Usage: kirok <command> [arguments]")
		println()
		println("Available commands:")
		println("  help\t\tPrints this help message.")
		println("  init [version]\t\tInitializes a new kirok project.")
	},
}
