package commands

import (
	"fmt"
	"os"
	"slices"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func GetHashObjects(args ...string) {
	err := utils.CheckGitRepo(".", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	var path string
	if args[2] == "-w" {
		path = args[3]
	} else {
		path = args[2]
	}

	var kind string
	if slices.Contains(args, "-t") {
		pos := utils.GetSlicePosition(args, "-t")
		if pos >= 0 {
			kind = args[pos+1]
		}
	} else {
		kind = "blob"
	}

	var hash [20]byte
	var object []byte
	switch kind {
	case "blob":
		hash, object, _ = utils.GetBlobHashObject(path)
	// case "tree":
	// 	hash, object, _ = utils.GetTreeHashObject(path)
	}

	if slices.Contains(args, "-w") {
		utils.SaveHashedObject(hash, object)
	}

	fmt.Printf("%x", hash)
}
