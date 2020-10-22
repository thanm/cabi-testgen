package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBasic(t *testing.T) {
	rand.Seed(0)
	checkTunables(tunables)
	for i := 0; i < 1000; i++ {
		f := GenFunc(i)
		var fp *funcdef = &f
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		emitCaller(fp, b)
		emitChecker(fp, b)
	}
}

func TestMoreComplicated(t *testing.T) {
	rand.Seed(0)
	saveit := tunables
	defer func() { tunables = saveit }()

	// Turn on complex, nested structs
	tunables.structDepth = 3
	tunables.typeFractions[0] -= 10
	tunables.typeFractions[4] += 10

	// Verbctl = 5

	checkTunables(tunables)
	for i := 0; i < 10000; i++ {
		f := GenFunc(i)
		var fp *funcdef = &f
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		emitCaller(fp, b)
		verb(1, "finished iter %d caller", i)
		emitChecker(fp, b)
		verb(1, "finished iter %d checker", i)
	}
}

func TestIsBuildable(t *testing.T) {
	// Verbctl = 2

	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		t.Errorf("something very screwy going on here, no GOPATH")
	}

	gosrc := fmt.Sprintf("%s/src", gopath)
	td, err := ioutil.TempDir(gosrc, "cabi-testgen")
	if err != nil {
		t.Errorf("can't create temp dir")
	}
	defer os.RemoveAll(td)

	verb(1, "generating into temp dir %s", td)

	rand.Seed(1)
	checkTunables(tunables)
	pack := filepath.Base(td)
	fcnmask := make(map[int]int)
	Generate("x", td, pack, 10, int64(0), fcnmask)

	verb(1, "building %s\n", td)

	mainfile := fmt.Sprintf("%s/xMain.go", td)
	cmd := exec.Command("go", "run", mainfile)
	coutput, cerr := cmd.CombinedOutput()
	if cerr != nil {
		t.Errorf("go build command failed: %s\n", string(coutput))
	}
	verb(1, "output is: %s\n", string(coutput))
}
