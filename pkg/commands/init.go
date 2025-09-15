package commands

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func InitRepository(args ...string) {
	baseDir := "."
	if len(args) > 2 {
		baseDir = args[2]
	}

	err := utils.CheckGitRepo(baseDir, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	for _, dir := range []string{".git", ".git/hooks", ".git/objects/info", ".git/objects/pack", ".git/refs"} {
		if baseDir != "" && dir != baseDir {
			dir = fmt.Sprintf("%s/%s", baseDir, dir)
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		}
	}

	headFileContents := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(fmt.Sprintf("%s/.git/HEAD", baseDir), headFileContents, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
	}

	fmt.Println("Initialized git directory")

}
