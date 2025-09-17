package commands

import (
	"fmt"
	"log/slog"
	"maps"
	"os"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Add(args ...string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: ccgit add [<file>...]\n")
		os.Exit(1)
	}

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

	var errorPaths []string
	var paths []string
	for _, arg := range args[2:] {
		stat, err := os.Stat(arg)
		if err != nil {
			if os.IsNotExist(err) {
				errorPaths = append(errorPaths, arg)
				continue
			} else {
				fmt.Fprintf(os.Stderr, "Error to add file: %v\n", err)
				os.Exit(1)
			}
		}

		if stat.IsDir() {
			fileNames, _ := utils.GetDirTree(stat.Name(), ignoreNames, true)
			paths = append(paths, fileNames...)
		} else {
			paths = append(paths, arg)
		}
	}

	if len(errorPaths) > 0 {
		headTree := map[string]string{}
		indexEntries := map[string]string{}

		indexFile := ReadIndex(args...)
		for _, entry := range indexFile.Entries {
			hash := fmt.Sprintf("%x", entry.SHA1[:])
			indexEntries[entry.Path] = hash
		}

		headTreeObject := ReadHead()
		maps.Copy(headTree, ExtractTreeHashs(".", headTreeObject.Entries))

		for _, path := range errorPaths {
			// _, inHead := headTree[path]
			indexHash, inIndex := indexEntries[path]

			var newIndexEntries []types.Entry
			if inIndex {
				for _, entry := range indexFile.Entries {
					if indexHash == fmt.Sprintf("%x", entry.SHA1[:]) {
						continue
					}

					newIndexEntries = append(newIndexEntries, entry)
				}

				WriteIndex(newIndexEntries)
				return
			} else {
				fmt.Fprintf(os.Stderr, "fatal: pathspec '%s' did not match any files\n", path)
				os.Exit(1)
			}
		}
	}

	if len(paths) > 0 {
		for _, path := range paths {
			hash, _, content := utils.GetBlobHashObject(path)
			slog.Debug(fmt.Sprintf("%s - %x - %+v", path, hash, string(content)))

			UpdateIndex(path, hash)
		}
	}
}
