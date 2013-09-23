/*
subcmd contains functions, structures, and interfaces used by RepoMan's subcommand system.
*/

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

package subcmd

import (
	"fmt"
)


// Subcommand is an interface for RepoMan's subcommands to implement.
type Command interface {
	// Execute executes the command, returning a subcmd.Error if an error occurred.
	Execute(args ...string) Error

	// Summary returns a one line string that summarizes what this command does.
	Summary() string

	// Description returns a one or multiple line string that serves as a more in-depth explanation of the command's purpose than Summary does.
	Description() string

	// Usage returns a one line string that lists the command's arguments (for example, "REPO_DIR VERSION_NAME VERSION_ID").
	Usage() string

	// ArgHelp returns a one or multiple line string that explains what each of the arguments listed in Usage do.
	ArgHelp() string
}


// Error is an interface that provides information about an error that occurred while executing a subcommand.
type Error interface {
	// Implement the error interface.
	Error() string

	// Message returns just this command's error message without the cause info that the Error function returns.

	// ExitCode specifies what the process's exit code should be when this error is returned.
	ExitCode() int

	// Cause specifies another error that caused this command error. May also be nil if there was no internal error that caused this error.
	Cause() error

	// ShowUsage returns true if the Usage information for the command that returned this error should be printed after the command's Error string.
	ShowUsage() bool
}


// Struct that represents an error that was not caused by another error.
type msgError struct {
	// The error message to display.
	msg string

	// The exit code value that this error should cause the process to exit with.
	exitCode int

	// cause is the error that caused this command error. May be nil.
	cause error

	printUsage bool
}

// Error returns just this error's message string.
func (err msgError) Error() string {
	return err.msg
}

// Message returns the full message that should be printed when this error occurs. This includes the cause's message if there is one.
func (err msgError) Message() string {
	msg := err.msg
	cause := err.cause
	if cause == nil {
		return msg
	} else {
		return fmt.Sprintf("%s\n  Caused by: %s", msg, cause)
	}
}

func (err msgError) ExitCode() int {
	return err.exitCode
}

func (err msgError) Cause() error {
	return err.cause
}

func (err msgError) ShowUsage() bool {
	return err.printUsage
}

// CausedError returns an Error with the given message, exit code, and cause.
func CausedError(message string, exitCode int, cause error) Error {
	return msgError{msg: message, exitCode: exitCode, cause: cause, printUsage: false}
}

// MessageError returns an Error with the given message and exit code.
func MessageError(message string, exitCode int) Error {
	return msgError{msg: message, exitCode: exitCode, cause: nil, printUsage: false}
}

// UsageError returns an Error that will print the command's usage info. If message isn't blank, it will be printed before the usage info.
func UsageError(message string) Error {
	return msgError{msg: message, exitCode: -1, cause: nil, printUsage: true}
}

