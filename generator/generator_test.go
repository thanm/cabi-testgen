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

func TestExhaustive(t *testing.T) {

	td, err := ioutil.TempDir("", "cabi-testgen")
	if err != nil {
		t.Errorf("can't create temp dir")
	}
	defer os.RemoveAll(td)
	//println("=-= td is ", td)

	verb(1, "generating into temp dir %s", td)

	scenarios := []struct {
		name     string
		adjuster func()
	}{
		{
			"minimal",
			func() {
				tunables.nParmRange = 3
				tunables.nReturnRange = 3
				tunables.structDepth = 1
				tunables.recurPerc = 0
				tunables.methodPerc = 0
				tunables.doReflectCall = false
				tunables.doDefer = false
				tunables.takeAddress = false
				checkTunables(tunables)
			},
		},
		{
			"moreparms",
			func() {
				tunables.nParmRange = 15
				tunables.nReturnRange = 7
				tunables.structDepth = 3
				checkTunables(tunables)
			},
		},
		{
			"addrecur",
			func() {
				tunables.recurPerc = 20
				checkTunables(tunables)
			},
		},
		{
			"addmethod",
			func() {
				tunables.methodPerc = 25
				checkTunables(tunables)
			},
		},
		{
			"addtakeaddr",
			func() {
				tunables.takeAddress = true
				tunables.takenFraction = 20
				checkTunables(tunables)
			},
		},
		{
			"addreflect",
			func() {
				tunables.doReflectCall = true
				checkTunables(tunables)
			},
		},
		{
			"adddefer",
			func() {
				tunables.doDefer = true
				tunables.deferFraction = 30
				checkTunables(tunables)
			},
		},
	}

	// Loop over scenarios and make sure each one works properly.
	for i, s := range scenarios {
		s.adjuster()
		rand.Seed(int64(i + 9))
		os.RemoveAll(td)
		pack := filepath.Base(td)
		fcnmask := make(map[int]int)
		errs := Generate("x", td, pack, 10, 10, int64(0), "", fcnmask)
		if errs != 0 {
			t.Errorf("%d errors during scenarios %q Generate", errs, s.name)
		}

		verb(1, "building %s\n", td)

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = td
		coutput, cerr := cmd.CombinedOutput()
		if cerr != nil {
			t.Errorf("run failed for scenario %q:  %s\n", s.name, string(coutput))
		}
		verb(1, "output is: %s\n", string(coutput))
	}
}
