package generator

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func mkGenState() *genstate {
	return &genstate{
		outdir: "/tmp",
		ipref:  "foo/",
		tag:    "gen",
		numtpk: 1,
		pfuncs: make(map[string]string),
		rfuncs: make(map[string]string),
		gvars:  make(map[string]string),
	}
}

func TestBasic(t *testing.T) {
	rand.Seed(0)
	checkTunables(tunables)
	s := mkGenState()
	for i := 0; i < 1000; i++ {
		fp := s.GenFunc(i, i)
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		s.emitCaller(fp, b, i)
		s.emitChecker(fp, b, i)
	}
	if s.errs != 0 {
		t.Errorf("%d errors during Generate", s.errs)
	}
}

func TestMoreComplicated(t *testing.T) {
	rand.Seed(0)
	saveit := tunables
	defer func() { tunables = saveit }()

	checkTunables(tunables)
	s := mkGenState()
	for i := 0; i < 10000; i++ {
		fp := s.GenFunc(i, i)
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		s.emitCaller(fp, b, i)
		verb(1, "finished iter %d caller", i)
		s.emitChecker(fp, b, i)
		verb(1, "finished iter %d checker", i)
		if s.errs != 0 {
			t.Errorf("%d errors during Generate iter %d", s.errs, i)
		}
	}
}

func TestIsBuildable(t *testing.T) {

	//Verbctl = 4

	td, err := ioutil.TempDir("", "cabi-testgen")
	if err != nil {
		t.Errorf("can't create temp dir")
	}
	defer os.RemoveAll(td)
	//println("=-= td is ", td)

	verb(1, "generating into temp dir %s", td)

	rand.Seed(1)
	checkTunables(tunables)
	pack := filepath.Base(td)
	fcnmask := make(map[int]int)
	errs := Generate("x", td, pack, 10, 10, int64(0), "", fcnmask)
	if errs != 0 {
		t.Errorf("%d errors during Generate", errs)
	}

	verb(1, "building %s\n", td)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = td
	coutput, cerr := cmd.CombinedOutput()
	if cerr != nil {
		t.Errorf("go build command failed: %s\n", string(coutput))
	}
	verb(1, "output is: %s\n", string(coutput))
}
