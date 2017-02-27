package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
)

type TunableParams struct {
	// between 0 and N params
	nParmRange uint8

	// between 0 and N returns
	nReturnRange uint8

	// structs have between 0 and N members
	nStructFields uint8

	// arrays have between 0 and N elements
	nArrayElements uint8

	// Controls how often ints wind up as 8/16/32/64, should
	// add up to 100. Ex: 100 0 0 0 means all ints are 8 bit,
	// 25 25 25 25 means equal likelihood of all types.
	intBitRanges [4]uint8

	// Similar to the above but for 32/64 float types
	floatBitRanges [2]uint8

	// Similar to the above but for unsigned, signed ints.
	unsignedRanges [2]uint8

	// How deeply structs are allowed to be nested.
	structDepth uint8

	// Fraction of param and return types assigned to each
	// category: struct/array/int/float/pointer at the top
	// level. If nesting precludes using a struct, other types
	// are chosen from instead according to same proportions.
	typeFractions [5]uint8
}

var tunables = TunableParams{
	nParmRange:     20,
	nReturnRange:   10,
	nStructFields:  5,
	nArrayElements: 5,
	intBitRanges:   [4]uint8{30, 20, 20, 30},
	floatBitRanges: [2]uint8{50, 50},
	unsignedRanges: [2]uint8{50, 50},
	structDepth:    2,
	typeFractions:  [5]uint8{35, 15, 30, 20, 0},
}

func DefaultTunables() TunableParams {
	return tunables
}

func checkTunables(t TunableParams) {
	var s int = 0

	for _, v := range t.intBitRanges {
		s += int(v)
	}
	if s != 100 {
		log.Fatal(errors.New("intBitRanges tunable does not sum to 100"))
	}

	s = 0
	for _, v := range t.unsignedRanges {
		s += int(v)
	}
	if s != 100 {
		log.Fatal(errors.New("unsignedRanges tunable does not sum to 100"))
	}

	s = 0
	for _, v := range t.floatBitRanges {
		s += int(v)
	}
	if s != 100 {
		log.Fatal(errors.New("floatBitRanges tunable does not sum to 100"))
	}

	s = 0
	for _, v := range t.typeFractions {
		s += int(v)
	}
	if s != 100 {
		log.Fatal(errors.New("typeFractions tunable does not sum to 100"))
	}
}

