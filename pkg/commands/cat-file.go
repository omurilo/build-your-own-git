package commands

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func CatFile(writer io.Writer, args ...string) {
	err := utils.CheckGitRepo(".", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	if !slices.Contains(args, "-p") {
		fmt.Fprintf(os.Stderr, "usage: ccgit cat-file -p <hash> [<args>...]\n")
		os.Exit(1)
	}

	pos := utils.GetSlicePosition(args, "-p")
	hash := args[pos+1]
	folder, file := hash[0:2], hash[2:]

	decompressedData := CatFileReadObject(folder, file)

	kind := CatFileExtractKind(decompressedData)

	if slices.Contains(args, "-t") {
		fmt.Println(kind)
		return
	}

	switch kind {
	case "blob":
		fmt.Fprintf(writer, "%+v", strings.Join(strings.Split(string(decompressedData), "\x00")[1:], "\x00"))
	case "tree":
		treeObject, err := DeserializeTreeObject(decompressedData)
		if err != nil {
			log.Fatalf("An error ocurred on read file, %v", err)
		}
		slog.Debug(fmt.Sprintf("data: %+v\n", treeObject))
		for _, entry := range treeObject.Entries {
			fmt.Fprintf(writer, "%s %s %x %s\n", entry.Mode, utils.ModeStringToKind(entry.Mode), entry.Hash, entry.Name)
		}
	case "commit":
		fmt.Fprintf(writer, "%+v", strings.Join(strings.Split(string(decompressedData), "\x00")[1:], "\x00"))
	}
}

func CatFileExtractKind(decompressedData []byte) string {
	var kind string
	i := 0
	for i < len(decompressedData) {
		if kind != "" {
			break
		}
		startKind := i
		for i < len(decompressedData) && decompressedData[i] != ' ' {
			i++
		}

		kind = string(decompressedData[startKind:i])
		i++
	}
	return kind
}

func CatFileReadObject(folder string, file string) []byte {
	fileContent, err := os.Open(fmt.Sprintf(".git/objects/%s/%s", folder, file))
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}
	defer fileContent.Close()

	r, err := zlib.NewReader(fileContent)
	if err != nil {
		panic(err)
	}
	decompressedData, err := io.ReadAll(r)
	if err != nil {
		log.Fatalf("Failed to read index: %v", err)
	}
	r.Close()
	return decompressedData
}

func DeserializeTreeObject(data []byte) (*types.TreeObject, error) {
	nulIndex := bytes.IndexByte(data, 0)
	if nulIndex < 0 {
		return nil, fmt.Errorf("formato inválido: header sem NUL")
	}

	header := string(data[:nulIndex])
	body := data[nulIndex+1:]

	var size int
	_, err := fmt.Sscanf(header, "tree %d", &size)
	if err != nil {
		return nil, fmt.Errorf("Tree file is corrupted")
	}

	entries := []types.TreeEntry{}

	i := 0
	for i < len(body) {
		startMode := i
		for i < len(body) && body[i] != ' ' {
			i++
		}
		if i >= len(body) {
			return nil, fmt.Errorf("formado: sem espaço depois do modo")
		}

		mode := body[startMode:i]
		i++

		startFilename := i
		for i < len(body) && body[i] != 0 {
			i++
		}
		if i >= len(body) {
			return nil, fmt.Errorf("malformado: sem NUL depois do nome")
		}
		fileName := body[startFilename:i]
		i++

		if i+20 > len(body) {
			return nil, fmt.Errorf("malformado: sem bytes suficientes para OID")
		}
		hash := make([]byte, 20)
		copy(hash, body[i:i+20])
		i += 20

		entries = append(entries, types.TreeEntry{
			Mode: string(mode),
			Name: string(fileName),
			Hash: hash,
		})
	}

	return &types.TreeObject{Entries: entries}, nil
}
