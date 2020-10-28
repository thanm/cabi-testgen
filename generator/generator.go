package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
)

const (
	initialVal     = 1
	decrementParam = 2
	constVal       = 3
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

	// Percentage of struct fields that should be "_". Ranges
	// from 0 to 100.
	blankPerc uint8

	// How deeply structs are allowed to be nested.
	structDepth uint8

	// Fraction of param and return types assigned to each
	// category: struct/array/int/float/complex/byte/pointer at the top
	// level. If nesting precludes using a struct, other types
	// are chosen from instead according to same proportions.
	typeFractions [7]uint8

	// Whether we try to emit recurive calls
	EmitRecur bool
}

var tunables = TunableParams{
	nParmRange:     20,
	nReturnRange:   10,
	nStructFields:  7,
	nArrayElements: 5,
	intBitRanges:   [4]uint8{30, 20, 20, 30},
	floatBitRanges: [2]uint8{50, 50},
	unsignedRanges: [2]uint8{50, 50},
	blankPerc:      0,
	structDepth:    3,
	typeFractions:  [7]uint8{30, 15, 20, 15, 5, 10, 5},
	EmitRecur:      true,
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

	if t.blankPerc > 100 {
		log.Fatal(errors.New("blankPerc bad value, over 100"))
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
	Declare(b *bytes.Buffer, prefix string, suffix string, caller bool)
	GenElemRef(elidx int, path string) (string, parm)
	GenValue(value int, caller bool) (string, int)
	IsControl() bool
	NumElements() int
	String() string
	TypeName() string
	QualName() string
}

type numparm struct {
	tag         string
	widthInBits uint32
	ctl         bool
}

var f32parm *numparm = &numparm{"float", uint32(32), false}
var f64parm *numparm = &numparm{"float", uint32(64), false}

func (p numparm) TypeName() string {
	if p.tag == "byte" {
		return "byte"
	}
	return fmt.Sprintf("%s%d", p.tag, p.widthInBits)
}

func (p numparm) QualName() string {
	return p.TypeName()
}

func (p numparm) String() string {
	if p.tag == "byte" {
		return "byte"
	}
	ctl := ""
	if p.ctl {
		ctl = " [ctl=yes]"
	}
	return fmt.Sprintf("%s%s", p.TypeName(), ctl)
}

func (p numparm) NumElements() int {
	return 1
}

func (p numparm) IsControl() bool {
	return p.ctl
}

func (p numparm) GenElemRef(elidx int, path string) (string, parm) {
	return path, p
}

func (p numparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	t := fmt.Sprintf("%s%d%s", p.tag, p.widthInBits, suffix)
	if p.tag == "byte" {
		t = fmt.Sprintf("%s%s", p.tag, suffix)
	}
	b.WriteString(prefix + " " + t)
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
	if p.tag == "uint" || p.tag == "byte" {
		var v int
		if which < 3 {
			// max
			v = (1 << p.widthInBits) - 1
		}
		nrange := 1 << (p.widthInBits - 2)
		v = rand.Intn(nrange)
		if p.tag == "byte" {
			return fmt.Sprintf("%s(%d)", p.tag, v), value + 1
		}
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
		if p.widthInBits == 64 {
			f1, v2 := f32parm.genRandNum(value)
			f2, v3 := f32parm.genRandNum(v2)
			return fmt.Sprintf("complex(%s,%s)", f1, f2), v3
		}
		if p.widthInBits == 128 {
			f1, v2 := f64parm.genRandNum(value)
			f2, v3 := f64parm.genRandNum(v2)
			return fmt.Sprintf("complex(%v,%v)", f1, f2), v3
		}
		panic("unknown complex type")
	}
	panic("unknown numeric type")
}

func (p numparm) GenValue(value int, caller bool) (string, int) {
	return p.genRandNum(value)
}

type structparm struct {
	sname  string
	qname  string
	fields []parm
	blank  []bool
}

func (p structparm) TypeName() string {
	return p.sname
}

func (p structparm) QualName() string {
	return p.qname
}

func (p structparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	n := p.sname
	if caller {
		n = p.qname
	}
	b.WriteString(fmt.Sprintf("%s %s%s", prefix, n, suffix))
}

func (p structparm) FieldName(i int) string {
	if p.blank[i] {
		return "_"
	}
	return fmt.Sprintf("F%d", i)
}

func (p structparm) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("struct %s {\n", p.sname))
	for fi, f := range p.fields {
		buf.WriteString(fmt.Sprintf("%s %s\n", p.FieldName(fi), f.String()))
	}
	buf.WriteString("}")
	return buf.String()
}

