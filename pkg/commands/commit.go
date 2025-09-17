package commands

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

type CommitStatus struct {
	Mode  string
	Stage string
	Path  string
}

func Commit(args ...string) {
	if len(args) < 4 {
		fmt.Fprintf(os.Stderr, "usage: ccgit commit -m <message>\n")
		os.Exit(1)
	}

	var messages []string
	for _, arg := range os.Args[2:] {
		if arg != "-m" && arg != "-am" {
			messages = append(messages, arg)
		}
	}

	headTree := map[string]types.TreeEntry{}

	indexFile := ReadIndex(args...)
	headTreeObject := ReadHead()
	maps.Copy(headTree, extractTreeEntries(".", headTreeObject.Entries))
	dirTree, _ := utils.GetDirTree(".", []string{}, false)

	treeHash := WriteTree()
	hash, object := utils.GetCommitHashObject(treeHash, messages...)
	utils.SaveHashedObject(hash, object)
	branch := utils.GetHeadBranch()
	_ = os.MkdirAll(filepath.Join(".git", "refs", "heads"), 0755)
	refBranchHead, err := os.Create(filepath.Join(".git", "refs", "heads", branch))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v", err)
		os.Exit(1)
	}
	defer refBranchHead.Close()

	_, err = fmt.Fprintf(refBranchHead, "%x", hash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "[%s %s] %s\n", branch, fmt.Sprintf("%x", hash[:])[:7], messages[0])
	fmt.Fprintf(os.Stdout, "Date: %s\n", time.Now().Format("Mon Jan 2 15:04:05 2006 -0700"))

	commitTree := []CommitStatus{}

	for _, e := range indexFile.Entries {
		valueHead, inHead := headTree[e.Path]
		if !inHead {
			hash, object, _ := utils.GetBlobHashObject(e.Path)
			utils.SaveHashedObject(hash, object)
			commitTree = append(commitTree, CommitStatus{
				Path: e.Path, Mode: utils.TreeModeString(os.FileMode(e.Mode)), Stage: "create",
			})
		} else if string(valueHead.Hash) != fmt.Sprintf("%x", e.SHA1[:]) {
			hash, object, _ := utils.GetBlobHashObject(e.Path)
			utils.SaveHashedObject(hash, object)
		}
	}

	for path, entry := range headTree {
		if !slices.Contains(dirTree, path) {
			commitTree = append(commitTree, CommitStatus{
				Path: entry.Name, Mode: entry.Mode, Stage: "delete",
			})
		}
	}

	sort.Slice(commitTree, func(i, j int) bool {
		return commitTree[i].Path < commitTree[j].Path
	})

	for _, e := range commitTree {
		if e.Stage == "create" {
			fmt.Printf("create mode %s %s\n", e.Mode, e.Path)
		} else if e.Stage == "delete" {
			fmt.Printf("delete mode %s %s\n", e.Mode, e.Path)
		}
	}
}

func extractTreeEntries(basePath string, entries []types.TreeEntry) map[string]types.TreeEntry {
	hashes := map[string]types.TreeEntry{}
	for _, entry := range entries {
		if utils.ModeStringToKind(entry.Mode) == "tree" {
			hash := fmt.Sprintf("%x", entry.Hash[:])

			treeFileData := CatFileReadObject(hash[0:2], hash[2:])
			headTreeObject, _ := DeserializeTreeObject(treeFileData)
			subtreeHahses := extractTreeEntries(entry.Name, headTreeObject.Entries)
			maps.Copy(hashes, subtreeHahses)
		} else {
			hashes[filepath.Join(basePath, entry.Name)] = entry
		}
	}

	return hashes
}
