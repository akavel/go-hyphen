package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/akavel/go-epub/hyphenate"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"unicode"
)

const (
	htmlBase = iota
	htmlTag
	htmlBody
	htmlTagInBody
)

var pattBody = regexp.MustCompile(`^\s*([^:]*:\s*)?\b[Bb][Oo][Dd][Yy]\b`) // start simple
var utf8softhyphen = []byte{0xc2, 0xad}

func hyphHtml(r io.Reader, w io.Writer, h *hyphenate.Hyphenations) error {
	// TODO: skip HTML comments
	// TODO: skip SVG etc.; maybe handle only selected elems?... but html5 maybe allows variety?
	// TODO: in case of problems, try using code.google.com/p/go.net/html parser
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	word := make([]rune, 0, 128)
	state := htmlBase
	for {
		c, sz, err := br.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if c == unicode.ReplacementChar && sz == 1 {
			return errors.New("Invalid UTF-8 code in input")
		}

		switch state {
		case htmlBase:
			if c == '<' {
				state = htmlTag
				word = word[0:0]
			}
			_, err = bw.WriteRune(c)
		case htmlTag:
			if c == '>' {
				if pattBody.MatchString(string(word)) {
					state = htmlBody
				} else {
					state = htmlBase
				}
				word = word[0:0]
			} else {
				word = append(word, c)
			}
			_, err = bw.WriteRune(c)
		case htmlBody:
			if unicode.IsLetter(c) {
				word = append(word, c)
				break
			}
			if c == '<' {
				state = htmlTagInBody
			}
			if len(word) > 0 {
				parts := hyphenate.Word([]byte(string(word)), *h)
				newword := bytes.Join(parts, utf8softhyphen)
				_, err = bw.Write(newword)
				if err != nil {
					return err
				}
				word = word[0:0]
			}
			_, err = bw.WriteRune(c)
		case htmlTagInBody:
			if c == '>' {
				state = htmlBody
			}
			_, err = bw.WriteRune(c)
		}
		if err != nil {
			return err
		}
	}
	panic("not reached")
}

func hyph(epubpath, hyphpath string) error {
	hh, err := os.Open(hyphpath)
	if err != nil {
		return err
	}
	defer hh.Close()

	hyph, err := hyphenate.ParseTexHyph(hh)
	if err != nil {
		return err
	}

	// Open EPUB file as zip archive
	r, err := zip.OpenReader(epubpath)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Ext(f.Name) != ".html" {
			continue
		}
		fmt.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			return err
		}

		err = hyphHtml(rc, os.Stdout, hyph)
		//_, err = io.CopyN(os.Stdout, rc, 68)
		if err != nil && err != io.EOF {
			return err
		}
		rc.Close()
		fmt.Println()
	}

	_ = hyph
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("USAGE: %s EBOOKPATH.epub HYPH.tex\n"+
			"Will create EBOOKPATH.2.epub\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	err := hyph(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

}