func (p structparm) GenValue(value int, caller bool) (string, int) {
	var buf bytes.Buffer

	n := p.sname
	if caller {
		n = p.qname
	}
	buf.WriteString(fmt.Sprintf("%s{", n))
	nbfi := 0
	for fi, f := range p.fields {
		if p.blank[fi] {
			continue
		}
		var valstr string
		writeCom(&buf, nbfi)
		buf.WriteString(p.FieldName(fi) + ": ")
		valstr, value = f.GenValue(value, caller)
		buf.WriteString(valstr)
		nbfi++
	}
	buf.WriteString("}")
	return buf.String(), value
}

func (p structparm) IsControl() bool {
	return false
}

func (p structparm) NumElements() int {
	ne := 0
	for fi, f := range p.fields {
		if p.blank[fi] {
			continue
		}
		ne += f.NumElements()
	}
	return ne
}

func (p structparm) GenElemRef(elidx int, path string) (string, parm) {
	ct := 0
	verb(4, "begin GenElemRef(%d,%s) on %s", elidx, path, p.String())
	for fi, f := range p.fields {
		fne := f.NumElements()
		if p.blank[fi] {
			continue
		}

		verb(4, "+ examining field %d fne %d ct %d", fi, fne, ct)

		// Empty field. Continue on.
		if elidx == ct && fne == 0 {
			continue
		}

		// Is this field the element we're interested in?
		if fne == 1 && elidx == ct {
			verb(4, "found field %d type %s in GenElemRef(%d,%s)", fi, f.String(), elidx, path)
			return fmt.Sprintf("%s.F%d", path, fi), f
		}

		// Is the element we want somewhere inside this field?
		if fne > 1 && elidx >= ct && elidx < ct+fne {
			ppath := fmt.Sprintf("%s.F%d", path, fi)
			return f.GenElemRef(elidx-ct, ppath)
		}

		ct += fne
	}
	panic(fmt.Sprintf("GenElemRef failed for struct %s elidx %d", p.TypeName(), elidx))
}

type arrayparm struct {
	aname     string
	qname     string
	nelements uint8
	eltype    parm
}

func (p arrayparm) IsControl() bool {
	return false
}

func (p arrayparm) TypeName() string {
	return p.aname
}

func (p arrayparm) QualName() string {
	return p.qname
}

func (p arrayparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	n := p.aname
	if caller {
		n = p.qname
	}
	b.WriteString(fmt.Sprintf("%s %s%s", prefix, n, suffix))
}

func (p arrayparm) String() string {
	return fmt.Sprintf("%s %d-element array of %s", p.aname, p.nelements, p.eltype.String())
}

func (p arrayparm) GenValue(value int, caller bool) (string, int) {
	var buf bytes.Buffer

	n := p.aname
	if caller {
		n = p.qname
	}
	buf.WriteString(fmt.Sprintf("%s{", n))
	for i := 0; i < int(p.nelements); i++ {
		var valstr string
		valstr, value = p.eltype.GenValue(value, caller)
		writeCom(&buf, i)
		buf.WriteString(valstr)
	}
	buf.WriteString("}")
	return buf.String(), value
}

