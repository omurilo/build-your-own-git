package commands

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func Push(args ...string) {
	httpClient := http.DefaultClient

	var basicAuth string
	if slices.Contains(args, "-u") {
		usernamePosition := utils.GetSlicePosition(args, "-u") + 1
		passwordPosition := utils.GetSlicePosition(args, "-p") + 1
		data := fmt.Sprintf("%s:%s", args[usernamePosition], args[passwordPosition])
		basicAuth = base64.StdEncoding.EncodeToString([]byte(data))
	}

	request, err := http.NewRequest("GET", "https://github.com/omurilo/ccgit-test.git/info/refs?service=git-receive-pack", nil)
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

	hash, ref := parseReceivePack(body)

	headHash := utils.GetHeadHash()

	if bytes.Equal(hash, headHash) {
		fmt.Fprintf(os.Stderr, "Your branch is up to date with 'origin/main'\n")
		os.Exit(1)
	}

	refLine := utils.GetUpdateRefLine(hash, headHash, ref)

	fmt.Printf("refLine: %s", refLine)
}

func parseReceivePack(data []byte) ([]byte, string) {
	offset := 0
	lengthStr := string(data[offset : offset+4])
	chunkSize, _ := strconv.ParseInt(lengthStr, 16, 64)
	offset += int(chunkSize) + 4

	// sizeStr := string(data[offset : offset+4])
	// size, _ := strconv.ParseInt(sizeStr, 16, 64)
	offset += 4

	hash := data[offset : offset+40]
	offset += 40

	// consume \s
	offset += 1

	ref := strings.Split(string(data[offset:]), "\x00")

	return hash, ref[0]
}
