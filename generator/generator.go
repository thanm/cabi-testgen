package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
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
	structDepth:    1,
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

func writeCom(b *bytes.Buffer, i int) {
	if i != 0 {
		b.WriteString(", ")
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
	Declare(b *bytes.Buffer, prefix string, suffix string, parmNo int)
	GenValue(value int) (string, int)
}

type numparm struct {
	tag         string
	widthInBits uint32
}

var f32parm *numparm = &numparm{"float", uint32(32)}
var f64parm *numparm = &numparm{"float", uint32(64)}

func (p numparm) TypeName() string {
	return fmt.Sprintf("%s%d", p.tag, p.widthInBits)
}

func (p numparm) Declare(b *bytes.Buffer, prefix string, suffix string, parmNo int) {
	b.WriteString(fmt.Sprintf("%s%d %s%d%s", prefix, parmNo, p.tag, p.widthInBits, suffix))
}

func (p numparm) genRandNum(value int) (string, int) {
	which := uint8(rand.Intn(100))
	if p.tag == "int" {
		var v int
		if which < 3 {
			// max
			v = (1 << (p.widthInBits - 1)) - 1
		} else if which < 5 {
			// min
			v = (-1 << (p.widthInBits - 1))
		} else {
			v = rand.Intn(1 << (p.widthInBits - 2))
			if value%2 != 0 {
				v = -v
			}
		}
		return fmt.Sprintf("%s%d(%d)", p.tag, p.widthInBits, v), value + 1
	}
	if p.tag == "uint" {
		var v int
		if which < 3 {
			// max
			v = (1 << p.widthInBits) - 1
		}
		nrange := 1 << (p.widthInBits - 2)
		v = rand.Intn(nrange)
		return fmt.Sprintf("%s%d(%d)", p.tag, p.widthInBits, v), value + 1
	}
	if p.tag == "float" {
		if p.widthInBits == 32 {
			rf := rand.Float32() * (math.MaxFloat32 / 4)
			if value%2 != 0 {
				rf = -rf
			}
			return fmt.Sprintf("%s%d(%v)", p.tag, p.widthInBits, rf), value + 1
		}
		if p.widthInBits == 64 {
			rf := rand.Float64() * (math.MaxFloat64 / 4)
			if value%2 != 0 {
				rf = -rf
			}
			return fmt.Sprintf("%s%d(%v)", p.tag, p.widthInBits,
				rand.NormFloat64()), value + 1
		}
		panic("unknown float type")
	}
	if p.tag == "complex" {
		if p.widthInBits == 32 {
			f1, v2 := f32parm.genRandNum(value)
			f2, v3 := f32parm.genRandNum(v2)
			return fmt.Sprintf("complex32(%s,%s)", f1, f2), v3
		}
		if p.widthInBits == 64 {
			f1, v2 := f64parm.genRandNum(value)
			f2, v3 := f64parm.genRandNum(v2)
			return fmt.Sprintf("complex64(%v,%v)", f1, f2), v3
		}
		panic("unknown complex type")
	}
	panic("unknown numeric type")
}

func (p numparm) GenValue(value int) (string, int) {
	return p.genRandNum(value)

	// var s string
	// v := value
	// if p.tag == "int" && v%2 != 0 {
	// 	v = -v
	// }
	// if p.tag == "complex" {
	// 	s = fmt.Sprintf("complex(%d, %d)", value, value)
	// } else {
	// 	s = fmt.Sprintf("%s%d(%d)", p.tag, p.widthInBits, v)
	// }
	// return s, value + 1
}

type structparm struct {
	sname  string
	fields []parm
}

func (p structparm) TypeName() string {
	return p.sname
}

func (p structparm) Declare(b *bytes.Buffer, prefix string, suffix string, parmNo int) {
	b.WriteString(fmt.Sprintf("%s%d %s%s", prefix, parmNo, p.sname, suffix))
}

func (p structparm) GenValue(value int) (string, int) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s{", p.sname))
	for fi, f := range p.fields {
		var valstr string
		valstr, value = f.GenValue(value)
		writeCom(&buf, fi)
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

func (p arrayparm) Declare(b *bytes.Buffer, prefix string, suffix string, parmNo int) {

	b.WriteString(fmt.Sprintf("%s%d %s%s", prefix, parmNo, p.aname, suffix))
}

func (p arrayparm) GenValue(value int) (string, int) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s{", p.aname))
	for i := 0; i < int(p.nelements); i++ {
		var valstr string
		valstr, value = p.eltype.GenValue(value)
		writeCom(&buf, i)
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

func intBits() uint32 {
	which := uint8(rand.Intn(100))
	var t uint8 = 0
	var bits uint32 = 8
	for _, v := range tunables.intBitRanges {
		t += v
		if which < t {
			return bits
		}
		bits *= 2
	}
	return uint32(tunables.intBitRanges[3])
}

func floatBits() uint32 {
	which := uint8(rand.Intn(100))
	if which < tunables.floatBitRanges[0] {
		return uint32(32)
	}
	return uint32(64)
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
			var fp numparm
			fp.tag = "complex"
			fp.widthInBits = floatBits()
			return fp
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

func emitStructAndArrayDefs(f *funcdef, b *bytes.Buffer) {
	for _, s := range f.structdefs {
		b.WriteString(fmt.Sprintf("type %s struct {\n", s.sname))
		for fi, sp := range s.fields {
			sp.Declare(b, "F", "\n", fi)
		}
		b.WriteString("}\n\n")
	}
	for _, a := range f.arraydefs {
		b.WriteString(fmt.Sprintf("type %s [%d]%s\n\n", a.aname,
			a.nelements, a.eltype.TypeName()))
	}
}

func emitCaller(f *funcdef, b *bytes.Buffer) {

	b.WriteString(fmt.Sprintf("func Caller%d() {\n", f.idx))
	var value int = 1
	for pi, p := range f.params {
		p.Declare(b, "  var p", "\n", pi)
		var valstr string
		valstr, value = p.GenValue(value)
		b.WriteString(fmt.Sprintf("  p%d = %s\n", pi, valstr))
	}

	// calling code
	b.WriteString(fmt.Sprintf("  // %d returns %d params\n",
		len(f.returns), len(f.params)))
	b.WriteString("  ")
	for ri, _ := range f.returns {
		writeCom(b, ri)
		b.WriteString(fmt.Sprintf("r%d", ri))
	}
	if len(f.returns) > 0 {
		b.WriteString(" := ")
	}
	b.WriteString(fmt.Sprintf("Test%d(", f.idx))
	for pi, _ := range f.params {
		writeCom(b, pi)
		b.WriteString(fmt.Sprintf("p%d", pi))
	}
	b.WriteString(")\n")

	// checking values returned
	value = 1
	for ri, r := range f.returns {
		var valstr string
		valstr, value = r.GenValue(value)
		b.WriteString(fmt.Sprintf("  c%d := %s\n", ri, valstr))
		b.WriteString(fmt.Sprintf("  if r%d != c%d {\n", ri, ri))
		b.WriteString(fmt.Sprintf("    NoteFailure(%d, \"return\", %d)\n", f.idx, ri))
		b.WriteString("  }\n")
	}

	b.WriteString("}\n\n")
}

func emitChecker(f *funcdef, b *bytes.Buffer) {
	emitStructAndArrayDefs(f, b)
	b.WriteString(fmt.Sprintf("// %d returns %d params\n", len(f.returns), len(f.params)))
	b.WriteString(fmt.Sprintf("func Test%d(", f.idx))

	// params
	for pi, p := range f.params {
		writeCom(b, pi)
		p.Declare(b, "p", "", pi)
	}
	b.WriteString(") ")

	// returns
	if len(f.returns) > 0 {
		b.WriteString("(")
	}
	for ri, r := range f.returns {
		writeCom(b, ri)
		r.Declare(b, "r", "", ri)
	}
	if len(f.returns) > 0 {
		b.WriteString(")")
	}
	b.WriteString(" {\n")

	// checking code
	value := 1
	for pi, p := range f.params {
		var valstr string
		valstr, value = p.GenValue(value)
		b.WriteString(fmt.Sprintf("  c%d := %s\n", pi, valstr))
		b.WriteString(fmt.Sprintf("  if p%d != c%d {\n", pi, pi))
		b.WriteString(fmt.Sprintf("    NoteFailure(%d, \"parm\", %d)\n", f.idx, pi))
		b.WriteString("  }\n")
	}

	// returning code
	b.WriteString("  return ")
	if len(f.returns) > 0 {
		value := 1
		for ri, r := range f.returns {
			var valstr string
			writeCom(b, ri)
			valstr, value = r.GenValue(value)
			b.WriteString(fmt.Sprintf("%s", valstr))
		}
	}
	b.WriteString("\n")
	b.WriteString("}\n\n")
}

func GenPair(calloutfile *os.File, checkoutfile *os.File, fidx int, b *bytes.Buffer, seed int64) int64 {

	verb(1, "gen fidx %d", fidx)

	checkTunables(tunables)

	// Generate a function with a random number of params and returns
	f := GenFunc(fidx)
	var fp *funcdef = &f

	// Emit caller side
	rand.Seed(seed)
	emitCaller(fp, b)
	b.WriteTo(calloutfile)
	b.Reset()

	// Emit checker side
	rand.Seed(seed)
	emitChecker(fp, b)
	b.WriteTo(checkoutfile)
	b.Reset()

	return seed + 1
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
	fmt.Fprintf(outf, "var FailCount int\n\n")
	fmt.Fprintf(outf, "func NoteFailure(fidx int, pref string, parmNo int) {\n")
	fmt.Fprintf(outf, "  FailCount += 1\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail on func %%d %%s %%d\\n\", fidx, pref, parmNo)\n")
	fmt.Fprintf(outf, "}\n\n")
}

func emitMain(outf *os.File, numit int) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "func main() {\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"starting main\\n\")\n")
	for i := 0; i < numit; i++ {
		fmt.Fprintf(outf, "  Caller%d()\n", i)
	}
	fmt.Fprintf(outf, "  if FailCount != 0 {\n")
	fmt.Fprintf(outf, "    fmt.Fprintf(os.Stderr, \"FAILURES: %%d\\n\", FailCount)\n")
	fmt.Fprintf(outf, "    os.Exit(2)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"finished %d tests\\n\")\n", numit)
	fmt.Fprintf(outf, "}\n")
}

func makeDir(d string) {
	verb(1, "creating %s", d)
	os.Mkdir(d, 0777)
}

func Generate(tag string, outdir string, pkgpath string, numit int, seed int64) {
	callerpkg := tag + "Caller"
	checkerpkg := tag + "Checker"
	utilspkg := tag + "Utils"
	mainpkg := tag + "Main"

	var ipref string
	if len(pkgpath) > 0 {
		ipref = pkgpath + "/"
	}

	if outdir != "." {
		makeDir(outdir)
	}
	verb(1, "creating %s", outdir)
	makeDir(outdir + "/" + callerpkg)
	makeDir(outdir + "/" + callerpkg)
	makeDir(outdir + "/" + checkerpkg)
	makeDir(outdir + "/" + utilspkg)

	callerfile := outdir + "/" + callerpkg + "/" + callerpkg + ".go"
	checkerfile := outdir + "/" + checkerpkg + "/" + checkerpkg + ".go"
	utilsfile := outdir + "/" + utilspkg + "/" + utilspkg + ".go"
	mainfile := outdir + "/" + mainpkg + ".go"

	verb(1, "files: %s %s %s %s", callerfile, checkerfile, utilsfile, mainfile)

	calleroutfile := openOutputFile(callerfile, callerpkg,
		[]string{checkerpkg, utilspkg}, ipref)
	checkeroutfile := openOutputFile(checkerfile, checkerpkg,
		[]string{utilspkg}, ipref)
	utilsoutfile := openOutputFile(utilsfile, utilspkg, []string{}, "")
	mainoutfile := openOutputFile(mainfile, "main", []string{callerpkg, utilspkg}, ipref)

	verb(1, "emit utils")
	emitUtils(utilsoutfile)
	emitMain(mainoutfile, numit)
	var b bytes.Buffer
	for i := 0; i < numit; i++ {
		seed = GenPair(calleroutfile, checkeroutfile, i, &b, seed)
	}

	verb(1, "closing files")
	utilsoutfile.Close()
	calleroutfile.Close()
	checkeroutfile.Close()
	mainoutfile.Close()
}