func (p arrayparm) GenElemRef(elidx int, path string) (string, parm) {
	ene := p.eltype.NumElements()
	verb(4, "begin GenElemRef(%d,%s) on %s ene %d", elidx, path, p.String(), ene)

	// For empty arrays, convention is to return empty string
	if ene == 0 {
		return "", p
	}

	// Find slot within array of element of interest
	slot := elidx / ene

	// If this is the element we're interested in, return it
	if ene == 1 {
		verb(4, "hit scalar element")
		return fmt.Sprintf("%s[%d]", path, slot), p.eltype
	}

	verb(4, "recur slot=%d GenElemRef(%d,...)", slot, elidx-(slot*ene))

	// Otherwise our victim is somewhere inside the slot
	ppath := fmt.Sprintf("%s[%d]", path, slot)
	return p.eltype.GenElemRef(elidx-(slot*ene), ppath)
}

func (p arrayparm) NumElements() int {
	return p.eltype.NumElements() * int(p.nelements)
}

type pointerparm struct {
	tag    string
	totype parm
}

func (p pointerparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	n := p.totype.TypeName()
	if caller {
		n = p.totype.QualName()
	}
	b.WriteString(fmt.Sprintf("%s *%s%s", prefix, n, suffix))
}

func (p pointerparm) GenElemRef(elidx int, path string) (string, parm) {
	return path, p
}

func (p pointerparm) GenValue(value int, caller bool) (string, int) {
	n := p.totype.TypeName()
	if caller {
		n = p.totype.QualName()
	}
	return fmt.Sprintf("(*%s)(nil)", n), value
}

func (p pointerparm) IsControl() bool {
	return false
}

func (p pointerparm) NumElements() int {
	return 1
}

func (p pointerparm) String() string {
	return fmt.Sprintf("*%s", p.totype)
}

func (p pointerparm) TypeName() string {
	return fmt.Sprintf("*%s", p.totype.TypeName())
}

func (p pointerparm) QualName() string {
	return fmt.Sprintf("*%s", p.totype.QualName())
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

func (s *genstate) GenParm(f *funcdef, depth int, mkctl bool, pidx int) parm {

	// Enforcement for struct or array nesting depth (zeros tf[0] and
	// tf[1])
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

	// Convert tf into a cumulative sum
	sum := uint8(0)
	for i := 0; i < len(tf); i++ {
		sum += tf[i]
		tf[i] = sum
	}
	for i := 2; i < len(tf); i++ {
		tf[i] += off
	}

	// Make adjusted selection (pick a bucket within tf)
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
			sp.qname = fmt.Sprintf("%s.StructF%dS%d",
				s.checkerPkg(pidx), f.idx, ns)
			f.structdefs = append(f.structdefs, sp)
			tnf := int(tunables.nStructFields) / int(depth+1)
			nf := rand.Intn(tnf)
			for fi := 0; fi < nf; fi++ {
				fp := s.GenParm(f, depth+1, false, pidx)
				sp.fields = append(sp.fields, fp)
				isblank := uint8(rand.Intn(100)) < tunables.blankPerc
				sp.blank = append(sp.blank, isblank)
			}
			return sp
		}
	case which < tf[1]:
		{
			ap := new(arrayparm)
			ns := len(f.arraydefs)
			nel := uint8(rand.Intn(int(tunables.nArrayElements)))
			ap.aname = fmt.Sprintf("ArrayF%dS%dE%d", f.idx, ns, nel)
			ap.qname = fmt.Sprintf("%s.ArrayF%dS%dE%d", s.checkerPkg(pidx),
				f.idx, ns, nel)
			f.arraydefs = append(f.arraydefs, ap)
			ap.nelements = nel
			ap.eltype = s.GenParm(f, depth+1, false, pidx)
			return ap
		}
	case which < tf[2]:
		{
			var ip numparm
			ip.tag = intFlavor()
			ip.widthInBits = intBits()
			if mkctl {
				ip.ctl = true
			}
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
			fp.widthInBits = floatBits() * 2
			return fp
		}
	case which < tf[5]:
		{
			var bp numparm
			bp.tag = "byte"
			bp.widthInBits = 8
			return bp
		}
	case which < tf[6]:
		{
			var pp pointerparm
			pp.tag = "pointer"
			pp.totype = s.GenParm(f, depth, false, pidx)
			return pp
		}
	}

	// fallback
	var ip numparm
	ip.tag = "uint"
	ip.widthInBits = 8
	return ip
}

