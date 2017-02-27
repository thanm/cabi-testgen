package generator

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
)

var Verbctl int = 0

func verb(vlevel int, s string, a ...interface{}) {
	if Verbctl >= vlevel {
		fmt.Printf(s, a...)
		fmt.Printf("\n")
	}
}

type parm interface {
	Declare(outf *os.File, prefix string, suffix string, parmNo int)
	CallerSetInitValue(outf *os.File, parmNo int, value int) int
	GenValue(value int) (string, int)
	//CheckerParam()
	//CheckerCheckValue(parmNo int, value int)
}

type numparm struct {
	tag         string
	widthInBits int
}

func (p numparm) Declare(outf *os.File, prefix string, suffix string, parmNo int) {
	fmt.Fprintf(outf, "%s%d %s%d%s", prefix, parmNo, p.tag, p.widthInBits, suffix)
}

func (p numparm) GenValue(value int) (string, int) {
	var s string
	if p.tag == "complex" {
		s = fmt.Sprintf("complex(%d, %d)", value, value)
	} else {
		s = fmt.Sprintf("%s%d(%d)", p.tag, p.widthInBits, value)
	}
	return s, value + 1
}

func (p numparm) CallerSetInitValue(outf *os.File, parmNo int, value int) int {
	valstr, value := p.GenValue(value)
	fmt.Fprintf(outf, "p%d = %s\n", parmNo, valstr)
	return value
}

type structparm struct {
	sname  string
	fields []parm
}

func (p structparm) Declare(outf *os.File, prefix string, suffix string, parmNo int) {
	fmt.Fprintf(outf, "%s%d %s%s", prefix, parmNo, p.sname, suffix)
}

func writeCom(outf *os.File, i int) {
	if i != 0 {
		fmt.Fprintf(outf, ", ")
	}
}

func bufCom(buf *bytes.Buffer, i int) {
	if i != 0 {
		buf.WriteString(", ")
	}
}

func (p structparm) GenValue(value int) (string, int) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s{", p.sname))
	for fi, f := range p.fields {
		var valstr string
		valstr, value = f.GenValue(value)
		bufCom(&buf, fi)
		buf.WriteString(valstr)
	}
	buf.WriteString("}")
	return buf.String(), value
}

func (p structparm) CallerSetInitValue(outf *os.File, parmNo int, value int) int {
	valstr, value := p.GenValue(value)
	fmt.Fprintf(outf, "p%d = %s\n", parmNo, valstr)
	return value
}

type funcdef struct {
	idx        int
	structdefs []*structparm
	params     []parm
	returns    []parm
}

func intBits() int {
	which := rand.Intn(10)
	switch which {
	case 0, 1, 2, 3:
		return 8
	case 4, 5:
		return 16
	case 6, 7:
		return 32
	case 8, 9, 10:
		return 64
	}
	return 0
}

func floatBits() int {
	which := rand.Intn(10)
	if which > 5 {
		return 32
	}
	return 64
}

func GenParm(f *funcdef, depth int) parm {
	which := rand.Intn(20)
	if depth == 0 && which < 5 {
		sp := new(structparm)
		ns := len(f.structdefs)
		sp.sname = fmt.Sprintf("StructF%dS%d", f.idx, ns)
		f.structdefs = append(f.structdefs, sp)
		nf := rand.Intn(7)
		for fi := 0; fi < nf; fi++ {
			sp.fields = append(sp.fields, GenParm(f, depth+1))
		}
		return sp
	}
	if which < 5 {
		var ip numparm
		ip.tag = "int"
		ip.widthInBits = intBits()
		return ip
	}
	if which < 10 {
		var ip numparm
		ip.tag = "uint"
		ip.widthInBits = intBits()
		return ip
	}
	var fp numparm
	fp.tag = "float"
	fp.widthInBits = floatBits()
	return fp
}

func GenReturn(f *funcdef, depth int) parm {
	return GenParm(f, depth)
}

