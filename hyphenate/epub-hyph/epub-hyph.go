package main

import (
	"archive/zip"
	"fmt"
	"github.com/akavel/go-epub/hyphenate"
	"io"
	"log"
	"os"
	"path/filepath"
)

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

	// Open a zip archive for reading.
	r, err := zip.OpenReader(epubpath)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.CopyN(os.Stdout, rc, 68)
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
