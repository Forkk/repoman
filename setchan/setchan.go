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

package setchan

import (
	"encoding/json"
	"fmt"
	"github.com/MultiMC/GoUpdate/repo"
	"github.com/MultiMC/repoman/subcmd"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type Command struct{}

func (cmd Command) Summary() string {
	return "Sets the current version of a repository's channel. Can also be used to remove the channel if the version isn't specified."
}
func (cmd Command) Description() string {
	return "Sets the current version of the given channel in the given repository to the given version ID. If no version ID is specified, the given channel will be deleted."
}
func (cmd Command) Usage() string {
	return "REPO_DIR CHANNEL_ID [VERSION_ID]"
}
func (cmd Command) ArgHelp() string {
	return "REPO_DIR - The repository directory to create the channel in.\nCHANNEL_ID - Unique string ID of the channel to create.\nVERSION_ID - Optional version ID. If specified, the given channel's current version will be set to this version ID. If not specified, the given channel will be removed."
}

func (cmd Command) Execute(args ...string) subcmd.Error {
	if len(args) < 2 {
		return subcmd.UsageError("'setchan' command takes at least two arguments.")
	} else {
		repoDir := args[0]
		chanId := args[1]
		versionIdStr := "-1"

		if len(args) >= 3 {
			versionIdStr = args[2]
		}

		versionId, err := strconv.ParseInt(versionIdStr, 10, 0)

		if err != nil {
			return subcmd.UsageError("Version ID must be a positive integer.")
		} else {
			return SetChan(repoDir, chanId, int(versionId))
		}
	}
}

func SetChan(repoDir, chanId string, versionId int) subcmd.Error {
	var errFmt string
	if versionId >= 0 {
		errFmt = fmt.Sprintf("Can't set channel '%s' to version '%d' for repository '%s': %%s", chanId, versionId, repoDir)
	} else {
		errFmt = fmt.Sprintf("Can't remove channel '%s' from repository '%s': %%s", chanId, repoDir)
	}

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
		return subcmd.CausedError(fmt.Sprintf(errFmt, err.Error()), 12, err)
	}

	// Now, check if the channel exists and, if so, set its current version to the version ID given (or remove it if version ID is < 0).
	chanExists := false
	for i, existingChan := range indexData.Channels {
		if existingChan.Id == chanId {
			if versionId < 0 {
				// Removing from a slice is messy, but this works.
				// Basically, it splits the slice into two parts, the first half being everything before the index to remove and the second half being everything after the index to remove.
				// Then, just append those two together.
				indexData.Channels = append(indexData.Channels[:i], indexData.Channels[i+1:]...)
			} else {
				existingChan.CurrentVersion = versionId
				indexData.Channels[i] = existingChan
			}
			chanExists = true
			break
		}
	}

	// If the channel doesn't already exist, add it.
	if !chanExists {
		// Create a new, blank channel data structure with the given version ID as its current version. We'll overwrite this with the existing one if it exists.
		channel := repo.Channel{Id: chanId, Name: chanId, CurrentVersion: versionId}

		// Add the channel to the list.
		indexData.Channels = append(indexData.Channels, channel)
	}

	// Finally, write the index back to the file.
	if indexFile, err := os.OpenFile(indexFilePath, os.O_WRONLY | os.O_TRUNC, 0644); err != nil {
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
