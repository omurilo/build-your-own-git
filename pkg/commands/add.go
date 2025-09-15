package commands

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Add(args ...string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: ccgit add [<file>...]\n")
		os.Exit(1)
	}

	if str, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch str {
		case "debug", "DEBUG":
			slog.SetLogLoggerLevel(slog.LevelDebug)
		}
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

	var paths []string
	for _, arg := range args[2:] {
		dirOrFile, err := os.Open(arg)
		if err != nil {
			log.Fatalf("Error to add file(s), %v", err)
		}
		stat, err := dirOrFile.Stat()
		if err != nil {
			log.Fatalf("Error to add file(s), %v", err)
		}
		if stat.IsDir() {
			fileNames, _ := utils.GetDirTree(stat.Name(), ignoreNames, true)
			paths = append(paths, fileNames...)
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		hash, _, content := utils.GetBlobHashObject(path)
		slog.Debug(fmt.Sprintf("%s - %x - %+v", path, hash, string(content)))

		UpdateIndex(path, hash)
	}
}
