package commands

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/codecrafters-io/git-starter-go/pkg/utils"
)

type Edit struct {
	Type string
	Line string
}

const (
	bgGreenLight  = "\033[48;5;22m"
	bgRedLight    = "\033[48;5;52m"
	fgGreenDark   = "\033[48;5;65m"
	fgRedDark     = "\033[48;5;88m"
	reset         = "\033[0m"
)

func Diff(args ...string) {
	indexFile := ReadIndex(args...)

	var noColor bool
	if slices.Contains(args, "--no-color") {
		noColor = true
	}

	shouldColor := !noColor && utils.IsTerminal()

	for _, entry := range indexFile.Entries {
		_, _, workingContent := utils.GetBlobHashObject(entry.Path)

		hash := fmt.Sprintf("%x", entry.SHA1[:])
		blobContent := CatFileReadObject(hash[0:2], hash[2:])

		if !fileExists(entry.Path) {
			continue
		}

		if !bytes.Equal(blobContent, workingContent) {
			PrintDiff(entry.Path, blobContent, workingContent, shouldColor)
		}
	}
}

func PrintDiff(path string, old []byte, new []byte, color bool) {
	oldLines := strings.Split(strings.Join(strings.Split(string(old), "\x00")[1:], "\x00"), "\n")
	newLines := strings.Split(string(new), "\n")

	edits := myersDiff(oldLines, newLines)

	fmt.Printf("diff --ccgit a/%s b/%s\n", path, path)
	fmt.Printf("--- a/%s\n", path)
	fmt.Printf("+++ b/%s\n", path)

	printHunks(edits, color)
}

func printHunks(edits []Edit, color bool) {
	const contextLines = 3

	var aLine, bLine int // linhas atuais em A e B

	for i := 0; i < len(edits); {
		// Pular iguais
		for i < len(edits) && edits[i].Type == "equal" {
			i++
			aLine++
			bLine++
		}
		if i >= len(edits) {
			break
		}

		// Começo do hunk
		start := max(0, i-contextLines)

		// Avançar até o final do hunk (com contexto)
		end := i
		for end < len(edits) {
			if isChange(edits[end]) {
				end += 1
			} else {
				// olha se tem contexto suficiente depois de mudança
				tmp := end
				for tmp < len(edits) && !isChange(edits[tmp]) && tmp-end < contextLines {
					tmp++
				}
				if tmp-end < contextLines {
					end = tmp
					break
				} else {
					end = tmp
				}
			}
		}
		end = min(len(edits), end+contextLines)

		// Calcular intervalo para A e B
		_, lenA := countLines(edits[start:end], "delete", "equal")
		_, lenB := countLines(edits[start:end], "insert", "equal")

		fmt.Printf("@@ -%d,%d +%d,%d @@\n", aLine+1, lenA, bLine+1, lenB)

		// Imprimir o hunk
		for j := start; j < end; j++ {
			line := edits[j]

			if j+1 < len(edits) && line.Type == "delete" && edits[j+1].Type == "insert" {
				oldColored, newColored := HighlightLineDiff(line.Line, edits[j+1].Line, color)
				fmt.Println(oldColored)
				fmt.Println(newColored)
				j++
				aLine++
				bLine++
				continue
			}

			prefix := " "
			colorCode := ""

			switch line.Type {
			case "insert":
				prefix = "+"
				if color {
					colorCode = "\033[32m"
				} else {
					colorCode = ""
				}
			case "delete":
				prefix = "-"
				if color {
					colorCode = "\033[31m"
				} else {
					colorCode = ""
				}
			case "equal":
				prefix = " "
				if color {
					colorCode = "\033[0m"
				} else {
					colorCode = ""
				}
			}

			fmt.Printf("%s%s%s\033[0m\n", colorCode, prefix, line.Line)

			if line.Type != "insert" {
				aLine++
			}
			if line.Type != "delete" {
				bLine++
			}
		}

		i = end
	}
}

func isChange(e Edit) bool {
	return e.Type == "insert" || e.Type == "delete"
}

func countLines(edits []Edit, types ...string) (start int, count int) {
	match := map[string]bool{}
	for _, t := range types {
		match[t] = true
	}
	for _, e := range edits {
		if match[e.Type] {
			count++
		}
	}
	return 0, count
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			fmt.Fprintf(os.Stderr, "Error to add file: %v\n", err)
			os.Exit(1)
		}
	}

	return true
}

func HighlightLineDiff(oldLine, newLine string, enabled bool) (string, string) {
	if !enabled {
		return "- " + oldLine, "+ " + newLine
	}

	oldChars := explodeString(oldLine)
	newChars := explodeString(newLine)

	edits := myersDiff(oldChars, newChars)

	var oldBuilder, newBuilder strings.Builder

	// old line: bg vermelho claro
	oldBuilder.WriteString(bgRedLight + "- ")
	// new line: bg verde claro
	newBuilder.WriteString(bgGreenLight + "+ ")

	for _, edit := range edits {
		switch edit.Type {
		case "equal":
			oldBuilder.WriteString(edit.Line)
			newBuilder.WriteString(edit.Line)
		case "delete":
			oldBuilder.WriteString(fgRedDark + edit.Line + reset + bgRedLight)
		case "insert":
			newBuilder.WriteString(fgGreenDark + edit.Line + reset + bgGreenLight)
		}
	}

	oldBuilder.WriteString(reset)
	newBuilder.WriteString(reset)

	return oldBuilder.String(), newBuilder.String()
}

func explodeString(s string) []string {
	runes := []rune(s)
	out := make([]string, len(runes))
	for i, r := range runes {
		out[i] = string(r)
	}
	return out
}

func myersDiff(oldLines, newLines []string) []Edit {
	N, M := len(oldLines), len(newLines)
	max := N + M
	v := map[int]int{1: 0}
	paths := map[int]map[int]int{}

	var x, y int
	for D := 0; D <= max; D++ {
		paths[D] = map[int]int{}
		for k := -D; k <= D; k += 2 {
			if _, ok := v[k-1]; !ok {
				v[k-1] = -1 << 31
			}
			if _, ok := v[k+1]; !ok {
				v[k+1] = -1 << 31
			}

			if k == -D || (k != D && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}

			y = x - k

			for x < N && y < M && oldLines[x] == newLines[y] {
				x++
				y++
			}

			v[k] = x
			paths[D][k] = x

			if x >= N && y >= M {
				return buildEdits(oldLines, newLines, paths, D)
			}
		}
	}

	return nil
}

func buildEdits(a, b []string, paths map[int]map[int]int, D int) []Edit {
	x, y := len(a), len(b)
	edits := []Edit{}

	for d := D; d > 0; d-- {
		k := x - y
		var prevK int
		if k == -d || (k != d && paths[d-1][k-1] < paths[d-1][k+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := paths[d-1][prevK]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			x--
			y--
			edits = append([]Edit{{Type: "equal", Line: a[x]}}, edits...)
		}

		if x == prevX {
			y--
			edits = append([]Edit{{Type: "insert", Line: b[y]}}, edits...)
		} else {
			x--
			edits = append([]Edit{{Type: "delete", Line: a[x]}}, edits...)
		}
	}

	for x > 0 && y > 0 && a[x-1] == b[y-1] {
		x--
		y--
		edits = append([]Edit{{Type: "equal", Line: a[x]}}, edits...)
	}

	return edits
}