func (s *genstate) GenReturn(f *funcdef, depth int, pidx int) parm {
	return s.GenParm(f, depth, false, pidx)
}

func (s *genstate) GenFunc(fidx int, pidx int) funcdef {
	var f funcdef
	f.idx = fidx
	numParams := rand.Intn(6)
	numReturns := rand.Intn(5)
	needControl := tunables.EmitRecur
	for pi := 0; pi < numParams; pi++ {
		newparm := s.GenParm(&f, 0, needControl, pidx)
		if newparm.IsControl() {
			needControl = false
		}
		f.params = append(f.params, newparm)
	}
	for ri := 0; ri < numReturns; ri++ {
		f.returns = append(f.returns, s.GenReturn(&f, 0, pidx))
	}
	return f
}

func emitStructAndArrayDefs(f *funcdef, b *bytes.Buffer) {
	for _, s := range f.structdefs {
		b.WriteString(fmt.Sprintf("type %s struct {\n", s.sname))
		for fi, sp := range s.fields {
			sp.Declare(b, s.FieldName(fi), "\n", false)
		}
		b.WriteString("}\n\n")
	}
	for _, a := range f.arraydefs {
		b.WriteString(fmt.Sprintf("type %s [%d]%s\n\n", a.aname,
			a.nelements, a.eltype.TypeName()))
	}
}

