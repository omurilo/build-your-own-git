package commands

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"log"
	"log/slog"
	"os"
	"sort"
	"syscall"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func UpdateIndex(path string, hash [20]byte) {
	stats, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Error to describe file: %v", err)
	}

	var entries []types.Entry
	index := ReadIndex()

	objBits, permBits := utils.GitModeFromGoMode(stats.Mode())
	mode32 := (objBits << 12) | permBits

	sys := stats.Sys().(*syscall.Stat_t)
	fileEntry := types.Entry{
		SHA1:             hash,
		Mode:             mode32,
		Size:             uint32(stats.Size()),
		ObjectType:       uint16(objBits),
		Perms:            uint16(permBits),
		NameLength:       uint16(len(path)),
		Path:             path,
		Dev:              uint32(sys.Dev),
		Ino:              uint32(sys.Ino),
		CtimeSeconds:     uint32(sys.Ctimespec.Sec),
		CtimeNanoseconds: uint32(sys.Ctimespec.Nsec),
		MtimeSeconds:     uint32(sys.Mtimespec.Sec),
		MtimeNanoseconds: uint32(sys.Mtimespec.Nsec),
		GID:              sys.Gid,
		UID:              sys.Uid,
		Stage:            0,
		AssumeValid:      false,
		ExtendedFlag:     false,
		IntentToAdd:      false,
		Future:           false,
		SkipWorktree:     false,
	}

	entries = append(entries, fileEntry)
	for _, entry := range index.Entries {
		if entry.SHA1 != fileEntry.SHA1 && entry.Path != fileEntry.Path {
			entries = append(entries, entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	indexBuffer := WriteIndex(entries)
	slog.Debug("Index buffer:\n%+v", indexBuffer)
}

func WriteIndex(entries []types.Entry) bytes.Buffer {
	indexFile, err := os.Create(".git/index")
	if err != nil {
		log.Fatalf("Error to describe file: %v", err)
	}
	defer indexFile.Close()

	var indexBuffer bytes.Buffer

	header := []byte{'D', 'I', 'R', 'C'}
	header = append(header, 0x00, 0x00, 0x00, 0x02) // version 2
	header = binary.BigEndian.AppendUint32(header, uint32(len(entries)))
	indexBuffer.Write(header)

	for _, entry := range entries {
		entryBytes := entry.ToBytes()
		indexBuffer.Write(entryBytes)
	}

	sum := sha1.Sum(indexBuffer.Bytes())
	indexBuffer.Write(sum[:])
	_, _ = indexFile.Write(indexBuffer.Bytes())

	return indexBuffer
}
