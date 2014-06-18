package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	
	"github.com/akavel/go-hyphen"
)

func wraperr(prefix string, err error) error {
	return errors.New(prefix + ": " + err.Error())
}

const (
	htmlBase = iota
	htmlTag
	htmlBody
	htmlTagInBody
	htmlEntityInBody
)

var pattBody = regexp.MustCompile(`^\s*([^:]*:\s*)?\b[Bb][Oo][Dd][Yy]\b`) // start simple
var utf8softhyphen = []byte("&shy;")                                      //[]byte{0xc2, 0xad}

func hyphHtml(r io.Reader, w io.Writer, h *hyphen.Hyphenations) error {
	// TODO: skip HTML comments
	// TODO: skip SVG etc.; maybe handle only selected elems?... but html5 maybe allows variety?
	// TODO: in case of problems, try using code.google.com/p/go.net/html parser
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	word := make([]rune, 0, 128)
	entity := make([]rune, 0, 16)
	state := htmlBase
	for {
		c, sz, err := br.ReadRune()
		if err == io.EOF {
			bw.Flush()
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
			if c == '&' {
				state = htmlEntityInBody
				entity = entity[0:0]
				break
			}
			if c == '<' {
				state = htmlTagInBody
			}
			if len(word) > 0 {
				parts := hyphen.Word([]byte(string(word)), *h)
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
		case htmlEntityInBody:
			if c == ';' {
				state = htmlBody
				if string(entity) == "shy" {
					break
				}
				_, err = bw.Write([]byte(string(word)))
				if err != nil {
					return err
				}
				word = word[0:0]
				_, err = bw.Write([]byte("&" + string(entity) + ";"))
				break
			}
			entity = append(entity, c)
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

	hyph, err := hyphen.ParseTexHyph(hh)
	if err != nil {
		return err
	}

	// Open EPUB file as zip archive
	r, err := zip.OpenReader(epubpath)
	if err != nil {
		return wraperr("cannot open input epub", err)
	}
	defer r.Close()

	// create output EPUB
	ext := filepath.Ext(epubpath)
	noext := epubpath[:len(epubpath)-len(ext)]
	fw, err := os.Create(noext + ".1" + ext)
	if err != nil {
		return err
	}
	defer fw.Close()
	w := zip.NewWriter(fw)

	for _, f := range r.File {
		header := f.FileHeader
		header.CRC32 = 0
		zipf, err := w.CreateHeader(&header)
		if err != nil {
			return wraperr("cannot create zip header", err)
		}

		rc, err := f.Open()
		if err != nil {
			return wraperr("cannot open zip subfile", err)
		}

		fmt.Println("writing", f.Name)
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext == ".html" || ext == ".xhtml" {
			err = hyphHtml(rc, zipf, hyph)
			//_, err = io.CopyN(os.Stdout, rc, 68)
			if err != nil && err != io.EOF {
				return wraperr("cannot write hyphenated contents", err)
			}
		} else {
			_, err = io.Copy(zipf, rc)
			if err != nil {
				return wraperr("cannot write subfile in zip", err)
			}
		}
		rc.Close()
		fmt.Println()
	}

	err = w.Close()
	if err != nil {
		return wraperr("cannot close zip", err)
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
