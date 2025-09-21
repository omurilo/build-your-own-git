package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

type TreeCacheExtension struct {
	Sig     string
	Entries []TreeCacheEntry
}

type TreeCacheEntry struct {
	Path         string
	EntryCount   int64
	SubtreeCount int64
	Oid          []byte
}

type TreeEntry struct {
	Mode string
	Name string
	Hash []byte
}

type TreeObject struct {
	Entries []TreeEntry
}

type CommitObject struct {
	Tree *TreeObject
	// Parent *CommitObject
}

type FileInfo struct {
	Path  string
	Stage string
	Color string
}

type IndexFile struct {
	EntriesLen uint32
	Version    uint32
	Sign       []byte
	Entries    []Entry
	Extensions []TreeCacheExtension
}

type Entry struct {
	CtimeSeconds     uint32
	CtimeNanoseconds uint32
	MtimeSeconds     uint32
	MtimeNanoseconds uint32
	Dev              uint32
	Ino              uint32
	Mode             uint32
	UID              uint32
	GID              uint32
	Size             uint32

	ObjectType uint16 // 4 bits dos 16 bits mais baixos de Mode
	Perms      uint16 // 9 bits dos 16 bits mais baixos de Mode
	NameLength uint16

	Stage uint8

	AssumeValid  bool
	ExtendedFlag bool
	SkipWorktree bool
	IntentToAdd  bool
	Future       bool

	SHA1 [20]byte

	Path string
}

type GitObject struct {
	Type string
	Data []byte
}

func (e Entry) ToBytes() []byte {
	var buffer []byte

	stat := make([]byte, 40)
	binary.BigEndian.PutUint32(stat[0:4], e.CtimeSeconds)
	binary.BigEndian.PutUint32(stat[4:8], e.CtimeNanoseconds)
	binary.BigEndian.PutUint32(stat[8:12], e.MtimeSeconds)
	binary.BigEndian.PutUint32(stat[12:16], e.MtimeNanoseconds)
	binary.BigEndian.PutUint32(stat[16:20], e.Dev)
	binary.BigEndian.PutUint32(stat[20:24], e.Ino)

	mode32 := (uint32(e.ObjectType) & 0xF) << 12
	mode32 |= uint32(e.Perms) & 0x1FF

	binary.BigEndian.PutUint32(stat[24:28], mode32)
	binary.BigEndian.PutUint32(stat[28:32], e.UID)
	binary.BigEndian.PutUint32(stat[32:36], e.GID)
	binary.BigEndian.PutUint32(stat[36:40], e.Size)

	buffer = append(buffer, stat...)

	buffer = append(buffer, e.SHA1[:]...)

	var flags uint16 = 0
	nameLen := min(len(e.Path), 0x0FFF)
	flags |= uint16(nameLen) & 0x0FFF

	flagBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(flagBytes, flags)
	buffer = append(buffer, flagBytes...)

	buffer = append(buffer, []byte(e.Path)...)
	buffer = append(buffer, 0x00)

	entryLen := len(buffer)
	padLen := (8 - (entryLen % 8)) % 8
	if padLen > 0 {
		buffer = append(buffer, make([]byte, padLen)...)
	}

	return buffer
}

func (t *TreeObject) ToBytes() []byte {
	var buffer bytes.Buffer
	var body bytes.Buffer

	sort.Slice(t.Entries, func(i, j int) bool {
		return t.Entries[i].Name < t.Entries[j].Name
	})

	sort.Slice(t.Entries, func(i, j int) bool {
		nameI := t.Entries[i].Name
		nameJ := t.Entries[j].Name

		// Git internamente considera `tree` como terminando com /
		if t.Entries[i].Mode == "040000" {
			nameI += "/"
		}
		if t.Entries[j].Mode == "040000" {
			nameJ += "/"
		}

		return nameI < nameJ
	})

	for _, e := range t.Entries {
		body.WriteString(e.Mode)
		body.WriteByte(' ')

		body.WriteString(e.Name)
		body.WriteByte(0)

		body.Write(e.Hash)
	}

	header := fmt.Sprintf("tree %d\x00", body.Len())
	buffer.WriteString(header)
	buffer.Write(body.Bytes())

	return buffer.Bytes()
}
