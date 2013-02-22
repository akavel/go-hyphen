// Hyphenation, using Frank Liang's algorithm.
//
// MIT licensed, by Mateusz Czapliński, 2013.
// Based on public domain Python code, by Ned Batchelder, July 2007.
package hyphenate

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type Tree struct {
	Map    map[byte]Tree
	Points []int
}

type Hyphenations struct {
	Exceptions map[string][]int
	Tree
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Given a word, returns a list of pieces, broken at the possible
// hyphenation points. Note: returned pieces may reuse underlying storage
// of the input word.
//
// TODO: use `word []rune', not `word []byte'
// TODO[sometime]: proper case folding for irregular languages, like Turkish etc.
func Word(word []byte, h Hyphenations) [][]byte {
	// Short words aren't hyphenated.
	// TODO: make it in runes? or not?
	if len(word) <= 4 {
		return [][]byte{word}
	}

	// If the word is an exception, get the stored points.
	lower := string(bytes.ToLower(word))
	var points []int
	if exception, ok := h.Exceptions[lower]; ok {
		points = exception
	} else {
		work := make([]byte, 0, len(lower)+2)
		work = append(work, '.')
		work = append(work, lower...)
		work = append(work, '.')

		points = make([]int, len(work)+1)
		for i := range work {
			t := h.Tree
			for _, c := range work[i:] {
				tmp, ok := t.Map[c]
				if !ok {
					break
				}
				t = tmp

				p := t.Points
				if p == nil {
					continue
				}
				for j := range p {
					points[i+j] = max(points[i+j], p[j])
				}
			}
		}

		// No hyphens in the first two chars or the last two.
		points[1], points[2] = 0, 0
		points[len(points)-2], points[len(points)-3] = 0, 0
	}

	// Examine the points to build the pieces list.
	pieces := [][]byte{
		[]byte{},
	}
	points = points[2:]
	for i, c := range word { // TODO: refactor
		pieces[len(pieces)-1] = append(pieces[len(pieces)-1], c)
		if points[i]%2 != 0 {
			pieces = append(pieces, []byte{})
		}
	}
	return pieces
}

const (
	_Nothing = iota
	_Patterns
	_Exceptions
)

func Parse(r io.Reader) (*Hyphenations, error) {
	b := bufio.NewReader(r)
	h := &Hyphenations{
		Exceptions: make(map[string][]int),
	}
	state := _Nothing
	for {
		line, prefix, err := b.ReadLine()
		if err != nil {
			return nil, err
		}
		if prefix {
			return nil, errors.New("Line too long")
		}

		if comment := bytes.IndexByte(line, '%'); comment != -1 {
			line = line[:comment]
		}
		line = bytes.Trim(line, " \t\n\r")
		if len(line) == 0 {
			continue
		}

		switch string(line) {
		case `\patterns{`:
			state = _Patterns
			continue
		case `\hyphenation{`:
			state = _Exceptions
			continue
		case `}`:
			state = _Nothing
			continue
		}

		switch state {
		case _Patterns:

		case _Exceptions:
			points := make([]int, 1, len(line)+2)
			word := make([]byte, 0, len(line))
			for i := 0; i < len(line); i++ {
				if line[i] != '-' {
					points = append(points, 0)
				} else {
					i++
					points = append(points, 1)
				}
				word = append(word, line[i])
			}
			h.Exceptions[string(word)] = points
		}
	}
	return h, nil
}
