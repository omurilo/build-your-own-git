package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Push(args ...string) {
	httpClient := http.DefaultClient

	if len(args) < 8 {
		fmt.Fprintf(os.Stderr, "usage: ccgit push <remote_url> <remote_branch = main> -u <username> -p <password>\n")
		os.Exit(1)
	}

	var basicAuth string
	if slices.Contains(args, "-u") {
		usernamePosition := utils.GetSlicePosition(args, "-u") + 1
		passwordPosition := utils.GetSlicePosition(args, "-p") + 1
		data := fmt.Sprintf("%s:%s", args[usernamePosition], args[passwordPosition])
		basicAuth = base64.StdEncoding.EncodeToString([]byte(data))
	}

	request, err := http.NewRequest("GET", fmt.Sprintf("%s/info/refs?service=git-receive-pack", args[2]), nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuth))
	request.Header.Set("User-Agent", "curl/7.87.0")
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Connection", "keep-alive")

	res, err := httpClient.Do(request)

	if err != nil {
		log.Fatalf("Error on http request: %v\n", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error %+v", err)
	}

	remoteHash, ref := parseReceivePack(body)

	headHash := utils.GetHeadHash()

	if bytes.Equal(remoteHash, headHash) {
		fmt.Fprintf(os.Stderr, "Your branch is up to date with 'origin/main'\n")
		os.Exit(1)
	}

	var object []byte
	refLine := utils.GetUpdateRefLine(remoteHash, headHash, ref)
	object = append(object, refLine...)
	object = append(object, []byte("0000")...)

	headFile := CatFileReadObject(string(headHash[:2]), string(headHash[2:]))
	nulIndex := bytes.IndexByte(headFile, 0)
	headFileBody := headFile[nulIndex+1:]

	var gitObjs []types.GitObject

	blobEntries := map[string][]byte{}
	commitEntries := map[string][]byte{}
	treeEntries := map[string][]byte{}

	commitObject, parentHash, treeHash, _ := DeserializeCommitObject(headFile)

	treeFile := commitObject.Tree.ToBytes()
	nulIndex = bytes.IndexByte(treeFile, 0)
	treeFileBody := treeFile[nulIndex+1:]

	commitEntries[string(headHash)] = headFileBody

	treeEntries[fmt.Sprintf("%x", treeHash)] = treeFileBody

	walkTreeBlobEntries, walkTreeTreeEntries := walkTreeEntries(commitObject.Tree.Entries)
	maps.Copy(blobEntries, walkTreeBlobEntries)
	maps.Copy(treeEntries, walkTreeTreeEntries)

	if parentHash != nil {
		walkBlobEntries, walkCommitEntries, walkTreeEntries := walkCommitTree(remoteHash, parentHash)
		maps.Copy(blobEntries, walkBlobEntries)
		maps.Copy(commitEntries, walkCommitEntries)
		maps.Copy(treeEntries, walkTreeEntries)
	}

	for h, data := range commitEntries {
		fmt.Printf("c: %s\n", h)
		gitObjs = append(gitObjs, types.GitObject{
			Type: "commit",
			Data: data,
		})
	}

	for h, data := range treeEntries {
		fmt.Printf("t: %s\n", h)
		gitObjs = append(gitObjs, types.GitObject{
			Type: "tree",
			Data: data,
		})
	}

	for h, data := range blobEntries {
		fmt.Printf("b: %s\n", h)
		gitObjs = append(gitObjs, types.GitObject{
			Type: "blob",
			Data: data,
		})
	}

	_, packObj := utils.GetPackObject(gitObjs)

	object = append(object, packObj...)

	request, err = http.NewRequest("POST", fmt.Sprintf("%s/git-receive-pack", args[2]), bytes.NewReader(object))

	if err != nil {
		log.Fatalf("Error to create request: %+v", err)
	}

	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuth))
	request.Header.Set("User-Agent", "curl/7.87.0")
	request.Header.Set("Accept", "application/x-git-receive-pack-result")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Content-Type", "application/x-git-receive-pack-request")

	resp, err := httpClient.Do(request)

	if err != nil {
		log.Fatalf("Error on http request: %v\n", err)
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error: %+v\n", err)
	}

	fmt.Printf("Body: %s\n", string(body))
}

