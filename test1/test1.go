package main

import (
	"fmt"
	"github.com/akavel/go-epub/hyphenate"
	"os"
)

var fhyph = `c:\prog\go-epub\_research\hyph-pl.tex`

func main() {
	f, err := os.Open(fhyph)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	h, err := hyphenate.ParseTexHyph(f)
	if err != nil {
		panic(err)
	}

	for _, c := range []string{
		// exceptions
		"bynajmniej",
		"skądinąd",
		"przynajmniej",
		// regular words
		"kolejka",
		"żółw",
		"żagiew",
		"Józefitów",
		"ten",
		"żąłżem",
	} {
		x := hyphenate.Word([]byte(c), *h)
		for _, part := range x {
			fmt.Print(string(part), " ")
		}
		fmt.Println()
	}
}