func GenFunc(fidx int) funcdef {
	var f funcdef
	f.idx = fidx
	numParams := rand.Intn(4)
	numReturns := rand.Intn(2)
	for pi := 0; pi < numParams; pi++ {
		f.params = append(f.params, GenParm(&f, 0))
	}
	for ri := 0; ri < numReturns; ri++ {
		f.returns = append(f.returns, GenReturn(&f, 0))
	}
	return f
}

func emitStructDefs(f *funcdef, outf *os.File) {
	for _, s := range f.structdefs {
		fmt.Fprintf(outf, "type %s struct {\n", s.sname)
		for fi, sp := range s.fields {
			sp.Declare(outf, "F", "\n", fi)
		}
		fmt.Fprintf(outf, "}\n\n")
	}
}

func emitCaller(f *funcdef, outf *os.File) {

	fmt.Fprintf(outf, "func Caller%d() {\n", f.idx)
	var value int = 1
	for pi, p := range f.params {
		p.Declare(outf, "var p", "\n", pi)
		value = p.CallerSetInitValue(outf, pi, value)
	}

	// calling code
	fmt.Fprintf(outf, "// %d returns %d params\n",
		len(f.returns), len(f.params))
	for ri, _ := range f.returns {
		writeCom(outf, ri)
		fmt.Fprintf(outf, "r%d", ri)
	}
	if len(f.returns) > 0 {
		fmt.Fprintf(outf, " := ")
	}
	fmt.Fprintf(outf, "Test%d(", f.idx)
	for pi, _ := range f.params {
		writeCom(outf, pi)
		fmt.Fprintf(outf, "p%d", pi)
	}
	fmt.Fprintf(outf, ")\n")

	// checking values returned
	value = 1
	for ri, r := range f.returns {
		var valstr string
		valstr, value = r.GenValue(value)
		fmt.Fprintf(outf, "  c%d := %s\n", ri, valstr)
		fmt.Fprintf(outf, "  if r%d != c%d {\n", ri, ri)
		fmt.Fprintf(outf, "    NoteFailure(%d, \"return\", %d)\n", f.idx, ri)
		fmt.Fprintf(outf, "  }\n")
	}

	fmt.Fprintf(outf, "}\n\n")
}

func emitChecker(f *funcdef, outf *os.File) {
	emitStructDefs(f, outf)
	fmt.Fprintf(outf, "// %d returns %d params\n", len(f.returns), len(f.params))
	fmt.Fprintf(outf, "func Test%d(", f.idx)

	// params
	for pi, p := range f.params {
		writeCom(outf, pi)
		p.Declare(outf, "p", "", pi)
	}
	fmt.Fprintf(outf, ") ")

	// returns
	if len(f.returns) > 0 {
		fmt.Fprintf(outf, "(")
	}
	for ri, r := range f.returns {
		writeCom(outf, ri)
		r.Declare(outf, "r", "", ri)
	}
	if len(f.returns) > 0 {
		fmt.Fprintf(outf, ")")
	}
	fmt.Fprintf(outf, " {\n")

	// checking code
	value := 1
	for pi, p := range f.params {
		var valstr string
		valstr, value = p.GenValue(value)
		fmt.Fprintf(outf, "  c%d := %s\n", pi, valstr)
		fmt.Fprintf(outf, "  if p%d != c%d {\n", pi, pi)
		fmt.Fprintf(outf, "    NoteFailure(%d, \"parm\", %d)\n", f.idx, pi)
		fmt.Fprintf(outf, "  }\n")
	}

	// returning code
	fmt.Fprintf(outf, "  return ")
	if len(f.returns) > 0 {
		value := 1
		for ri, r := range f.returns {
			var valstr string
			writeCom(outf, ri)
			valstr, value = r.GenValue(value)
			fmt.Fprintf(outf, "%s", valstr)
		}
	}
	fmt.Fprintf(outf, "\n")
	fmt.Fprintf(outf, "}\n\n")
}

func Generate(calloutfile *os.File, checkoutfile *os.File, fidx int) {
	// Generate a function with a random number of params and returns
	f := GenFunc(fidx)
	var fp *funcdef = &f

	// Emit caller side
	emitCaller(fp, calloutfile)

	// Emit checker side
	emitChecker(fp, checkoutfile)
}
