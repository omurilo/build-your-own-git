package commands

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/codecrafters-io/git-starter-go/pkg/types"
	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

func ReadIndex(args ...string) types.IndexFile {
	fileContent, err := os.Open(".git/index")
	if err != nil {
		return types.IndexFile{}
	}
	defer fileContent.Close()

	content, err := io.ReadAll(fileContent)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	if len(content)-20 < 0 {
		return types.IndexFile{}
	}

	offset := 0
	previousPath := ""

	header := content[:12]
	offset += 12

	sign := header[:4]
	version := binary.BigEndian.Uint32(header[4:8])
	entriesLen := binary.BigEndian.Uint32(header[8:])

	var entries []types.Entry
	for range entriesLen {
		entry := readEntry(content, &offset, version, &previousPath)
		entries = append(entries, entry)
	}

	var extensions []types.TreeCacheExtension
	for offset < len(content)-20 {
		sig := string(content[offset : offset+4])
		size := binary.BigEndian.Uint32(content[offset+4 : offset+8])
		offset += 8

		data := content[offset : offset+int(size)]
		offset += int(size)

		if sig == "TREE" {
			treeEntries, err := parseCacheTreeExtension(data)
			if err != nil {
				log.Fatalf("An error ocurred: %v", err)
			}

			extensions = append(extensions, types.TreeCacheExtension{Sig: sig, Entries: treeEntries})
		}
	}

	sum := sha1.Sum(content[:len(content)-20])
	if !bytes.Equal(sum[:], content[len(content)-20:]) {
		log.Println("⚠️ Checksum mismatch!")
	}

	return types.IndexFile{
		Entries:    entries,
		Sign:       sign,
		Version:    version,
		EntriesLen: entriesLen,
		Extensions: extensions,
	}
}

func readEntry(content []byte, offset *int, version uint32, previousPath *string) types.Entry {
	entryStart := *offset
	var path []byte

	ctimeseconds := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	ctimenanoseconds := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	mtimeseconds := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	mtimenanoseconds := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	dev := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	ino := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	mode := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	raw := mode & 0xFFFF

	objtype := (raw >> 12) & 0xF // 4 bits
	_ = (raw >> 9) & 0x7         // 3 bits
	perms := raw & 0x1FF         // 9 bits

	uid := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	gid := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	filesize := binary.BigEndian.Uint32(content[*offset : *offset+4])
	*offset += 4

	objectname := content[*offset : *offset+20] // sha1
	*offset += 20

	if *offset+2 > len(content) {
		log.Fatalf("Out of bounds before flags: offset=%d len(content)=%d", *offset, len(content))
	}
	flags := binary.BigEndian.Uint16(content[*offset : *offset+2])
	*offset += 2

	assumevalid := (flags >> 15) & 0x1
	extendedflag := (flags >> 14) & 0x1
	stage := (flags >> 12) & 0x3
	namelen := flags & 0x0FFF

	var future uint16
	var skipworktree uint16
	var intenttoadd uint16
	if extendedflag == 1 {
		// version 3 or later 16-bit
		raw := binary.BigEndian.Uint16(content[*offset : *offset+2])
		*offset += 2

		future = (raw >> 15) & 0x1
		skipworktree = (raw >> 14) & 0x1
		intenttoadd = (raw >> 13) & 0x1
		_ = raw & 0x1FF
	}

	switch version {
	case 2, 3:
		if namelen < 0x0FFF {
			if int(namelen) > len(content) {
				break
			}
			path = content[*offset : *offset+int(namelen)]
			*offset += int(namelen)
			*offset++
		} else {
			start := *offset
			for content[*offset] != 0 {
				*offset++
			}
			path = content[start:*offset]
			*offset++
		}

		entryEnd := *offset
		padding := (8 - (entryEnd-entryStart)%8) % 8
		*offset += padding
	case 4:
		n := utils.ReadVarInt(content, offset)

		start := *offset
		for content[*offset] != 0 {
			*offset++
		}
		suffix := string(content[start:*offset])
		*offset++

		prefix := (*previousPath)[:len(*previousPath)-int(n)]
		path := prefix + suffix

		*previousPath = path
	}

	return types.Entry{
		CtimeSeconds:     ctimeseconds,
		CtimeNanoseconds: ctimenanoseconds,
		MtimeSeconds:     mtimeseconds,
		MtimeNanoseconds: mtimenanoseconds,
		Dev:              dev,
		Ino:              ino,
		Mode:             mode,
		ObjectType:       uint16(objtype),
		Perms:            uint16(perms),
		UID:              uid,
		GID:              gid,
		Size:             filesize,
		SHA1:             [20]byte(objectname),
		AssumeValid:      assumevalid == 1,
		ExtendedFlag:     extendedflag == 1,
		Stage:            uint8(stage),
		NameLength:       namelen,
		SkipWorktree:     skipworktree == 1,
		IntentToAdd:      intenttoadd == 1,
		Future:           future == 1,
		Path:             string(path),
	}
}

func parseCacheTreeExtension(data []byte) ([]types.TreeCacheEntry, error) {
	var entries []types.TreeCacheEntry
	i := 0
	for i < len(data) {
		startPath := i

		for i < len(data) && data[i] != 0 {
			i++
		}

		if i >= len(data) {
			return nil, fmt.Errorf("no NUL‑terminator in cache tree path component")
		}

		pathComp := string(data[startPath:i])
		i++
		startCount := i
		for i < len(data) && data[i] != ' ' {
			i++
		}
		if i >= len(data) {
			return nil, fmt.Errorf("no space after entry_count in cache tree")
		}
		entryCountStr := string(data[startCount:i])
		entryCount, err := strconv.ParseInt(entryCountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid entry_count: %v", err)
		}
		i++

		startSub := i
		for i < len(data) && data[i] != '\n' {
			i++
		}
		if i >= len(data) {
			return nil, fmt.Errorf("no newline after subtrees count in cache tree")
		}
		subtreeCountStr := string(data[startSub:i])
		subtreeCount, err := strconv.ParseInt(subtreeCountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid subtree_count: %v", err)
		}
		i++

		var oid []byte
		if entryCount >= 0 {
			if i+20 > len(data) {
				return nil, fmt.Errorf("not enough bytes for OID in cache tree")
			}
			oid = data[i : i+20]
			i += 20
		} else {
			// entryCount < 0 → invalidado, sem OID
			oid = nil
		}

		entries = append(entries, types.TreeCacheEntry{
			Path:         pathComp,
			EntryCount:   entryCount,
			SubtreeCount: subtreeCount,
			Oid:          oid,
		})
	}

	return entries, nil
}