func parseReceivePack(data []byte) ([]byte, string) {
	offset := 0

	lengthStr := string(data[offset : offset+4])
	chunkSize, err := strconv.ParseInt(lengthStr, 16, 64)
	if err != nil || chunkSize == 0 {
		return nil, ""
	}
	offset += 4

	offset += int(chunkSize)

	offset += 4

	if len(data) < offset+40 {
		return nil, ""
	}

	hash := data[offset : offset+40]
	offset++

	rest := string(data[offset:])
	refStrings := strings.Split(strings.SplitN(rest, "\x00", 2)[0], " ")
	ref := refStrings[len(refStrings)-1]

	zeroHash := make([]byte, 40)
	if bytes.Equal(hash, zeroHash) {
		ref = "refs/heads/main"
	}

	fmt.Printf("hash: %s, ref: %s\n", hash, ref)

	return hash, ref
}

func walkTreeEntries(entries []types.TreeEntry) (map[string][]byte, map[string][]byte) {
	blobEntries := map[string][]byte{}
	treeEntries := map[string][]byte{}

	for _, entry := range entries {
		switch entry.Mode {
		case "040000", "40000":
			treeHash := fmt.Sprintf("%x", entry.Hash[:])
			treeFile := CatFileReadObject(treeHash[:2], treeHash[2:])
			nulIndex := bytes.IndexByte(treeFile, 0)
			body := treeFile[nulIndex+1:]
			treeObject, _ := DeserializeTreeObject(treeFile)
			treeEntries[treeHash] = body
			fmt.Println("Tree hash: ", treeHash)
			walkTreeBlobEntries, walkTreeTreeEntries := walkTreeEntries(treeObject.Entries)
			maps.Copy(blobEntries, walkTreeBlobEntries)
			maps.Copy(treeEntries, walkTreeTreeEntries)
		case "100644", "100755", "120000":
			blobHash := fmt.Sprintf("%x", entry.Hash)
			blobObj := CatFileReadObject(blobHash[:2], blobHash[2:])
			nulIndex := bytes.IndexByte(blobObj, 0)
			blobBody := blobObj[nulIndex+1:]

			blobEntries[blobHash] = blobBody
		}
	}

	return blobEntries, treeEntries
}

func walkCommitTree(rHash []byte, pHash []byte) (blob map[string][]byte, commit map[string][]byte, tree map[string][]byte) {
	blobEntries := map[string][]byte{}
	commitEntries := map[string][]byte{}
	treeEntries := map[string][]byte{}

	if !bytes.Equal(pHash, rHash) && !bytes.Equal(pHash, make([]byte, 40)) {
		h := fmt.Sprintf("%x", pHash[:])
		parentFile := CatFileReadObject(h[:2], h[2:])
		nulIndex := bytes.IndexByte(parentFile, 0)
		body := parentFile[nulIndex+1:]
		commitEntries[string(pHash)] = body
		fmt.Println("\nParent file: ", string(pHash))
		parentCommitObject, parentHash, treeHash, err := DeserializeCommitObject(parentFile)

		ptHash := fmt.Sprintf("%x", treeHash[:])
		treeFile := parentCommitObject.Tree.ToBytes()
		nulIndex = bytes.IndexByte(treeFile, 0)
		parentTreeBody := treeFile[nulIndex+1:]
		treeEntries[ptHash] = parentTreeBody

		if err != nil {
			log.Fatalf("Error on deserialize commit object: %+v", err)
		}

		if parentCommitObject != nil {
			walkTreeBlobEntries, walkTreeTreeEntries := walkTreeEntries(parentCommitObject.Tree.Entries)

			maps.Copy(blobEntries, walkTreeBlobEntries)
			maps.Copy(treeEntries, walkTreeTreeEntries)
		}

		if parentHash != nil {
			walkBlobEntries, walkCommitEntries, walkTreeEntries := walkCommitTree(rHash, parentHash)

			maps.Copy(blobEntries, walkBlobEntries)
			maps.Copy(commitEntries, walkCommitEntries)
			maps.Copy(treeEntries, walkTreeEntries)
		}
	}

	return blobEntries, commitEntries, treeEntries
}
