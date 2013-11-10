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

package update

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MultiMC/repoman/md5util"
	"github.com/MultiMC/repoman/subcmd"

	"github.com/MultiMC/GoUpdate/repo"
)

type Command struct{}

func (cmd Command) Summary() string {
	return "Updates a given repository with the files in a given directory."
}
func (cmd Command) Description() string {
	return "The update command updates a given repository with a set of files in a given directory. It then creates a new version for those files based on the given arguments."
}
func (cmd Command) Usage() string {
	return "REPO_DIR FILE_STORAGE URL_BASE UPDATE_DIR VERSION_NAME VERSION_ID"
}
func (cmd Command) ArgHelp() string {
	return "REPO_DIR - The directory name of the repository to update.\nFILE_STORAGE - The path to the directory where the update files will be stored.\nURL_BASE - The base URL to use to create HTTP sources in the new version. This should point to the file storage directory so any files in the file storage directory can be accessed via this base URL.\nUPDATE_DIR - The directory containing the new version's files.\nVERSION_NAME - The version name (e.g. 4.3.0.42) of the new version.\nVERSION_ID - The new version's integer ID."
}

func (cmd Command) Execute(args ...string) subcmd.Error {
	if len(args) < 6 {
		return subcmd.UsageError("'update' command requires at least six arguments.")
	} else {
		repoDir := args[0]
		filesDir := args[1]
		urlBase := args[2]
		newVersionDir := args[3]
		versionName := args[4]
		versionIdStr := args[5]

		versionId, err := strconv.ParseInt(versionIdStr, 10, 0)

		if err != nil {
			return subcmd.UsageError("Version ID must be a positive integer.")
		} else {
			return UpdateRepo(repoDir, filesDir, urlBase, newVersionDir, versionName, int(versionId))
		}
	}
}

// Structure for holding information about a file that already exists in the file storage directory.
type fileStorageData struct {
	// Path to this file relative to the file storage directory.
	FileStoragePath string

	// Path where this file should be installed.
	InstallPath string

	MD5 string
}

