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

/*
create contains the Command struct for repoman's "create" subcommand.
*/

package create

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/MultiMC/repoman/subcmd"

	"github.com/MultiMC/GoUpdate/repo"
)

type Command struct{}

func (cmd Command) Summary() string { return "Creates a new GoUpdate repository." }
func (cmd Command) Description() string {
	return "Creates a new, blank GoUpdate repository at a given path."
}
func (cmd Command) Usage() string { return "REPO_DIR" }
func (cmd Command) ArgHelp() string {
	return "REPO_DIR - The repository's directory. This directory must not already exist. It will be created."
}

func (cmd Command) Execute(args ...string) subcmd.Error {
	// Determine what directory to create the repository in.
	if len(args) <= 0 {
		return subcmd.UsageError("'create' command requires at least one argument.")
	} else {
		repoDir := args[0]
		return CreateRepo(repoDir)
	}
}

func CreateRepo(repoDir string) subcmd.Error {
	fileMode := os.FileMode(0644)
	dirMode  := os.FileMode(0755)

	// Try to create the repository directory. If it already exists, this should cause an error. We shouldn't try to create a repository in a directory that already exists.
	if err := os.Mkdir(repoDir, dirMode); err != nil {
		if os.IsExist(err) {
			// Tell the user we can't overwrite an existing repository.
			return subcmd.CausedError(fmt.Sprintf("Can't create repository at %s because the directory already exists. Cannot create a repository in an existing directory.", repoDir), 11, err)
		} else if os.IsNotExist(err) {
			// Tell the user that the repository's parent directory probably doesn't exist.
			return subcmd.CausedError(fmt.Sprintf("Can't create repository at %s. Make sure the parent directory exists.", repoDir), 12, err)
		} else {
			// An unknown error occurred.
			return subcmd.CausedError(fmt.Sprintf("Failed to create repository at %s. An unknown error occurred.", repoDir), 10, err)
		}
	}

	// Determine the path to the index file.
	indexFilePath := path.Join(repoDir, repo.IndexFileName)

	// Get a new, blank index data struct.
	indexData := repo.NewBlankIndex()

	// Serialize the index structure to JSON...
	jsonData, jsonError := json.Marshal(indexData)

	if jsonError != nil {
		return subcmd.CausedError("Failed to marshal index data to JSON. This probably shouldn't happen...", -1, jsonError)
	}

	// ...and write it to the index file.
	writeError := ioutil.WriteFile(indexFilePath, jsonData, fileMode)

	if writeError != nil {
		return subcmd.CausedError("Failed to write index file.", 20, writeError)
	}

	return nil
}
