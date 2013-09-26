// Copyright 2013 MultiMC Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"github.com/Forkk/repoman/create"
	"github.com/Forkk/repoman/mkchan"
	"github.com/Forkk/repoman/subcmd"
	"github.com/Forkk/repoman/update"
	"os"
)

var commands map[string]subcmd.Command

func main() {
	// Initialize the command map.
	commands = map[string]subcmd.Command{
		"help":   helpCommand{},
		"create": create.Command{},
		"update": update.Command{},
		"mkchan": mkchan.Command{},
	}

	// Get the command line arguments.
	args := os.Args

	// There must be at least one argument (the sub-command). If not, print the help text and exit.
	if len(args) <= 1 {
		executeCommand(commands["help"], "help")
		os.Exit(1)
	}

	// If there is a command argument, get it.
	cmd := args[1]

	// Look up the command in the command map.
	if cmdInfo, ok := commands[cmd]; ok {
		// Run the command.
		os.Exit(executeCommand(cmdInfo, cmd, args[2:]...))
	} else {
		// If the command doesn't exist, print the help message and exit.
		os.Exit(executeCommand(commands["help"], "help", args[2:]...))
	}

	return
}

// executeCommand executes the given command and returns the exit code that the process should exit with.
func executeCommand(cmd subcmd.Command, cmdName string, args ...string) int {
	err := cmd.Execute(args...)
	if err == nil {
		return 0
	} else {
		if err.Error() != "" {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
		if err.ShowUsage() {
			fmt.Fprintf(os.Stderr, "Usage: %s %s\n", cmdName, cmd.Usage())
		}

		return err.ExitCode()
	}
}

//////////////////////////////
//////// HELP COMMAND ////////
//////////////////////////////

type helpCommand struct{}

func (cmd helpCommand) Summary() string {
	return "Shows a list of available commands and information about them."
}
func (cmd helpCommand) Description() string {
	return "Shows a list of available commands and information about them."
}
func (cmd helpCommand) Usage() string   { return "" }
func (cmd helpCommand) ArgHelp() string { return "" }

func (cmd helpCommand) Execute(args ...string) subcmd.Error {
	help := fmt.Sprintf("Usage: %s COMMAND [arg...]\n", os.Args[0])

	for cmdStr, cmdInfo := range commands {
		help += fmt.Sprintf("    %-10.10s%s\n", cmdStr, cmdInfo.Summary())
	}

	fmt.Fprintf(os.Stderr, help)

	return nil
}
