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

package mkchan

import (
	"encoding/json"
	"fmt"
	"github.com/Forkk/GoUpdate/repo"
	"github.com/Forkk/repoman/subcmd"
	"io/ioutil"
	"os"
	"path"
)

type Command struct{}

func (cmd Command) Summary() string { return "Creates a new release channel in a repository." }
func (cmd Command) Description() string {
	return "Creates a new release channel in a given repository. Once this is done, you may set the channel's version ID with the 'chanset' command."
}
func (cmd Command) Usage() string { return "REPO_DIR CHANNEL_ID" }
func (cmd Command) ArgHelp() string {
	return "REPO_DIR - The repository directory to create the channel in.\nCHANNEL_ID - Unique string ID of the channel to create."
}

func (cmd Command) Execute(args ...string) subcmd.Error {
	if len(args) < 2 {
		return subcmd.UsageError("'mkchan' command takes at least two arguments.")
	} else {
		repoDir := args[0]
		chanId := args[1]
		return CreateChan(repoDir, chanId)
	}
}

func CreateChan(repoDir, chanId string) subcmd.Error {
	errFmt := fmt.Sprintf("Can't create channel '%s' for repository '%s': %%s", chanId, repoDir)

	// Make sure the repository directory exists.
	if info, err := os.Stat(repoDir); err != nil {
		var code int
		var msg string
		switch {
		case os.IsNotExist(err):
			msg = "Invalid repository: repository directory doesn't exist."
			code = 10
		case os.IsPermission(err):
			msg = "Can't access repository directory: permission denied."
			code = 20
		default:
			msg = "Can't access repository directory: an unknown error occurred."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf(errFmt, msg), code, err)
	} else if !info.IsDir() {
		return subcmd.MessageError(fmt.Sprintf(errFmt, "The path %s is not a valid repository. Must be a directory.", repoDir), 10)
	}

	indexFilePath := path.Join(repoDir, "index.json")

	// Try to read the index file.
	fileData, indexErr := ioutil.ReadFile(indexFilePath)
	if indexErr != nil {
		// Handle errors.
		var code int
		var msg string
		switch {
		case os.IsNotExist(indexErr):
			msg = "Invalid repository (%s): index file is missing."
			code = 13
		case os.IsPermission(indexErr):
			msg = "Can't access repository's index file: permission denied."
			code = 23
		default:
			msg = "An unknown error occurred when trying to read the repository's index file."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf(errFmt, msg), code, indexErr)
	}

	// Unmarshal the JSON.
	var indexData repo.Index
	if err := json.Unmarshal(fileData, &indexData); err != nil {
		return subcmd.CausedError(fmt.Sprintf(errFmt), 12, err)
	}

	// Now make sure the channel doesn't already exist.
	for _, existingChan := range indexData.Channels {
		if existingChan.Id == chanId {
			return subcmd.MessageError(fmt.Sprintf(errFmt, "Channel already exists"), 42)
		}
	}

	// Now create the channel.
	channel := repo.Channel{Id: chanId, Name: chanId}

	// Add the channel to the index.
	indexData.Channels = append(indexData.Channels, channel)

	// And finally, write the index back to the file.
	if indexFile, err := os.OpenFile(indexFilePath, os.O_RDWR, 0644); err != nil {
		var code int
		var msg string
		switch {
		case os.IsPermission(err):
			msg = "Can't write index file: permission denied."
			code = 45
		default:
			msg = "An unknown error occurred when trying to write the index file."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf(errFmt, msg), code, err)
	} else {
		jsonData, _ := json.Marshal(indexData)
		indexFile.Write(jsonData)
	}

	return nil
}
