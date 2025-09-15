package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Status(args ...string) {
	indexFile := ReadIndex(args...)

	// for each entry, check if it exists on dir tree to determine the status
	matchedFiles := map[string]bool{}
	var changedFiles []string
	var untrackedFiles []string
	var deletedFiles []string

	ignoreFile, _ := os.ReadFile(".gitignore")
	var ignoreNames []string
	i := 0
	for i < len(ignoreFile) {
		startPath := i

		for i < len(ignoreFile) && ignoreFile[i] != '\n' {
			i++
		}

		pathComp := string(ignoreFile[startPath:i])

		ignoreNames = append(ignoreNames, pathComp)
		i++
	}

	dirNames, _ := utils.GetDirTree(".", ignoreNames, false)

	for _, fileName := range dirNames {
		for _, entry := range indexFile.Entries {
			if _, ok := matchedFiles[entry.Path]; !ok {
				matchedFiles[entry.Path] = false
			}
			if entry.Path == fileName {
				matchedFiles[entry.Path] = true
				hash, _, _ := utils.GetBlobHashObject(entry.Path)
				if hash != entry.SHA1 {
					changedFiles = append(changedFiles, entry.Path)
				}
			}
		}

		_, ok := matchedFiles[fileName]

		if !ok {
			untrackedFiles = append(untrackedFiles, fileName)
		}
	}

	for fileName, matched := range matchedFiles {
		if !matched {
			deletedFiles = append(deletedFiles, fileName)
		}
	}

	branch := utils.GetHeadBranch()

	fmt.Fprintf(os.Stdout, "On branch %s\n", branch)

	if len(deletedFiles) == 0 && len(changedFiles) == 0 && len(untrackedFiles) == 0 {
		fmt.Fprintf(os.Stdout, "nothing to commit, working tree clean")
	}

	if len(deletedFiles) > 0 || len(changedFiles) > 0 {
		fmt.Fprintf(os.Stdout, "Changes not staged for commit:\n")
		files := make([]types.FileInfo, 0, len(changedFiles)+len(deletedFiles))

		for _, file := range deletedFiles {
			files = append(files, types.FileInfo{Path: file, Stage: "deleted"})
		}

		for _, file := range changedFiles {
			files = append(files, types.FileInfo{Path: file, Stage: "modified"})
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		for _, file := range files {
			fmt.Fprintf(os.Stdout, "\t%s:\t%s\n", file.Stage, file.Path)
		}
	}

	if len(untrackedFiles) > 0 {
		fmt.Fprintf(os.Stdout, "\nUntracked files:\n")
		for _, file := range untrackedFiles {
			fmt.Fprintf(os.Stdout, "\t%s\n", file)
		}
	}
}
