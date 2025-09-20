package utils

import (
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

func ReadVarInt(data []byte, offset *int) uint {
	var result uint
	shift := 0
	for {
		b := data[*offset]
		*offset++
		result |= uint(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result
}

func GitModeFromGoMode(mode os.FileMode) (objectTypeBits uint32, permsBits uint32) {
	perms := uint32(mode.Perm())

	var obj uint32
	if mode.IsRegular() {
		obj = 0b1000
	} else if mode&os.ModeSymlink != 0 {
		obj = 0b1010
	} else {
		obj = 0b1110
	}

	return obj, perms
}

func TreeModeString(mode os.FileMode) string {
	if mode.IsDir() {
		return "040000"
	}
	if mode&os.ModeSymlink != 0 {
		return "120000"
	}
	perm := mode.Perm()
	if perm&0111 != 0 {
		return "100755"
	}
	return "100644"
}

func ModeStringToKind(mode string) string {
	switch mode {
	case "040000", "40000":
		return "tree"
	case "100644", "100755", "120000":
		return "blob"
	default:
		return "unknown"
	}
}

func GetHeadBranch() string {
	headFile, _ := os.ReadFile(".git/HEAD")
	ref := []byte("ref: refs/heads/")
	branch := string(headFile[len(ref):])
	branch = strings.Split(branch, "\n")[0]

	return branch
}

func GetHeadHash() []byte {
	branch := GetHeadBranch()
	var headHash []byte
	headFile, err := os.ReadFile(fmt.Sprintf(".git/refs/heads/%s", branch))
	if err != nil {
		packedRefsFile, _ := os.ReadFile(".git/packed-refs")
		i := 0
		for i < len(packedRefsFile) {
			startLine := i
			for i < len(packedRefsFile) && packedRefsFile[i] != '\n' {
				i++
			}
			packedRefLine := packedRefsFile[startLine:i]
			line := strings.Split(string(packedRefLine), " ")
			if strings.Contains(line[1], branch) {
				headHash = append(headHash, packedRefLine[0:20]...)
			}
			i++
		}

		return headHash
	}

	i := 0
	for i < len(headFile) {
		startHash := i
		for i < len(headFile) && headFile[i] != '\n' {
			i++
		}
		headHash = append(headHash, headFile[startHash:i]...)
		i++
	}

	return headHash
}

func GetDirTree(path string, ignores []string, sub bool) ([]string, error) {
	dirTree, _ := os.ReadDir(path)
	var dirNames []string

	for _, dir := range dirTree {
		if slices.Contains(ignores, dir.Name()) || dir.Name() == ".git" {
			continue
		}
		if dir.IsDir() {
			subdirs, _ := GetDirTree(filepath.Join(path, dir.Name(), "/"), ignores, true)
			dirNames = append(dirNames, subdirs...)
		} else if sub {
			dirNames = append(dirNames, filepath.Join(path, dir.Name(), "/"))
		} else {
			dirNames = append(dirNames, dir.Name())
		}
	}

	return dirNames, nil
}

func CheckGitRepo(path string, init bool) error {
	dirs, _ := os.ReadDir(path)

	for _, dir := range dirs {
		if dir.Name() == ".git" && init {
			return fmt.Errorf("Directory already is a git repository")
		}

		if dir.Name() == ".git" {
			return nil
		}
	}

	if init {
		return nil
	}

	return fmt.Errorf("Not is a git repository")
}

func GetBlobHashObject(path string) (h [20]byte, o []byte, c []byte) {
	content, _ := os.ReadFile(path)
	file_len := len(content)
	var object []byte
	object = append(object, fmt.Appendf(nil, "blob %d\x00", file_len)...)
	object = append(object, content...)

	hash := sha1.Sum([]byte(object))
	return hash, object, content
}

func GetTreeHashObject(content []byte) ([20]byte, []byte, []byte) {
	file_len := len(content)
	var object []byte
	object = append(object, fmt.Appendf(nil, "tree %d\x00", file_len)...)
	object = append(object, content...)

	hash := sha1.Sum([]byte(object))
	return hash, object, content
}

func GetCommitHashObject(treeHash [20]byte, messages ...string) ([20]byte, []byte) {
	authorName := "Murilo Alves"
	authorEmail := "hi@omurilo.dev"
	ts := time.Now().Unix()
	_, offset := time.Now().Zone()
	offsetHours := offset / 3600
	offsetMinutes := (offset % 3600) / 60
	tzOffset := fmt.Sprintf("%+03d%02d", offsetHours, int(math.Abs(float64(offsetMinutes))))

	parent := GetHeadHash()

	var body []byte
	body = append(body, fmt.Appendf(nil, "tree %x\n", treeHash)...)
	if parent != nil {
		body = append(body, fmt.Appendf(nil, "parent %s\n", parent)...)
	}
	body = append(body, fmt.Appendf(nil, "author %s <%s> %d %s\n", authorName, authorEmail, ts, tzOffset)...)
	body = append(body, fmt.Appendf(nil, "committer %s <%s> %d %s\n\n", authorName, authorEmail, ts, tzOffset)...)
	for _, message := range messages {
		body = append(body, fmt.Appendf(nil, "%s\n", message)...)
	}

	var object []byte
	object = append(object, fmt.Appendf(nil, "commit %d\x00", len(body))...)
	object = append(object, body...)
	hash := sha1.Sum(object)
	return hash, object
}

func SaveHashedObject(hash [20]byte, object []byte) {
	os.Mkdir(fmt.Sprintf(".git/objects/%x", hash[0:1]), 0755)
	objectFile, err := os.Create(fmt.Sprintf(".git/objects/%x/%x", hash[0:1], hash[1:]))
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		panic(err)
	}
	defer objectFile.Close()

	w := zlib.NewWriter(objectFile)
	_, err = w.Write(object)
	if err != nil {
		panic(err)
	}
	w.Close()
}

func GetSlicePosition[T comparable](slice []T, element T) int {
	pos := -1
	for i, n := range slice {
		if n == element {
			pos = i
		}
	}

	return pos
}

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
