// Program to generate test files for C ABI testing (insure that the
// compiler is putting things in registers or memory and/or casting
// as appropriate).

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/thanm/cabi-testgen/generator"
)

var verbflag = flag.Int("v", 0, "Verbose trace output level")
var numitflag = flag.Int("n", 1000, "Number of tests to generate")
var seedflag = flag.Int64("s", 10101, "Random seed")
var tagflag = flag.String("t", "gen", "Prefix name of go files/pkgs to generate")
var outdirflag = flag.String("o", "", "Output directory for generated files")
var pkgpathflag = flag.String("p", "gen", "Base package path for generated files")
var numtpkflag = flag.Int("q", 1, "Number of test packages")
var fcnmaskflag = flag.String("M", "", "Mask containing list of fcn numbers to emit")
var pkmaskflag = flag.String("P", "", "Mask containing list of pkg numbers to emit")

var reflectflag = flag.Bool("reflect", true, "Include testing of reflect.Call.")
var deferflag = flag.Bool("defer", true, "Include testing of defer stmts.")
var recurflag = flag.Bool("recur", true, "Include testing of recursive calls.")
var takeaddrflag = flag.Bool("takeaddr", true, "Include functions that take the address of their parameters and results.")
var methodflag = flag.Bool("method", true, "Include testing of method calls.")
var inlimitflag = flag.Int("inmax", -1, "Max number of input params.")
var outlimitflag = flag.Int("outmax", -1, "Max number of input params.")
var pragmaflag = flag.String("pragma", "", "Tag generated test routines with pragma //go:<value>.")
var maxfailflag = flag.Int("maxfail", 10, "Maximum runtime failures before test self-terminates")
var stackforceflag = flag.Bool("forcestackgrowth", false, "Use hooks to force stack growth.")

// for testcase minimization
var utilsinlineflag = flag.Bool("inlutils", false, "Emit inline utils code (for minimization)")

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

func setupTunables() {
	tunables := generator.DefaultTunables()
	if !*reflectflag {
		tunables.DisableReflectionCalls()
	}
	if !*deferflag {
		tunables.DisableDefer()
	}
	if !*recurflag {
		tunables.DisableRecursiveCalls()
	}
	if !*takeaddrflag {
		tunables.DisableTakeAddr()
	}
	if !*methodflag {
		tunables.DisableMethodCalls()
	}
	if *inlimitflag != -1 {
		tunables.LimitInputs(*inlimitflag)
	}
	if *outlimitflag != -1 {
		tunables.LimitOutputs(*outlimitflag)
	}
	generator.SetTunables(tunables)
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
	rand.Seed(*seedflag)
	if flag.NArg() != 0 {
		usage("unknown extra arguments")
	}
	verb(1, "tag is %s", *tagflag)

	mkmask := func(arg string, tag string) map[int]int {
		if arg == "" {
			return nil
		}
		verb(1, "%s mask is %s", tag, arg)
		m := make(map[int]int)
		ss := strings.Split(arg, ":")
		for _, s := range ss {
			if strings.Contains(s, "-") {
				rng := strings.Split(s, "-")
				if len(rng) != 2 {
					verb(0, "malformed range %s in %s mask, ignoring", s, tag)
					continue
				}
				if i, err := strconv.Atoi(rng[0]); err == nil {
					if j, err := strconv.Atoi(rng[1]); err == nil {
						for k := i; k < j; k++ {
							m[k] = 1
						}
					}
				}
			} else {
				if i, err := strconv.Atoi(s); err == nil {
					m[i] = 1
				}
			}
		}
		return m
	}
	fcnmask := mkmask(*fcnmaskflag, "fcn")
	pkmask := mkmask(*pkmaskflag, "pkg")

	verb(2, "pkg mask is %v", pkmask)
	verb(2, "fn mask is %v", fcnmask)

	verb(1, "starting generation")
	setupTunables()
	errs := generator.Generate(*tagflag, *outdirflag, *pkgpathflag,
		*numitflag, *numtpkflag, *seedflag, *pragmaflag,
		fcnmask, pkmask, *utilsinlineflag, *maxfailflag, *stackforceflag)
	if errs != 0 {
		log.Fatal("errors during generation")
	}
	verb(0, "... files written to directory %s", *outdirflag)
	verb(1, "leaving main")
}