func SetTunables(t TunableParams) {
	checkTunables(t)
	tunables = t
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

var Verbctl int = 0

func verb(vlevel int, s string, a ...interface{}) {
	if Verbctl >= vlevel {
		fmt.Printf(s, a...)
		fmt.Printf("\n")
	}
}

type parm interface {
	TypeName() string
	Declare(outf *os.File, prefix string, suffix string, parmNo int)
	GenValue(value int) (string, int)
}

type numparm struct {
	tag         string
	widthInBits int
}

func (p numparm) TypeName() string {
	return fmt.Sprintf("%s%d", p.tag, p.widthInBits)
}

func (p numparm) Declare(outf *os.File, prefix string, suffix string, parmNo int) {
	fmt.Fprintf(outf, "%s%d %s%d%s", prefix, parmNo, p.tag, p.widthInBits, suffix)
}

func (p numparm) GenValue(value int) (string, int) {
	var s string
	v := value
	if p.tag == "int" && v%2 != 0 {
		v = -v
	}
	if p.tag == "complex" {
		s = fmt.Sprintf("complex(%d, %d)", value, value)
	} else {
		s = fmt.Sprintf("%s%d(%d)", p.tag, p.widthInBits, v)
	}
	return s, value + 1
}

type structparm struct {
	sname  string
	fields []parm
}

func (p structparm) TypeName() string {
	return p.sname
}

func (p structparm) Declare(outf *os.File, prefix string, suffix string, parmNo int) {
	fmt.Fprintf(outf, "%s%d %s%s", prefix, parmNo, p.sname, suffix)
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

type arrayparm struct {
	aname     string
	nelements uint8
	eltype    parm
}

func (p arrayparm) TypeName() string {
	return p.aname
}

func (p arrayparm) Declare(outf *os.File, prefix string, suffix string, parmNo int) {

	fmt.Fprintf(outf, "%s%d %s%s", prefix, parmNo, p.aname, suffix)
}

func (p arrayparm) GenValue(value int) (string, int) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s{", p.aname))
	for i := 0; i < int(p.nelements); i++ {
		var valstr string
		valstr, value = p.eltype.GenValue(value)
		bufCom(&buf, i)
		buf.WriteString(valstr)
	}
	buf.WriteString("}")
	return buf.String(), value
}

type funcdef struct {
	idx        int
	structdefs []*structparm
	arraydefs  []*arrayparm
	params     []parm
	returns    []parm
}

func intFlavor() string {
	which := uint8(rand.Intn(100))
	if which < tunables.unsignedRanges[0] {
		return "uint"
	}
	return "int"
}

func intBits() int {
	which := uint8(rand.Intn(100))
	var t uint8 = 0
	var bits int = 8
	for _, v := range tunables.intBitRanges {
		t += v
		if which < t {
			return bits
		}
		bits *= 2
	}
	return int(tunables.intBitRanges[3])
}

func floatBits() int {
	which := uint8(rand.Intn(100))
	if which < tunables.floatBitRanges[0] {
		return 32
	}
	return 64
}

func GenParm(f *funcdef, depth int) parm {

	// Enforcement for struct or array nesting depth
	tf := tunables.typeFractions
	amt := 100
	off := uint8(0)
	toodeep := depth >= int(tunables.structDepth)
	if toodeep {
		off = tf[0] + tf[1]
		amt -= int(off)
		tf[0] = 0
		tf[1] = 0
	}
	s := uint8(0)
	for i := 0; i < len(tf); i++ {
		tf[i] += s
		s += tf[i]
	}
	for i := 2; i < len(tf); i++ {
		tf[i] += off
	}

	// Make adjusted selection
	which := uint8(rand.Intn(amt)) + off
	switch {
	case which < tf[0]:
		{
			if toodeep {
				panic("should not be here")
			}
			sp := new(structparm)
			ns := len(f.structdefs)
			sp.sname = fmt.Sprintf("StructF%dS%d", f.idx, ns)
			f.structdefs = append(f.structdefs, sp)
			nf := rand.Intn(int(tunables.nStructFields))
			for fi := 0; fi < nf; fi++ {
				sp.fields = append(sp.fields, GenParm(f, depth+1))
			}
			return sp
		}
	case which < tf[1]:
		{
			ap := new(arrayparm)
			ns := len(f.arraydefs)
			ap.aname = fmt.Sprintf("ArrayF%dS%d", f.idx, ns)
			f.arraydefs = append(f.arraydefs, ap)
			ap.nelements = uint8(rand.Intn(int(tunables.nArrayElements)))
			ap.eltype = GenParm(f, depth+1)
			return ap
		}
	case which < tf[2]:
		{
			var ip numparm
			ip.tag = intFlavor()
			ip.widthInBits = intBits()
			return ip
		}
	case which < tf[3]:
		{
			var fp numparm
			fp.tag = "float"
			fp.widthInBits = floatBits()
			return fp
		}
	case which < tf[4]:
		{
			panic("pointers not yet implemented")
		}
	}

	// fallback
	var ip numparm
	ip.tag = "uint"
	ip.widthInBits = 8
	return ip
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

func emitStructAndArrayDefs(f *funcdef, outf *os.File) {
	for _, s := range f.structdefs {
		fmt.Fprintf(outf, "type %s struct {\n", s.sname)
		for fi, sp := range s.fields {
			sp.Declare(outf, "F", "\n", fi)
		}
		fmt.Fprintf(outf, "}\n\n")
	}
	for _, a := range f.arraydefs {
		fmt.Fprintf(outf, "type %s [%d]%s\n\n", a.aname,
			a.nelements, a.eltype.TypeName())
	}
}

func emitCaller(f *funcdef, outf *os.File) {

	fmt.Fprintf(outf, "func Caller%d() {\n", f.idx)
	var value int = 1
	for pi, p := range f.params {
		p.Declare(outf, "var p", "\n", pi)
		var valstr string
		valstr, value = p.GenValue(value)
		fmt.Fprintf(outf, "p%d = %s\n", pi, valstr)
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
	emitStructAndArrayDefs(f, outf)
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

	verb(1, "gen fidx %d", fidx)

	checkTunables(tunables)

	// Generate a function with a random number of params and returns
	f := GenFunc(fidx)
	var fp *funcdef = &f

	// Emit caller side
	emitCaller(fp, calloutfile)

	// Emit checker side
	emitChecker(fp, checkoutfile)
}
