package commands

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Status(args ...string) {
	var noColor bool
	if slices.Contains(args, "--no-color") {
		noColor = true
	}
	shouldColor := !noColor && utils.IsTerminal()

	headTree := map[string]string{}
	indexEntries := map[string]string{}
	workingFiles := map[string]string{}

	indexFile := ReadIndex(args...)
	for _, entry := range indexFile.Entries {
		hash := fmt.Sprintf("%x", entry.SHA1[:])
		indexEntries[entry.Path] = hash
	}

	headTreeObject := ReadHead()
	maps.Copy(headTree, ExtractTreeHashs(".", headTreeObject.Entries))

	dirTree, _ := utils.GetDirTree(".", []string{}, false)
	for _, path := range dirTree {
		hash, _, _ := utils.GetBlobHashObject(path)
		workingFiles[path] = fmt.Sprintf("%x", hash[:])
	}

	var changedFiles []string
	var untrackedFiles []string
	var deletedFiles []string
	var stagedFiles []string
	for path, shaDisk := range workingFiles {
		shaIndex, inIndex := indexEntries[path]
		shaHead, inHead := headTree[path]

		if !inIndex && !inHead {
			// untracked
			untrackedFiles = append(untrackedFiles, path)
		}

		if inIndex && shaDisk != shaIndex {
			// modified, not staged
			changedFiles = append(changedFiles, path)
		}

		if inIndex && (shaIndex != shaHead) {
			// staged changes
			stagedFiles = append(stagedFiles, path)
		}
	}

	for path := range headTree {
		_, inIndex := indexEntries[path]
		_, inDisk := workingFiles[path]

		if !inIndex && !inDisk {
			stagedFiles = append(stagedFiles, path)
		} else if inIndex && !inDisk {
			deletedFiles = append(deletedFiles, path)
		}
	}

	branch := utils.GetHeadBranch()

	fmt.Fprintf(os.Stdout, "On branch %s\n", branch)

	if len(deletedFiles) == 0 && len(changedFiles) == 0 && len(untrackedFiles) == 0 && len(stagedFiles) == 0 {
		fmt.Fprintf(os.Stdout, "nothing to commit, working tree clean")
	}

	if len(stagedFiles) > 0 {
		fmt.Fprintf(os.Stdout, "Changes to be committed:\n")
		files := make([]types.FileInfo, 0, len(stagedFiles))

		for _, file := range stagedFiles {
			var stage string
			color := "\033[0m"

			if _, ok := headTree[file]; !ok {
				stage = "new file"
				if shouldColor {
					color = "\033[32m"
				} else {
					color = ""
				}

			} else if _, ok := workingFiles[file]; !ok {
				stage = "deleted"
				if shouldColor {
					color = "\033[31m"
				} else {
					color = ""
				}

			} else {
				stage = "modified"
				if shouldColor {
					color = "\033[0m"
				} else {
					color = ""
				}
			}

			files = append(files, types.FileInfo{Path: file, Stage: stage, Color: color})
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		for _, file := range files {
			fmt.Fprintf(os.Stdout, "\t%s:\t%s\n", file.Stage, file.Path)
		}
		fmt.Println()
	}

	if len(deletedFiles)+len(changedFiles) > 0 {
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
			fmt.Fprintf(os.Stdout, "\t%s%s:\t%s\033[0m\n", file.Color, file.Stage, file.Path)
		}
	}

	if len(untrackedFiles) > 0 {
		fmt.Fprintf(os.Stdout, "\nUntracked files:\n")
		color := ""
		for _, file := range untrackedFiles {
			if shouldColor {
				color = "\033[33m"
			}
			fmt.Fprintf(os.Stdout, "\t%s%s\033[0m\n", color, file)
		}
	}
}

func ReadHead() *types.TreeObject {
	headHash := utils.GetHeadHash()

	if len(headHash) == 0 {
		return &types.TreeObject{}
	}

	headHashFile := CatFileReadObject(string(headHash[0:2]), string(headHash[2:]))
	treeHash := extractCommitTreeHash(headHashFile)
	treeFileData := CatFileReadObject(string(treeHash[0:2]), string(treeHash[2:]))
	headTreeObject, _ := DeserializeTreeObject(treeFileData)

	return headTreeObject
}

func extractCommitTreeHash(data []byte) []byte {
	var hash []byte
	content := strings.Join(strings.Split(string(data), "\x00")[1:], "\x00")
	content = strings.Split(content, "\n")[0]
	strs := strings.Split(content, " ")
	content = strs[len(strs)-1]
	hash = append(hash, []byte(content)...)
	return hash
}

func ExtractTreeHashs(basePath string, entries []types.TreeEntry) map[string]string {
	hashes := map[string]string{}
	for _, entry := range entries {
		if utils.ModeStringToKind(entry.Mode) == "tree" {
			hash := fmt.Sprintf("%x", entry.Hash[:])

			treeFileData := CatFileReadObject(hash[0:2], hash[2:])
			headTreeObject, _ := DeserializeTreeObject(treeFileData)
			subtreeHahses := ExtractTreeHashs(entry.Name, headTreeObject.Entries)
			maps.Copy(hashes, subtreeHahses)
		} else {
			hash := fmt.Sprintf("%x", entry.Hash[:])
			hashes[filepath.Join(basePath, entry.Name)] = hash
		}
	}

	return hashes
}
