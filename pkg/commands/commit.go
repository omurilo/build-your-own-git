package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

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

	fmt.Printf("%v %x", refBranchHead, hash)
	_, err = fmt.Fprintf(refBranchHead, "%x", hash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "[%s %x] %s\n", branch, hash[0:7], messages[0])
	fmt.Fprintf(os.Stdout, "Date: %s\n", time.Now().Format("Mon Jan 2 15:04:05 2006 -0700"))

	indexFile := ReadIndex(args...)

	for _, e := range indexFile.Entries {
		hash, object, _ := utils.GetBlobHashObject(e.Path)
		utils.SaveHashedObject(hash, object)
		fmt.Printf("create mode %s %s\n", utils.TreeModeString(os.FileMode(e.Mode)), e.Path)
	}
}
