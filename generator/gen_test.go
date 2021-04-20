package generator

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func mkGenState() *genstate {
	return &genstate{
		outdir:      "/tmp",
		ipref:       "foo/",
		tag:         "gen",
		numtpk:      1,
		derefFuncs:  make(map[string]string),
		assignFuncs: make(map[string]string),
		allocFuncs:  make(map[string]string),
		globVars:    make(map[string]string),
	}
}

func TestBasic(t *testing.T) {
	checkTunables(tunables)
	s := mkGenState()
	for i := 0; i < 1000; i++ {
		s.wr = NewWrapRand(int64(i), false)
		fp := s.GenFunc(i, i)
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		wr := NewWrapRand(int64(i), false)
		s.wr = wr
		s.emitCaller(fp, b, i)
		s.wr = NewWrapRand(int64(i), false)
		s.emitChecker(fp, b, i, true)
		wr.Check(s.wr)
	}
	if s.errs != 0 {
		t.Errorf("%d errors during Generate", s.errs)
	}
}

func TestMoreComplicated(t *testing.T) {
	saveit := tunables
	defer func() { tunables = saveit }()

	checkTunables(tunables)
	s := mkGenState()
	for i := 0; i < 10000; i++ {
		s.wr = NewWrapRand(int64(i), false)
		fp := s.GenFunc(i, i)
		var buf bytes.Buffer
		var b *bytes.Buffer = &buf
		wr := NewWrapRand(int64(i), false)
		s.wr = wr
		s.emitCaller(fp, b, i)
		verb(1, "finished iter %d caller", i)
		s.wr = NewWrapRand(int64(i), false)
		s.emitChecker(fp, b, i, true)
		verb(1, "finished iter %d checker", i)
		wr.Check(s.wr)
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

	checkTunables(tunables)
	pack := filepath.Base(td)
	errs := Generate("x", td, pack, 10, 10, int64(0), "", nil, nil, false, 10, false, false)
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
				tunables.pointerMethodCallPerc = 30
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
		t.Logf("running %s\n", s.name)
		s.adjuster()
		os.RemoveAll(td)
		pack := filepath.Base(td)
		errs := Generate("x", td, pack, 10, 10, int64(i+9), "", nil, nil, false, 10, false, false)
		if errs != 0 {
			t.Errorf("%d errors during scenarios %q Generate", errs, s.name)
		}
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = td
		coutput, cerr := cmd.CombinedOutput()
		if cerr != nil {
			t.Fatalf("run failed for scenario %q:  %s\n", s.name, string(coutput))
		}
		verb(1, "output is: %s\n", string(coutput))
	}
}

// To add: random type fractions