func (s *genstate) emitCaller(f *funcdef, b *bytes.Buffer, pidx int) {

	b.WriteString(fmt.Sprintf("func Caller%d() {\n", f.idx))

	// generate return constants
	var value int = 1
	for ri, r := range f.returns {
		var valstr string
		valstr, value = r.GenValue(value, true)
		b.WriteString(fmt.Sprintf("  c%d := %s\n", ri, valstr))
	}

	// generate param constants
	value = 1
	for pi, p := range f.params {
		p.Declare(b, fmt.Sprintf("  var p%d", pi), "\n", true)
		var valstr string
		if p.IsControl() {
			valstr = "10"
		} else {
			valstr, value = p.GenValue(value, true)
		}
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
	b.WriteString(fmt.Sprintf("%s.Test%d(", s.checkerPkg(pidx), f.idx))
	for pi, _ := range f.params {
		writeCom(b, pi)
		b.WriteString(fmt.Sprintf("p%d", pi))
	}
	b.WriteString(")\n")

	// checking values returned
	for ri, _ := range f.returns {
		b.WriteString(fmt.Sprintf("  if r%d != c%d {\n", ri, ri))
		b.WriteString(fmt.Sprintf("    %s.NoteFailure(%d, \"return\", %d)\n",
			s.utilsPkg(), f.idx, ri))
		b.WriteString("  }\n")
	}

	b.WriteString("}\n\n")
}

func (s *genstate) emitChecker(f *funcdef, b *bytes.Buffer, pidx int) {
	verb(4, "emitting struct and array defs")
	emitStructAndArrayDefs(f, b)
	b.WriteString(fmt.Sprintf("// %d returns %d params\n", len(f.returns), len(f.params)))
	b.WriteString(fmt.Sprintf("func Test%d(", f.idx))

	// params
	for pi, p := range f.params {
		writeCom(b, pi)
		p.Declare(b, fmt.Sprintf("p%d", pi), "", false)
	}
	b.WriteString(") ")

	// returns
	if len(f.returns) > 0 {
		b.WriteString("(")
	}
	for ri, r := range f.returns {
		writeCom(b, ri)
		r.Declare(b, fmt.Sprintf("r%d", ri), "", false)
	}
	if len(f.returns) > 0 {
		b.WriteString(")")
	}
	b.WriteString(" {\n")

	// generate return constants
	value := 1
	for ri, r := range f.returns {
		var valstr string
		valstr, value = r.GenValue(value, false)
		b.WriteString(fmt.Sprintf("  rc%d := %s\n", ri, valstr))
	}

	// parameter checking code
	value = 1
	haveControl := false
	for pi, p := range f.params {
		verb(4, "emitting parm checking code for p%d numel=%d pt=%s", pi, p.NumElements(), p.TypeName())
		if p.IsControl() {
			b.WriteString(fmt.Sprintf("  if p%d == 0 {\n", pi))
			emitReturnConst(f, b)
			b.WriteString("  }\n")
			haveControl = true
		} else {
			numel := p.NumElements()
			for i := 0; i < numel; i++ {
				var valstr string
				verb(4, "emitting check-code for p%d el %d", pi, i)
				elref, elparm := p.GenElemRef(i, fmt.Sprintf("p%d", pi))
				valstr, value = elparm.GenValue(value, false)
				if elref == "" {
					verb(4, "empty skip p%d el %d", pi, i)
					continue
				}
				b.WriteString(fmt.Sprintf("  p%df%dc := %s\n", pi, i, valstr))
				b.WriteString(fmt.Sprintf("  if %s != p%df%dc {\n", elref, pi, i))
				b.WriteString(fmt.Sprintf("    %s.NoteFailureElem(%d, \"parm\", %d, %d)\n", s.utilsPkg(), f.idx, pi, i))
				b.WriteString("    return\n")
				b.WriteString("  }\n")
			}
		}
	}

	// return recursive call if we have a control, const val otherwise
	if haveControl {
		b.WriteString(" // recursive call\n")
		if len(f.returns) > 0 {
			b.WriteString(" return ")
		}
		b.WriteString(fmt.Sprintf(" Test%d(", f.idx))
		for pi, p := range f.params {
			writeCom(b, pi)
			if p.IsControl() {
				b.WriteString(fmt.Sprintf(" p%d-1", pi))
			} else {
				b.WriteString(fmt.Sprintf(" p%d", pi))
			}
		}
		b.WriteString(")\n")
		if len(f.returns) == 0 {
			b.WriteString(" return\n")
		}
	} else {
		emitReturnConst(f, b)
	}

	b.WriteString("}\n\n")
}

func emitReturnConst(f *funcdef, b *bytes.Buffer) {
	// returning code
	b.WriteString("    return ")
	if len(f.returns) > 0 {
		for ri, _ := range f.returns {
			writeCom(b, ri)
			b.WriteString(fmt.Sprintf("rc%d", ri))
		}
	}
	b.WriteString("\n")
}

func (s *genstate) GenPair(calloutfile *os.File, checkoutfile *os.File, fidx int, pidx int, b *bytes.Buffer, seed int64, emit bool) int64 {

	verb(1, "gen fidx %d pidx %d", fidx, pidx)

	checkTunables(tunables)

	// Generate a function with a random number of params and returns
	f := s.GenFunc(fidx, pidx)
	var fp *funcdef = &f

	// Emit caller side
	rand.Seed(seed)
	s.emitCaller(fp, b, pidx)
	if emit {
		b.WriteTo(calloutfile)
	}
	b.Reset()

	// Emit checker side
	rand.Seed(seed)
	s.emitChecker(fp, b, pidx)
	if emit {
		b.WriteTo(checkoutfile)
	}
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
		outf.WriteString(fmt.Sprintf("import \"%s%s\"\n", ipref, imp))
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
	fmt.Fprintf(outf, "\"Error: fail on =Test%%d= %%s %%d\\n\", fidx, pref, parmNo)\n")
	fmt.Fprintf(outf, "  if (FailCount > 10) {\n")
	fmt.Fprintf(outf, "    os.Exit(1)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "}\n\n")
	fmt.Fprintf(outf, "func NoteFailureElem(fidx int, pref string, parmNo int, elem int) {\n")
	fmt.Fprintf(outf, "  FailCount += 1\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail on =Test%%d= %%s %%d elem %%d\\n\", fidx, pref, parmNo, elem)\n")
	fmt.Fprintf(outf, "  if (FailCount > 10) {\n")
	fmt.Fprintf(outf, "    os.Exit(1)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "}\n\n")
}

func (s *genstate) emitMain(outf *os.File, numit int) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "func main() {\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"starting main\\n\")\n")
	for k := 0; k < s.numtpk; k++ {
		cp := fmt.Sprintf("%sCaller%d", s.tag, k)
		for i := 0; i < numit; i++ {
			fmt.Fprintf(outf, "  %s.Caller%d()\n", cp, i)
		}
	}
	fmt.Fprintf(outf, "  if %s.FailCount != 0 {\n", s.utilsPkg())
	fmt.Fprintf(outf, "    fmt.Fprintf(os.Stderr, \"FAILURES: %%d\\n\", %s.FailCount)\n", s.utilsPkg())
	fmt.Fprintf(outf, "    os.Exit(2)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"finished %d tests\\n\")\n", numit*s.numtpk)
	fmt.Fprintf(outf, "}\n")
}

