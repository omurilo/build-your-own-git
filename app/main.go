package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/codecrafters-io/git-starter-go/pkg/commands"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: ccgit <command> [<args>...]\n")
		os.Exit(1)
	}

	if str, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch str {
		case "debug", "DEBUG":
			slog.SetLogLoggerLevel(slog.LevelDebug)
		}
	}

	switch command := os.Args[1]; command {
	case "debug-commit":
		var messages []string
		for _, arg := range os.Args[2:] {
			if arg != "-m" && arg != "-am" {
				messages = append(messages, arg)
			}
		}
		hash, object := utils.GetCommitHashObject([20]byte{}, messages...)
		fmt.Printf("Hash: %x, Content: \n%s", hash, string(object))
	case "debug-index":
		indexFile := commands.ReadIndex(os.Args...)
		fmt.Printf("Index File: \n%+v\n", indexFile)
	case "init":
		commands.InitRepository(os.Args...)
	case "cat-file":
		commands.CatFile(os.Stdout, os.Args...)
	case "hash-object":
		commands.GetHashObjects(os.Args...)
	case "add":
		commands.Add(os.Args...)
	case "status":
		commands.Status(os.Args...)
	case "write-tree":
		commands.WriteTree(os.Args...)
	case "commit":
		commands.Commit(os.Args...)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}
