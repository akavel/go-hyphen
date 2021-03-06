// Hyphenation, using Frank Liang's algorithm.
//
// MIT licensed, by Mateusz Czapliński, 2013.
// Based on public domain Python code, by Ned Batchelder, July 2007.
package hyphen

import (
	"bufio"
	"bytes"
	"errors"
	//"fmt" // DEBUG
	"io"
)

type Tree struct {
	Map    map[byte]*Tree
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
			t := &h.Tree
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
	pieces := [][]byte{}
	points = points[2:]
	points[len(word)-1] = -1 // loop terminator
	last := 0
	for i := range word {
		if points[i]%2 == 0 {
			continue
		}
		pieces = append(pieces, word[last:i+1])
		last = i + 1
	}
	return pieces
}

const (
	_Nothing = iota
	_Patterns
	_Exceptions
)

// Parse TeX hyphenation patterns file, as published on http://tug.org/tex-hyphen/
func ParseTexHyph(r io.Reader) (*Hyphenations, error) {
	b := bufio.NewReader(r)
	h := &Hyphenations{
		Tree:       Tree{Map: make(map[byte]*Tree)},
		Exceptions: make(map[string][]int),
	}
	state := _Nothing
	for {
		line, prefix, err := b.ReadLine()
		if state == _Nothing && err == io.EOF {
			return h, nil
		}
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
			// Convert the a pattern like 'a1bc3d4' into a string of chars 'abcd'
			// and a list of points [0, 1, 0, 3, 4].
			//
			// Insert the pattern into the tree.  Each character finds a dict
			// another level down in the tree, and leaf nodes have the list of
			// points.
			t := &h.Tree
			points := []int{}
			p := 0
			for _, c := range line {
				if '0' <= c && c <= '9' {
					p = int(c - '0') // TODO: can these be multidigit? if yes, oops
					continue
				}
				points = append(points, p)
				p = 0

				_, ok := t.Map[c]
				if !ok {
					t.Map[c] = &Tree{Map: make(map[byte]*Tree)}
				}
				t = t.Map[c]
			}
			points = append(points, p)
			t.Points = points
			/*
				chars = re.sub('[0-9]', '', pattern)
				points = [ int(d or 0) for d in re.split("[.a-z]", pattern) ]

				# Insert the pattern into the tree.  Each character finds a dict
				# another level down in the tree, and leaf nodes have the list of
				# points.
				t = self.tree
				for c in chars:
					if c not in t:
						t[c] = {}
					t = t[c]
				t[None] = points
			*/
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
			points = append(points, 0)
			h.Exceptions[string(word)] = points
			//fmt.Println(string(word), points) // DEBUG
		}
	}
	panic("not reached")
}