func UpdateRepo(repoDir, filesDir, urlBase, newVersionDir, versionName string, versionId int) subcmd.Error {
	fileMode := os.FileMode(0644)
	if !strings.HasSuffix(urlBase, "/") {
		urlBase += "/"
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
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, err)
	} else if !info.IsDir() {
		return subcmd.MessageError(fmt.Sprintf("Can't update repository. The path %s is not a valid repository. Must be a directory.", repoDir), 10)
	}

	// Also make sure the file storage directory exists.
	if info, err := os.Stat(filesDir); err != nil {
		var code int
		var msg string
		switch {
		case os.IsNotExist(err):
			msg = "Invalid file storage directory: file storage directory doesn't exist."
			code = 11
		case os.IsPermission(err):
			msg = "Can't access file storage directory: permission denied."
			code = 21
		default:
			msg = "Can't access file storage directory: an unknown error occurred."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, err)
	} else if !info.IsDir() {
		return subcmd.MessageError(fmt.Sprintf("The path %s is not a valid file storage directory. Must be a directory.", filesDir), 11)
	}

	// And the new version's directory...
	if info, err := os.Stat(filesDir); err != nil {
		var code int
		var msg string
		switch {
		case os.IsNotExist(err):
			msg = "Invalid new version directory: new version directory doesn't exist."
			code = 12
		case os.IsPermission(err):
			msg = "Can't access new version directory: permission denied."
			code = 22
		default:
			msg = "Can't access new version directory: an unknown error occurred."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, err)
	} else if !info.IsDir() {
		return subcmd.MessageError(fmt.Sprintf("The path %s is not a valid new version directory. Must be a directory.", filesDir), 12)
	}

	// Get the path to the index file.
	indexFilePath := path.Join(repoDir, repo.IndexFileName)

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
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, indexErr)
	}

	// Unmarshal the JSON.
	var indexData repo.Index
	if err := json.Unmarshal(fileData, &indexData); err != nil {
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir), 12, err)
	}

	// TODO: Cache calculated MD5s for the file storage directory.
	newVersionMD5s, nvMD5Err := md5util.RecursiveMD5Calc(newVersionDir, []string{})
	if nvMD5Err != nil {
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: Failed to calculate MD5s for new version directory (%s).", repoDir, filesDir), 30, nvMD5Err)
	}

	fileStorageMD5s, fsMD5Err := md5util.RecursiveMD5Calc(filesDir, []string{})
	if fsMD5Err != nil {
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: Failed to calculate MD5s for file storage directory (%s).", repoDir, filesDir), 31, fsMD5Err)
	}

	// File storage map. This maps the install paths of files to their path within the file storage directory.
	fileStorageMap := []fileStorageData{}

	if len(fileStorageMD5s) <= 0 {
		// HACK: This array needs at least one entry in order for the below code to actually add files to storage, so we add a dummy entry.
		// The other option was to have two separate loops and meh.
		fileStorageMD5s = []md5util.FileMD5Data{md5util.FileMD5Data{Path: "", MD5: ""}}
	}

	addToStorage := []fileStorageData{}
	for _, nvMD5Data := range newVersionMD5s {
		for i, fsMD5Data := range fileStorageMD5s {
			//fmt.Printf("%s vs %s", fsMD5Data, nvMD5Data)
			if nvMD5Data.MD5 == fsMD5Data.MD5 {
				// Map all the files we already have in storage.
				fileStorageMap = append(fileStorageMap, fileStorageData{fsMD5Data.Path, nvMD5Data.Path, fsMD5Data.MD5})
				break
			} else if i == len(fileStorageMD5s)-1 {
				// If we're on the last file storage MD5 entry and still haven't found a match, add an entry to the addToStorage list.

				// We need to figure out where we want to put our file in the file storage directory. It needs to be unique.
				// To do this, we can just prepend the first few characters of the MD5 sum to the filename.
				// If the file already exists, we'll add another character of the MD5 sum. If we run out of characters, we'll start numbering.
				// Yes, I know this is a bit of a messy way to do things. Meh.
				storageNameBase := filepath.Base(nvMD5Data.Path)
				prefixSize := 4
				prefix := nvMD5Data.MD5[:prefixSize]
				prefixNum := -1
				storageName := fmt.Sprintf("%s-%s", prefix, storageNameBase)
				for _, err := os.Stat(path.Join(filesDir, storageName)); !os.IsNotExist(err); _, err = os.Stat(path.Join(filesDir, storageName)) {
					if prefixSize < 32 {
						prefixSize++
					} else {
						prefixNum++
					}
					prefix = nvMD5Data.MD5[:prefixSize]
					if prefixNum < 0 {
						storageName = fmt.Sprintf("%s-%s", prefix, storageNameBase)
					} else {
						storageName = fmt.Sprintf("%s-%d-%s", prefix, prefixNum, storageNameBase)
					}
				}

				addToStorage = append(addToStorage, fileStorageData{storageName, nvMD5Data.Path, nvMD5Data.MD5})
			}
		}
	}

	// Now, we need to go through our add to storage list, copy all the files into file storage, and add them to our file mapping.
	for _, mapping := range addToStorage {
		outFilePath := path.Join(filesDir, mapping.FileStoragePath)
		fileOut, createErr := os.OpenFile(outFilePath, os.O_WRONLY | os.O_CREATE | os.O_EXCL, fileMode)
		if createErr != nil {
			return subcmd.CausedError(fmt.Sprintf("Failed updating repository %s. Couldn't create file %s.", repoDir, outFilePath), 42, createErr)
		}
		inFilePath := path.Join(newVersionDir, mapping.InstallPath)
		fileIn, readErr := os.Open(inFilePath)
		if readErr != nil {
			return subcmd.CausedError(fmt.Sprintf("Failed updating repository %s. Couldn't read file %s.", repoDir, inFilePath), 43, readErr)
		}
		io.Copy(fileOut, fileIn)
		fileStorageMap = append(fileStorageMap, mapping)
	}

	// Now that we're done with that crap, we can start building the version object.

	// Create the version data structure.
	versionData := repo.NewVersion(versionId, versionName)

	// Now, build the file list.
	for _, fsMapData := range fileStorageMap {
		fileInfo := repo.FileInfo{Path: fsMapData.InstallPath, Sources: []repo.FileSource{}, MD5: fsMapData.MD5}

		// Add sources.
		fileInfo.Sources = []repo.FileSource{repo.FileSource{SourceType: "http", Url: urlBase + fsMapData.FileStoragePath}}

		versionData.Files = append(versionData.Files, fileInfo)
	}

	// Add the new version data to the index.
	indexData.Versions = append(indexData.Versions, repo.VersionSummary{Id: versionId, Name: versionName})

	// Now write the version data to its file.
	if verFile, err := os.OpenFile(path.Join(repoDir, fmt.Sprintf("%d.json", versionId)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode); err != nil {
		var code int
		var msg string
		switch {
		case os.IsExist(err):
			msg = fmt.Sprintf("Version %d already exists.", versionId)
			code = 44
		case os.IsPermission(err):
			msg = "Can't write version file: permission denied."
			code = 45
		default:
			msg = "An unknown error occurred when trying to write the version file."
			code = -2
		}
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, err)
	} else {
		//jsonData, _ := json.MarshalIndent(versionData, "", "    ")
		jsonData, _ := json.Marshal(versionData)
		verFile.Write(jsonData)
	}

	// And finally, write the index file.
	if indexFile, err := os.OpenFile(indexFilePath, os.O_RDWR, fileMode); err != nil {
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
		return subcmd.CausedError(fmt.Sprintf("Can't update repository %s: %s", repoDir, msg), code, err)
	} else {
		jsonData, _ := json.Marshal(indexData)
		indexFile.Write(jsonData)
	}

	return nil
}
