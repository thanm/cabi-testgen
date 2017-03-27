// Program to generate test files for C ABI testing (insure that the
// compiler is putting things in registers or memory and/or casting
// as appropriate).

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/thanm/cabi-testgen/generator"
)

var verbflag = flag.Int("v", 0, "Verbose trace output level")
var numitflag = flag.Int("n", 1000, "Number of tests to generate")
var seedflag = flag.Int64("s", 10101, "Random seed")
var tagflag = flag.String("t", "gen", "Prefix name of go files/pkgs to generate")
var outdirflag = flag.String("o", "", "Output directory for generated files")
var pkgpathflag = flag.String("p", "", "Base package path for generated files")

func verb(vlevel int, s string, a ...interface{}) {
	if *verbflag >= vlevel {
		fmt.Printf(s, a...)
		fmt.Printf("\n")
	}
}

func usage(msg string) {
	if len(msg) > 0 {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	}
	fmt.Fprintf(os.Stderr, "usage: cabi-testgen [flags]\n\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "Example:\n\n")
	fmt.Fprintf(os.Stderr, "  cabi-testgen -n 500 -s 10101 -o gendir\n\n")
	fmt.Fprintf(os.Stderr, "  \tgenerates Go with 500 test cases into a set of subdirs\n")
	fmt.Fprintf(os.Stderr, "  \tin 'gendir', using random see 10101\n")

	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("cabi-testgen: ")
	flag.Parse()
	generator.Verbctl = *verbflag
	if *outdirflag == "" {
		usage("select an output directory with -o flag")
	}
	verb(1, "in main verblevel=%d", *verbflag)
	verb(1, "seed is %d", *seedflag)
	if flag.NArg() != 0 {
		usage("unknown extra arguments")
	}
	verb(1, "tag is %s", *tagflag)

	verb(1, "starting generation")
	generator.Generate(*tagflag, *outdirflag, *pkgpathflag, *numitflag, *seedflag)
	verb(1, "leaving main")
}