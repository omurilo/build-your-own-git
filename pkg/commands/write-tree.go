package commands

import (
	"crypto/sha1"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func WriteTree(args ...string) [20]byte {
	indexFile := ReadIndex(args...)

	if slices.Contains(args, "-d") {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if indexFile.Extensions != nil {
		slog.Debug(fmt.Sprintf("Msg: Has extensions, check if have tree extension, Extension: %+v", indexFile.Extensions))
		for _, extension := range indexFile.Extensions {
			if extension.Sig == "TREE" {
				for _, entry := range extension.Entries {
					slog.Debug(fmt.Sprintf("Entry: %+v\n", entry))
				}
			}
		}
	}

	trees := map[string][]types.TreeEntry{}
	subtrees := map[string][]types.TreeEntry{}

	for _, entry := range indexFile.Entries {
		dir, file := filepath.Split(entry.Path)
		if filepath.Base(dir) != filepath.Clean(dir) {
			dir = filepath.Clean(dir)
			if _, ok := trees[dir]; !ok {
				subtrees[dir] = []types.TreeEntry{}
			}
			subtrees[dir] = append(subtrees[dir], types.TreeEntry{Name: file, Mode: utils.TreeModeString(os.FileMode(entry.Mode)), Hash: entry.SHA1[:]})
		} else {
			dir = filepath.Clean(dir)
			if _, ok := trees[dir]; !ok {
				trees[dir] = []types.TreeEntry{}
			}
			trees[dir] = append(trees[dir], types.TreeEntry{Name: file, Hash: entry.SHA1[:], Mode: utils.TreeModeString(os.FileMode(entry.Mode))})
		}
	}

	for key, tree := range subtrees {
		baseDir := filepath.Dir(filepath.Clean(key))
		dir := filepath.Base(filepath.Clean(key))
		if _, ok := trees[baseDir]; ok {
			treeObject := &types.TreeObject{Entries: tree}
			object := treeObject.ToBytes()
			hash := sha1.Sum(object)
			utils.SaveHashedObject(hash, object)

			trees[baseDir] = append(trees[baseDir], types.TreeEntry{Name: dir, Mode: "040000", Hash: hash[:]})
			slog.Debug(fmt.Sprintf("BaseDir: %s - Dir: %s - Hash: %x\n", baseDir, dir, hash))
		}
	}

	finalTree := []types.TreeEntry{}
	for key, tree := range trees {
		if key == "." {
			finalTree = append(finalTree, tree...)
		} else {
			treeObject := &types.TreeObject{Entries: tree}
			object := treeObject.ToBytes()
			hash := sha1.Sum(object)
			utils.SaveHashedObject(hash, object)

			finalTree = append(finalTree, types.TreeEntry{Name: key, Mode: "040000", Hash: hash[:]})
			slog.Debug(fmt.Sprintf("Dir: %s - Hash: %x\n", key, hash))
		}
	}

	slog.Debug(fmt.Sprintf("Trees: %+v", trees))

	treeObject := &types.TreeObject{Entries: finalTree}
	object := treeObject.ToBytes()
	hash := sha1.Sum(object)
	slog.Debug(fmt.Sprintf("Final tree: Hash - %x, content: %+v", hash, finalTree))
	for _, entry := range finalTree {
		slog.Debug(fmt.Sprintf("%s %s %x %s\n", entry.Mode, utils.ModeStringToKind(entry.Mode), entry.Hash, entry.Name))
	}
	utils.SaveHashedObject(hash, object)

	return hash
}
