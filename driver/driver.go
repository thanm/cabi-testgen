//
// Rudimentary program for examining Android APK files. An APK file
// is basically a ZIP file that contains an Android manifest and a series
// of DEX files, strings, resources, bitmaps, and assorted other items.
// This specific reader looks only at the DEX files, not the other
// bits and pieces.
//
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/thanm/cabi-testgen/generator"
)

var verbflag = flag.Int("v", 0, "Verbose trace output level")
var numitflag = flag.Int("n", 1000, "Number of tests to generate")
var seedflag = flag.Int64("s", 10101, "Random seed")
var tagflag = flag.String("t", "gen", "Prefix name of go files/pkgs to generate")
var outdirflag = flag.String("o", ".", "Output directory for generated files")
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
	fmt.Fprintf(os.Stderr, "usage: apkread [flags] <APK file>\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func openOutputFile(filename string, pk string, imports []string, ipref string) *os.File {
	verb(1, "opening %s", filename)
	outf, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	outf.WriteString(fmt.Sprintf("package %s\n\n", pk))
	for _, imp := range imports {
		outf.WriteString(fmt.Sprintf("import . \"%s%s\"\n", ipref, imp))
	}
	outf.WriteString("\n")
	return outf
}

func emitUtils(outf *os.File) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "func NoteFailure(fidx int, pref string, parmNo int) {\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail on func %%d %%s %%d\\n\", fidx, pref, parmNo)\n")
	fmt.Fprintf(outf, "}\n\n")
}

func emitMain(outf *os.File, numit int) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "func main() {\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"starting main\\n\")\n")
	for i := 0; i < *numitflag; i++ {
		fmt.Fprintf(outf, "  Caller%d()\n", i)
	}
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"finished %d tests\\n\")\n", numit)
	fmt.Fprintf(outf, "}\n")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("cabi-testgen: ")
	flag.Parse()
	generator.Verbctl = *verbflag
	rand.Seed(*seedflag)
	verb(1, "in main")
	verb(1, "seed is %d", *seedflag)
	if flag.NArg() != 0 {
		usage("unknown extra arguments")
	}
	verb(1, "tag is %s", *tagflag)

	var ipref string
	if len(*pkgpathflag) > 0 {
		ipref = *pkgpathflag + "/"
	}

	callerpkg := *tagflag + "Caller"
	checkerpkg := *tagflag + "Checker"
	utilspkg := *tagflag + "Utils"
	mainpkg := *tagflag + "Main"

	os.Mkdir(*outdirflag+"/"+callerpkg, 0777)
	os.Mkdir(*outdirflag+"/"+checkerpkg, 0777)
	os.Mkdir(*outdirflag+"/"+utilspkg, 0777)

	callerfile := *outdirflag + "/" + callerpkg + "/" + callerpkg + ".go"
	checkerfile := *outdirflag + "/" + checkerpkg + "/" + checkerpkg + ".go"
	utilsfile := *outdirflag + "/" + utilspkg + "/" + utilspkg + ".go"
	mainfile := *outdirflag + "/" + mainpkg + ".go"

	calleroutfile := openOutputFile(callerfile, callerpkg,
		[]string{checkerpkg, utilspkg}, ipref)
	checkeroutfile := openOutputFile(checkerfile, checkerpkg,
		[]string{utilspkg}, ipref)
	utilsoutfile := openOutputFile(utilsfile, utilspkg, []string{}, "")
	mainoutfile := openOutputFile(mainfile, "main", []string{callerpkg}, ipref)

	emitUtils(utilsoutfile)
	emitMain(mainoutfile, *numitflag)
	for i := 0; i < *numitflag; i++ {
		generator.Generate(calleroutfile, checkeroutfile, i)
	}

	utilsoutfile.Close()
	calleroutfile.Close()
	checkeroutfile.Close()
	mainoutfile.Close()

	verb(1, "leaving main")
}