func makeDir(d string) {
	verb(1, "creating %s", d)
	os.Mkdir(d, 0777)
}

type genstate struct {
	outdir string
	ipref  string
	tag    string
	numtpk int
}

func (s *genstate) callerPkg(which int) string {
	return s.tag + "Caller" + strconv.Itoa(which)
}

func (s *genstate) callerFile(which int) string {
	cp := s.callerPkg(which)
	return s.outdir + "/" + cp + "/" + cp + ".go"
}

func (s *genstate) checkerPkg(which int) string {
	return s.tag + "Checker" + strconv.Itoa(which)
}

func (s *genstate) checkerFile(which int) string {
	cp := s.checkerPkg(which)
	return s.outdir + "/" + cp + "/" + cp + ".go"
}

func (s *genstate) utilsPkg() string {
	return s.tag + "Utils"
}

func Generate(tag string, outdir string, pkgpath string, numit int, numtpkgs int, seed int64, fcnmask map[int]int) {
	mainpkg := tag + "Main"

	var ipref string
	if len(pkgpath) > 0 {
		ipref = pkgpath + "/"
	}

	s := genstate{outdir: outdir, ipref: ipref, tag: tag, numtpk: numtpkgs}

	if outdir != "." {
		makeDir(outdir)
	}
	verb(1, "creating %s", outdir)

	mainimports := []string{}
	for i := 0; i < numtpkgs; i++ {
		makeDir(outdir + "/" + s.callerPkg(i))
		makeDir(outdir + "/" + s.checkerPkg(i))
		makeDir(outdir + "/" + s.utilsPkg())
		mainimports = append(mainimports, s.callerPkg(i))
	}
	mainimports = append(mainimports, s.utilsPkg())

	utilsfile := outdir + "/" + s.utilsPkg() + "/" + s.utilsPkg() + ".go"
	utilsoutfile := openOutputFile(utilsfile, s.utilsPkg(), []string{}, "")
	verb(1, "emit utils")
	emitUtils(utilsoutfile)
	utilsoutfile.Close()

	mainfile := outdir + "/" + mainpkg + ".go"
	mainoutfile := openOutputFile(mainfile, "main", mainimports, ipref)

	for k := 0; k < numtpkgs; k++ {
		calleroutfile := openOutputFile(s.callerFile(k), s.callerPkg(k),
			[]string{s.checkerPkg(k), s.utilsPkg()}, ipref)
		checkeroutfile := openOutputFile(s.checkerFile(k), s.checkerPkg(k),
			[]string{s.utilsPkg()}, ipref)

		var b bytes.Buffer
		for i := 0; i < numit; i++ {
			doemit := false
			if len(fcnmask) == 0 {
				doemit = true
			} else if _, ok := fcnmask[i]; ok {
				doemit = true
			}
			seed = s.GenPair(calleroutfile, checkeroutfile, i, k,
				&b, seed, doemit)
		}
		calleroutfile.Close()
		checkeroutfile.Close()
	}
	s.emitMain(mainoutfile, numit)

	// emit go.mod
	verb(1, "opening go.mod")
	fn := outdir + "/go.mod"
	outf, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	outf.WriteString(fmt.Sprintf("module %s\n\ngo 1.15\n", pkgpath))
	outf.Close()

	verb(1, "closing files")
	mainoutfile.Close()
}
