# cabi-testgen

Rudimentary test harness for function signature / ABI testing for Go programs.

Randomly generates Go functions that have interesting signatures (mix of arrays,
scalars, structs) plus code to call the emitted functions with specific
values. The resulting program (when run) executes code to call each test
function and verify that values are being passed/received properly.

This can be used "as is" or by building one half of the test (caller or callee)
with a "good" compiler and the other half with a "suspect" compiler, then test
to see if the suspect compiler is doing things correctly.

## What the generated code looks like

The first generated file is genChecker.go, which contains function that look something
like this (simplified):

```
type StructF4S0 struct {
F0 float64
F1 int16
F2 uint16
}

// 0 returns 2 params
func Test4(p0 int8, p1 StructF4S0)  {
  c0 := int8(-1)
  if p0 != c0 {
    NoteFailure(4, "parm", 0)
  }
  c1 := StructF4S0{float64(2), int16(-3), uint16(4)}
  if p1 != c1 {
    NoteFailure(4, "parm", 1)
  }
  return 
}
```

Here the test generator has randomly selected 0 return values and 2 params, then randomly generated types for the params.

The generator then emits code on the calling side into the file "genCaller.go", which might look like:

```
func Caller4() {
var p0 int8
p0 = int8(-1)
var p1 StructF4S0
p1 = StructF4S0{float64(2), int16(-3), uint16(4)}
// 0 returns 2 params
Test4(p0, p1)
}
```

The generator then emits some utility functions (ex: NoteFailure) and a main routine that cycles through all of the tests. 


## Example usage

To generate a set of source files, you can build and run the test generator as follows. This creates a new directory "cabiTest" within $GOPATH/src containing the generated test files:

```
$ git clone https://github.com/thanm/cabi-testgen
$ cd cabi-testgen/cabi-testgen
$ go build .
$ ./cabi-testgen -n 50 -s 12345 -o /tmp/cabiTest -p cabiTest
$ cd /tmp/cabiTest
$ find . -type f -print
./genCaller/genCaller.go
./genChecker/genChecker.go
./genUtils/genUtils.go
./genMain.go
$
```

You can build and run the generated files in the usual way:

```
$ cd /tmp/cabiTest
$ go build .
$ ./cabiTest
starting main
finished 50 tests
$ go build .
$

```

Let's say that I'm testing a compile change that I think may break parameter passing or returns in some way (for corner cases, etc). I could do this using the following:


```

# Download and build the generater
$ git clone https://github.com/thanm/cabi-testgen
...
$ cd cabi-testgen/cabi-testgen
$ go build .

# Run the generator, writing generated sources to /tmp/xxx. This 
# will emit 1 package with 50 functions, using random seed 12345.
$ ./cabi-testgen -q 1 -n 50 -s 12345 -o /tmp/xxx -p cabiTest
$ ls /tmp/xxx
genCaller0  genChecker0  genMain.go  genUtils  go.mod
$

# Build and run the generated sources.
$ cd /tmp/xxx
$ go run .
starting main
finished 500 tests
$

# Build 'genChecker0' package with suspect compiler option.
$ go build -gcflags=cabiTest/genChecker0=-enable-suspect-feature" .
$ 

# Run
$ ./genMain
starting main
finished 50 tests
$
```

Within the generated code above, a CallerXXX function in package "genCaller0" will invoke "TestXXX" in package "genChecker0"; the code in TestXXX will verify that its parameters have the correct expected values, and then will return a set of known values; back in CallerXXX, the returns will be checked as well.

## Command line options

The program has command line options to control the nature of the code generated, as well as size. Some basic options:

* the "-n" option tells the generator the number of test functions to emit per generated package. Best to keep this number down to something reasonable (1000 or less) so as to keep build times reasonable.

* the "-q" option controls the number of emitted test packages.

* the "-o" option provides that path of a directory into which the generator will emit code

* the "-p" option provides a packagepath prefix to use for the emitted code.

* the "-s" option provides the generator with a seed for its random number generator.

There are also options to tell the generator avoid using specific constructs:

* "-recur=0" tells the generator to avoid emitting recursive calls

* "-takeaddr=0" tells the generator to avoid taking the address of params or returns.

* "-inmax=N" tells the generator to emit at most N input params (by default number of input paramters is randomly chosen between 0 and 15)

* "-outmax=N" tells the generator to emit at most N output params (by default number of output parameters is randomly chosen between 0 and 15)

* "-reflect=0" tells the generator to avoid testing the reflect.Call path for test routines

* "-method=0" tells the generator to avoid emitting or testing methods

* "-pragma=XYZ" tells the generator to tag test routines with the pragma "//go:XYZ"

Run the generator with "-help" for a complete list of options.

## Limitations, future work

Method calls are supported, but only value receivers (not pointer receivers).

No support yet for variadic functions.

The set of generated types is still a bit thin; it doesn't yet include
interfaces, maps or slices.



