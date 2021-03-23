package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
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

	// Percentage of params, struct fields that should be "_". Ranges
	// from 0 to 100.
	blankPerc uint8

	// How deeply structs are allowed to be nested.
	structDepth uint8

	// Fraction of param and return types assigned to each of:
	// struct/array/int/float/complex/byte/pointer/string at the
	// top level. If nesting precludes using a struct, other types
	// are chosen from instead according to same proportions.
	typeFractions [8]uint8

	// Percentage of the time we'll emit recursive calls, from 0 to 100.
	recurPerc uint8

	// Percentage of time that we turn the test function into a method,
	// and if it is a method, fraction of time that we use a pointer
	// method call vs value method call.
	methodPerc            uint8
	pointerMethodCallPerc uint8

	// If true, test reflect.Call path as well.
	doReflectCall bool

	// If true, then randomly take addresses of params/returns.
	takeAddress bool

	// Fraction of the time that any params/returns are address taken.
	takenFraction uint8

	// For a given address-taken param or return, controls the
	// manner in which the indirect read or write takes
	// place. This is a set of percentages for
	// not/simple/passed/heap, where "not" means not address
	// taken, "simple" means a simple read or write, "passed"
	// means that the address is passed to a well-behaved
	// function, and "heap" means that the address is assigned to
	// a global.
	addrFractions [4]uint8

	// If true, then perform testing of go/defer statements.
	doDefer bool

	// fraction of test functions for which we emit a defer
	deferFraction uint8
}

var tunables = TunableParams{
	nParmRange:     15,
	nReturnRange:   7,
	nStructFields:  7,
	nArrayElements: 5,
	intBitRanges:   [4]uint8{30, 20, 20, 30},
	floatBitRanges: [2]uint8{50, 50},
	unsignedRanges: [2]uint8{50, 50},
	blankPerc:      15,
	structDepth:    3,
	typeFractions: [8]uint8{
		20, // struct
		15, // array
		25, // numeric
		15, // float
		5,  // complex
		5,  // byte
		5,  // pointer
		10, // string
	},
	recurPerc:             20,
	methodPerc:            10,
	pointerMethodCallPerc: 50,
	doReflectCall:         true,
	doDefer:               true,
	takeAddress:           true,
	takenFraction:         20,
	deferFraction:         30,
	addrFractions:         [4]uint8{50, 25, 15, 10},
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
	if t.recurPerc > 100 {
		log.Fatal(errors.New("recurPerc bad value, over 100"))
	}
	if t.methodPerc > 100 {
		log.Fatal(errors.New("methodPerc bad value, over 100"))
	}
	if t.pointerMethodCallPerc > 100 {
		log.Fatal(errors.New("pointerMethodCallPerc bad value, over 100"))
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

	s = 0
	for _, v := range t.addrFractions {
		s += int(v)
	}
	if s != 100 {
		log.Fatal(errors.New("addrFractions tunable does not sum to 100"))
	}
	if t.takenFraction > 100 {
		log.Fatal(errors.New("takenFraction not between 0 and 100"))
	}
	if t.deferFraction > 100 {
		log.Fatal(errors.New("deferFraction not between 0 and 100"))
	}
}

func SetTunables(t TunableParams) {
	checkTunables(t)
	tunables = t
}

func (t *TunableParams) DisableReflectionCalls() {
	t.doReflectCall = false
}

func (t *TunableParams) DisableRecursiveCalls() {
	t.recurPerc = 0
}

func (t *TunableParams) DisableMethodCalls() {
	t.methodPerc = 0
}

func (t *TunableParams) DisableTakeAddr() {
	t.takeAddress = false
}

func (t *TunableParams) DisableDefer() {
	t.doDefer = false
}

func (t *TunableParams) LimitInputs(n int) error {
	if n > 100 {
		return fmt.Errorf("value %d passed to LimitInputs is too large *(max 100)", n)
	}
	if n < 0 {
		return fmt.Errorf("value %d passed to LimitInputs is invalid", n)
	}
	t.nParmRange = uint8(n)
	return nil
}

func (t *TunableParams) LimitOutputs(n int) error {
	if n > 100 {
		return fmt.Errorf("value %d passed to LimitOutputs is too large *(max 100)", n)
	}
	if n < 0 {
		return fmt.Errorf("value %d passed to LimitOutputs is invalid", n)
	}
	t.nReturnRange = uint8(n)
	return nil
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

type funcdef struct {
	idx        int
	structdefs []structparm
	arraydefs  []arrayparm
	typedefs   []typedefparm
	receiver   parm
	params     []parm
	returns    []parm
	values     []int
	recur      bool
	method     bool
}

type genstate struct {
	outdir    string
	ipref     string
	tag       string
	numtpk    int
	errs      int
	pragma    string
	tunables  TunableParams
	tstack    []TunableParams
	pfuncs    map[string]string
	newpfuncs []funcdesc
	rfuncs    map[string]string
	newrfuncs []funcdesc
	gvars     map[string]string
	newgvars  []funcdesc
}

func (s *genstate) intFlavor() string {
	which := uint8(rand.Intn(100))
	if which < s.tunables.unsignedRanges[0] {
		return "uint"
	}
	return "int"
}

func (s *genstate) intBits() uint32 {
	which := uint8(rand.Intn(100))
	var t uint8 = 0
	var bits uint32 = 8
	for _, v := range s.tunables.intBitRanges {
		t += v
		if which < t {
			return bits
		}
		bits *= 2
	}
	return uint32(s.tunables.intBitRanges[3])
}

func (s *genstate) floatBits() uint32 {
	which := uint8(rand.Intn(100))
	if which < s.tunables.floatBitRanges[0] {
		return uint32(32)
	}
	return uint32(64)
}

func (s *genstate) genAddrTaken() addrTakenHow {
	which := uint8(rand.Intn(100))
	res := notAddrTaken
	var t uint8 = 0
	for _, v := range s.tunables.addrFractions {
		t += v
		if which < t {
			return res
		}
		res++
	}
	return notAddrTaken
}

func (s *genstate) pushTunables() {
	s.tstack = append(s.tstack, s.tunables)
}

func (s *genstate) popTunables() {
	if len(s.tstack) == 0 {
		panic("untables stack underflow")
	}
	s.tunables = s.tstack[0]
	s.tstack = s.tstack[1:]
}

func (s *genstate) precludePointerTypes() {
	s.tunables.typeFractions[0] += s.tunables.typeFractions[6]
	s.tunables.typeFractions[6] = 0
	checkTunables(s.tunables)
}

func (s *genstate) GenParm(f *funcdef, depth int, mkctl bool, pidx int) parm {

	// Enforcement for struct or array nesting depth (zeros tf[0] and
	// tf[1])
	tf := s.tunables.typeFractions
	amt := 100
	off := uint8(0)
	toodeep := depth >= int(s.tunables.structDepth)
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

	isblank := uint8(rand.Intn(100)) < s.tunables.blankPerc

	addrTaken := notAddrTaken
	if depth == 0 && tunables.takeAddress && !isblank {
		addrTaken = s.genAddrTaken()
	}

	// Make adjusted selection (pick a bucket within tf)
	which := uint8(rand.Intn(amt)) + off
	verb(3, "which=%d", which)
	var retval parm
	switch {
	case which < tf[0]:
		{
			if toodeep {
				panic("should not be here")
			}

			var sp structparm
			ns := len(f.structdefs)
			sp.sname = fmt.Sprintf("StructF%dS%d", f.idx, ns)
			sp.qname = fmt.Sprintf("%s.StructF%dS%d",
				s.checkerPkg(pidx), f.idx, ns)
			f.structdefs = append(f.structdefs, sp)
			tnf := int(s.tunables.nStructFields) / int(depth+1)
			nf := rand.Intn(tnf)
			for fi := 0; fi < nf; fi++ {
				fp := s.GenParm(f, depth+1, false, pidx)
				sp.fields = append(sp.fields, fp)
			}
			f.structdefs[ns] = sp
			retval = &sp
		}
	case which < tf[1]:
		{
			var ap arrayparm
			ns := len(f.arraydefs)
			nel := uint8(rand.Intn(int(s.tunables.nArrayElements)))
			ap.aname = fmt.Sprintf("ArrayF%dS%dE%d", f.idx, ns, nel)
			ap.qname = fmt.Sprintf("%s.ArrayF%dS%dE%d", s.checkerPkg(pidx),
				f.idx, ns, nel)
			f.arraydefs = append(f.arraydefs, ap)
			ap.nelements = nel
			ap.eltype = s.GenParm(f, depth+1, false, pidx)
			ap.eltype.SetBlank(false)
			f.arraydefs[ns] = ap
			retval = &ap
		}
	case which < tf[2]:
		{
			var ip numparm
			ip.tag = s.intFlavor()
			ip.widthInBits = s.intBits()
			if mkctl {
				ip.ctl = true
			}
			retval = &ip
		}
	case which < tf[3]:
		{
			var fp numparm
			fp.tag = "float"
			fp.widthInBits = s.floatBits()
			retval = &fp
		}
	case which < tf[4]:
		{
			var fp numparm
			fp.tag = "complex"
			fp.widthInBits = s.floatBits() * 2
			retval = &fp
		}
	case which < tf[5]:
		{
			var bp numparm
			bp.tag = "byte"
			bp.widthInBits = 8
			retval = &bp
		}
	case which < tf[6]:
		{
			pp := mkPointerParm(s.GenParm(f, depth, false, pidx))
			retval = &pp
		}
	case which < tf[7]:
		{
			var sp stringparm
			sp.tag = "string"
			retval = &sp
		}
	default:
		{
			// fallback
			var ip numparm
			ip.tag = "uint"
			ip.widthInBits = 8
			retval = &ip
		}
	}
	if !mkctl {
		retval.SetBlank(isblank)
	}
	retval.SetAddrTaken(addrTaken)
	return retval
}

func (s *genstate) GenReturn(f *funcdef, depth int, pidx int) parm {
	return s.GenParm(f, depth, false, pidx)
}

func (s *genstate) GenFunc(fidx int, pidx int) *funcdef {
	f := new(funcdef)
	f.idx = fidx
	numParams := rand.Intn(1 + int(s.tunables.nParmRange))
	numReturns := rand.Intn(1 + int(s.tunables.nReturnRange))
	f.recur = uint8(rand.Intn(100)) < s.tunables.recurPerc
	f.method = uint8(rand.Intn(100)) < s.tunables.methodPerc
	if f.method {
		// Receiver type can't be pointer type. Temporarily update
		// tunables to eliminate that possibility.
		s.pushTunables()
		s.precludePointerTypes()
		target := s.GenParm(f, 0, false, pidx)
		target.SetBlank(false)
		s.popTunables()
		f.receiver = s.makeTypedefParm(f, target, pidx)
		if f.receiver.IsBlank() {
			f.recur = false
		}
	}
	needControl := f.recur
	pTaken := uint8(rand.Intn(100)) < s.tunables.takenFraction
	for pi := 0; pi < numParams; pi++ {
		newparm := s.GenParm(f, 0, needControl, pidx)
		if !pTaken {
			newparm.SetAddrTaken(notAddrTaken)
		}
		if newparm.IsControl() {
			needControl = false
		}
		f.params = append(f.params, newparm)
	}
	if f.recur && needControl {
		f.recur = false
	}

	rTaken := uint8(rand.Intn(100)) < s.tunables.takenFraction
	for ri := 0; ri < numReturns; ri++ {
		r := s.GenReturn(f, 0, pidx)
		if !rTaken {
			r.SetAddrTaken(notAddrTaken)
		}
		f.returns = append(f.returns, r)
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
	for _, td := range f.typedefs {
		b.WriteString(fmt.Sprintf("type %s %s\n\n", td.aname,
			td.target.TypeName()))
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
	for pi, p := range f.params {
		verb(4, "emitCaller gen p%d value=%d", pi, value)
		p.Declare(b, fmt.Sprintf("  var p%d", pi), "\n", true)
		var valstr string
		if p.IsControl() {
			valstr = "10"
		} else {
			valstr, value = p.GenValue(value, true)
		}
		b.WriteString(fmt.Sprintf("  p%d = %s\n", pi, valstr))
		f.values = append(f.values, value)
	}

	// generate receiver constant if applicable
	if f.method {
		f.receiver.Declare(b, "  var rcvr", "\n", true)
		valstr, value := f.receiver.GenValue(value, true)
		b.WriteString(fmt.Sprintf("  rcvr = %s\n", valstr))
		f.values = append(f.values, value)
	}

	// calling code
	b.WriteString(fmt.Sprintf("  // %d returns %d params\n",
		len(f.returns), len(f.params)))
	b.WriteString("  ")
	for ri := range f.returns {
		writeCom(b, ri)
		b.WriteString(fmt.Sprintf("r%d", ri))
	}
	if len(f.returns) > 0 {
		b.WriteString(" := ")
	}
	pref := s.checkerPkg(pidx)
	if f.method {
		pref = "rcvr"
	}
	b.WriteString(fmt.Sprintf("%s.Test%d(", pref, f.idx))
	for pi := range f.params {
		writeCom(b, pi)
		b.WriteString(fmt.Sprintf("p%d", pi))
	}
	b.WriteString(")\n")

	// checking values returned
	for ri := range f.returns {
		b.WriteString(fmt.Sprintf("  if r%d != c%d {\n", ri, ri))
		b.WriteString(fmt.Sprintf("    %s.NoteFailure(%d, \"%s\", \"return\", %d, uint64(0))\n", s.utilsPkg(), f.idx, s.checkerPkg(pidx), ri))
		b.WriteString("  }\n")
	}

	if s.tunables.doReflectCall {
		// now make the same call via reflection
		b.WriteString("  // same call via reflection\n")
		if f.method {
			b.WriteString("  rcv := reflect.ValueOf(rcvr)\n")
			b.WriteString(fmt.Sprintf("  rc := rcv.MethodByName(\"Test%d\")\n", f.idx))
		} else {
			b.WriteString(fmt.Sprintf("  rc := reflect.ValueOf(%s.Test%d)\n",
				s.checkerPkg(pidx), f.idx))
		}
		b.WriteString("  ")
		if len(f.returns) > 0 {
			b.WriteString("rvslice := ")
		}
		b.WriteString("  rc.Call([]reflect.Value{")
		for pi := range f.params {
			writeCom(b, pi)
			b.WriteString(fmt.Sprintf("reflect.ValueOf(p%d)", pi))
		}
		b.WriteString("})\n")

		// check values returned
		for ri, r := range f.returns {
			b.WriteString(fmt.Sprintf("  rr%di := rvslice[%d].Interface()\n", ri, ri))
			b.WriteString(fmt.Sprintf("  rr%dv:= rr%di.(", ri, ri))
			r.Declare(b, "", "", true)
			b.WriteString(")\n")
			b.WriteString(fmt.Sprintf("  if rr%dv != c%d {\n", ri, ri))
			b.WriteString(fmt.Sprintf("    %s.NoteFailure(%d, \"%s\", \"reflect return\", %d, uint64(0))\n", s.utilsPkg(), f.idx, s.checkerPkg(pidx), ri))
			b.WriteString("  }\n")
		}
	}

	b.WriteString("}\n\n")
}

func checkableElements(p parm) int {
	if p.IsBlank() {
		return 0
	}
	sp, isstruct := p.(*structparm)
	if isstruct {
		s := 0
		for fi := range sp.fields {
			s += checkableElements(sp.fields[fi])
		}
		return s
	}
	ap, isarray := p.(*arrayparm)
	if isarray {
		if ap.nelements == 0 {
			return 0
		}
		return int(ap.nelements) * checkableElements(ap.eltype)
	}
	return 1
}

type funcdesc struct {
	p    parm
	pp   parm
	name string
}

func (s *genstate) emitDerefFuncs(b *bytes.Buffer) {
	for _, fd := range s.newpfuncs {
		b.WriteString("\n//go:noinline\n")
		b.WriteString(fmt.Sprintf("func %s(", fd.name))
		fd.pp.Declare(b, "x", "", false)
		b.WriteString(") ")
		fd.p.Declare(b, "", "", false)
		b.WriteString(" {\n")
		b.WriteString("  return *x\n")
		b.WriteString("}\n")
	}
	s.newpfuncs = nil
}

func (s *genstate) emitAssignFuncs(b *bytes.Buffer) {
	for _, fd := range s.newrfuncs {
		b.WriteString("\n//go:noinline\n")
		b.WriteString(fmt.Sprintf("func %s(", fd.name))
		fd.pp.Declare(b, "x", "", false)
		b.WriteString(", ")
		fd.p.Declare(b, "v", "", false)
		b.WriteString(") {\n")
		b.WriteString("  *x = v\n")
		b.WriteString("}\n")
	}
	s.newrfuncs = nil
}

func (s *genstate) emitGlobalVars(b *bytes.Buffer) {
	for _, fd := range s.newgvars {
		b.WriteString("\n")
		b.WriteString("var ")
		fd.pp.Declare(b, fd.name, "", false)
		b.WriteString("\n")
	}
	s.newgvars = nil
	b.WriteString("\n")
}

func (s *genstate) emitAddrTakenHelpers(b *bytes.Buffer) {
	s.emitDerefFuncs(b)
	s.emitAssignFuncs(b)
	s.emitGlobalVars(b)
}

func (s *genstate) genGvar(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "gv", "", false)
	tag := b.String()
	gv, ok := s.gvars[tag]
	if ok {
		return gv
	}
	gv = fmt.Sprintf("gvar_%d", len(s.gvars))
	s.newgvars = append(s.newgvars, funcdesc{pp: pp, p: p, name: gv})
	s.gvars[tag] = gv
	return gv
}

func (s *genstate) genParamDerefFunc(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "x", "", false)
	tag := b.String()
	f, ok := s.pfuncs[tag]
	if ok {
		return f
	}
	f = fmt.Sprintf("deref_%d", len(s.pfuncs))
	s.newpfuncs = append(s.newpfuncs, funcdesc{pp: pp, p: p, name: f})
	s.pfuncs[tag] = f
	return f
}

func (s *genstate) genAssignFunc(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "x", "", false)
	tag := b.String()
	f, ok := s.rfuncs[tag]
	if ok {
		return f
	}
	f = fmt.Sprintf("retassign_%d", len(s.rfuncs))
	s.newrfuncs = append(s.newrfuncs, funcdesc{pp: pp, p: p, name: f})
	s.rfuncs[tag] = f
	return f
}

func (s *genstate) genParamRef(p parm, idx int) string {
	switch p.AddrTaken() {
	case notAddrTaken:
		return fmt.Sprintf("p%d", idx)
	case addrTakenSimple, addrTakenHeap:
		return fmt.Sprintf("(*ap%d)", idx)
	case addrTakenPassed:
		f := s.genParamDerefFunc(p)
		return fmt.Sprintf("%s(ap%d)", f, idx)
	default:
		panic("bad")
	}
}

func (s *genstate) genReturnAssign(b *bytes.Buffer, r parm, idx int, val string) {
	switch r.AddrTaken() {
	case notAddrTaken:
		b.WriteString(fmt.Sprintf("  r%d = %s\n", idx, val))
	case addrTakenSimple, addrTakenHeap:
		b.WriteString(fmt.Sprintf("  (*ar%d) = %v\n", idx, val))
	case addrTakenPassed:
		f := s.genAssignFunc(r)
		b.WriteString(fmt.Sprintf("  %s(ar%d, %v)\n", f, idx, val))
	default:
		panic("bad")
	}
}

func (s *genstate) emitParamChecks(f *funcdef, b *bytes.Buffer, pidx int, value int) (int, bool) {
	haveControl := false
	dangling := []int{}
	for pi, p := range f.params {
		verb(4, "emitting parmcheck p%d numel=%d pt=%s value=%d",
			pi, p.NumElements(), p.TypeName(), value)
		if p.IsControl() {
			b.WriteString(fmt.Sprintf("  if %s == 0 {\n",
				s.genParamRef(p, pi)))
			s.emitReturn(f, b, false)
			b.WriteString("  }\n")
			haveControl = true
		} else if p.IsBlank() {
			var valstr string
			valstr, value = p.GenValue(value, false)
			if f.recur {
				b.WriteString(fmt.Sprintf("  brc%d := %s\n", pi, valstr))
			} else {
				b.WriteString(fmt.Sprintf("  _ = %s\n", valstr))
			}
		} else {
			numel := p.NumElements()
			cel := checkableElements(p)
			for i := 0; i < numel; i++ {
				var valstr string
				verb(4, "emitting check-code for p%d el %d value=%d", pi, i, value)
				elref, elparm := p.GenElemRef(i, s.genParamRef(p, pi))
				valstr, value = elparm.GenValue(value, false)
				if elref == "" || elref == "_" || cel == 0 {
					verb(4, "empty skip p%d el %d", pi, i)
					continue
				} else {
					b.WriteString(fmt.Sprintf("  p%df%dc := %s\n", pi, i, valstr))
					b.WriteString(fmt.Sprintf("  if %s != p%df%dc {\n", elref, pi, i))
					b.WriteString(fmt.Sprintf("    %s.NoteFailureElem(%d, \"%s\", \"parm\", %d, %d, pad[0])\n", s.utilsPkg(), f.idx, s.checkerPkg(pidx), pi, i))
					b.WriteString("    return\n")
					b.WriteString("  }\n")
				}
			}
			if cel == 0 && p.AddrTaken() != notAddrTaken {
				dangling = append(dangling, pi)
			}
		}
		if value != f.values[pi] {
			fmt.Fprintf(os.Stderr, "internal error: checker/caller value mismatch after emitting param %d func Test%d pkg %s: caller %d checker %d\n", pi, f.idx, s.checkerPkg(pidx), f.values[pi], value)
			s.errs++
		}
	}
	for _, pi := range dangling {
		b.WriteString(fmt.Sprintf("  _ = ap%d // ref\n", pi))
	}

	// receiver value check
	if f.method && !f.receiver.IsBlank() {
		numel := f.receiver.NumElements()
		for i := 0; i < numel; i++ {
			var valstr string
			verb(4, "emitting check-code for rcvr el %d value=%d", i, value)
			elref, elparm := f.receiver.GenElemRef(i, "rcvr")
			valstr, value = elparm.GenValue(value, false)
			if elref == "" || strings.HasPrefix(elref, "_") {
				verb(4, "empty skip rcvr el %d", i)
				continue
			} else {
				b.WriteString(fmt.Sprintf("  rcvrf%dc := %s\n", i, valstr))
				b.WriteString(fmt.Sprintf("  if %s != rcvrf%dc {\n", elref, i))
				b.WriteString(fmt.Sprintf("    %s.NoteFailureElem(%d, \"%s\", \"rcvr\", %d, -1, pad[0])\n", s.utilsPkg(), f.idx, s.checkerPkg(pidx), i))
				b.WriteString("    return\n")
				b.WriteString("  }\n")
			}
		}
	}

	return value, haveControl
}

// emitDeferChecks creates code like
//
//     defer func(...args...) {
//       check arg
//       check param
//     }(...)
//
// where we randomly choose to either pass a param through to the function literal,
// or have the param captured by the closure, then check its value in the defer.
func (s *genstate) emitDeferChecks(f *funcdef, b *bytes.Buffer, pidx int, value int) int {

	if len(f.params) == 0 {
		return value
	}

	// make a pass through the params and randomly decide which will be passed into the func.
	passed := []bool{}
	for range f.params {
		p := rand.Intn(100) < 50
		passed = append(passed, p)
	}

	b.WriteString("  defer func(")
	pc := 0
	for pi, p := range f.params {
		if p.IsControl() || p.IsBlank() {
			continue
		}
		if passed[pi] {
			writeCom(b, pc)
			n := fmt.Sprintf("p%d", pi)
			p.Declare(b, n, "", false)
			pc++
		}
	}
	b.WriteString(") {\n")

	for pi, p := range f.params {
		if p.IsControl() || p.IsBlank() {
			continue
		}
		which := "passed"
		if !passed[pi] {
			which = "captured"
		}
		b.WriteString("  // check parm " + which + "\n")
		numel := p.NumElements()
		cel := checkableElements(p)
		for i := 0; i < numel; i++ {
			elref, _ := p.GenElemRef(i, s.genParamRef(p, pi))
			if elref == "" || elref == "_" || cel == 0 {
				verb(4, "empty skip p%d el %d", pi, i)
				continue
			} else {
				b.WriteString(fmt.Sprintf("  if %s != p%df%dc {\n", elref, pi, i))
				b.WriteString(fmt.Sprintf("    %s.NoteFailureElem(%d, \"%s\", \"parm\", %d, %d, pad[0])\n", s.utilsPkg(), f.idx, s.checkerPkg(pidx), pi, i))
				b.WriteString("    return\n")
				b.WriteString("  }\n")
			}
		}
	}
	b.WriteString("  } (")
	pc = 0
	for pi, p := range f.params {
		if p.IsControl() || p.IsBlank() {
			continue
		}
		if passed[pi] {
			writeCom(b, pc)
			b.WriteString(fmt.Sprintf("p%d", pi))
			pc++
		}
	}
	b.WriteString(")\n\n")

	return value
}

func (s *genstate) emitChecker(f *funcdef, b *bytes.Buffer, pidx int) {
	verb(4, "emitting struct and array defs")
	emitStructAndArrayDefs(f, b)
	b.WriteString(fmt.Sprintf("// %d returns %d params\n", len(f.returns), len(f.params)))
	if s.pragma != "" {
		b.WriteString("//go:" + s.pragma + "\n")
	}
	b.WriteString("//go:noinline\n")

	b.WriteString("func")

	if f.method {
		b.WriteString(" (")
		n := "rcvr"
		if f.receiver.IsBlank() {
			n = "_"
		}
		f.receiver.Declare(b, n, "", false)
		b.WriteString(")")
	}

	b.WriteString(fmt.Sprintf(" Test%d(", f.idx))

	verb(4, "emitting checker p%d/Test%d", pidx, f.idx)

	// params
	for pi, p := range f.params {
		writeCom(b, pi)
		n := fmt.Sprintf("p%d", pi)
		if p.IsBlank() {
			n = "_"
		}
		p.Declare(b, n, "", false)
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

	// local storage
	b.WriteString("  // consume some stack space, so as to trigger morestack\n")
	b.WriteString("  var pad [256]uint64\n")
	b.WriteString(fmt.Sprintf("  pad[%s.FailCount]++\n", s.utilsPkg()))

	// generate return constants
	value := 1
	for ri, r := range f.returns {
		var valstr string
		valstr, value = r.GenValue(value, false)
		b.WriteString(fmt.Sprintf("  rc%d := %s\n", ri, valstr))
	}

	// Prepare to reference params/returns by address.
	lists := [][]parm{f.params, f.returns}
	names := []string{"p", "r"}
	var aCounts [2]int
	for i, lst := range lists {
		for pi, p := range lst {
			if p.AddrTaken() == notAddrTaken {
				continue
			}
			aCounts[i]++
			n := names[i]
			b.WriteString(fmt.Sprintf("  a%s%d := &%s%d\n", n, pi, n, pi))
			if p.AddrTaken() == addrTakenHeap {
				gv := s.genGvar(p)
				b.WriteString(fmt.Sprintf("  %s = a%s%d\n", gv, n, pi))
			}
		}
	}

	// parameter checking code
	var haveControl bool
	value, haveControl = s.emitParamChecks(f, b, pidx, value)

	// defer testing
	if s.tunables.doDefer && uint8(rand.Intn(100)) < s.tunables.deferFraction {
		value = s.emitDeferChecks(f, b, pidx, value)
	}

	// returns
	s.emitReturn(f, b, haveControl)

	b.WriteString(fmt.Sprintf("  // %d addr-taken params, %d addr-taken returns\n",
		aCounts[0], aCounts[1]))

	b.WriteString("}\n\n")

	// emit any new helper funcs referenced by this test function
	s.emitAddrTakenHelpers(b)
}

// emitRecursiveCall generates a recursive call to the test function in question.
func (s *genstate) emitRecursiveCall(f *funcdef) string {
	b := bytes.NewBuffer(nil)
	rcvr := ""
	if f.method {
		rcvr = "rcvr."
	}
	b.WriteString(fmt.Sprintf(" %sTest%d(", rcvr, f.idx))
	for pi, p := range f.params {
		writeCom(b, pi)
		if p.IsControl() {
			b.WriteString(fmt.Sprintf(" %s-1", s.genParamRef(p, pi)))
		} else {
			if !p.IsBlank() {
				b.WriteString(fmt.Sprintf(" %s", s.genParamRef(p, pi)))
			} else {
				b.WriteString(fmt.Sprintf(" brc%d", pi))
			}
		}
	}
	b.WriteString(")")
	return b.String()
}

// emitReturn generates a return sequence.
func (s *genstate) emitReturn(f *funcdef, b *bytes.Buffer, doRecursiveCall bool) {
	// If any of the return values are address-taken, then instead of
	//
	//   return x, y, z
	//
	// we emit
	//
	//   r1 = ...
	//   r2 = ...
	//   ...
	//   return
	//
	// Make an initial pass through the returns to see if we need to do this.
	// Figure out the final return values in the process.
	indirectReturn := false
	retvals := []string{}
	for ri, r := range f.returns {
		if r.AddrTaken() != notAddrTaken {
			indirectReturn = true
		}
		t := ""
		if doRecursiveCall {
			t = "t"
		}
		retvals = append(retvals, fmt.Sprintf("rc%s%d", t, ri))
	}

	// generate the recursive call itself if applicable
	if doRecursiveCall {
		b.WriteString("  // recursive call\n  ")
		rcall := s.emitRecursiveCall(f)
		if indirectReturn {
			for ri := range f.returns {
				writeCom(b, ri)
				b.WriteString(fmt.Sprintf("  rct%d", ri))
			}
			b.WriteString(" := ")
			b.WriteString(rcall)
			b.WriteString("\n")
		} else {
			if len(f.returns) == 0 {
				b.WriteString(fmt.Sprintf("%s\n  return\n", rcall))
			} else {
				b.WriteString(fmt.Sprintf("  return %s\n", rcall))
			}
			return
		}
	}

	// now the actual return
	if indirectReturn {
		for ri, r := range f.returns {
			s.genReturnAssign(b, r, ri, retvals[ri])
		}
		b.WriteString("  return\n")
	} else {
		b.WriteString("  return ")
		for ri := range f.returns {
			writeCom(b, ri)
			b.WriteString(retvals[ri])
		}
		b.WriteString("\n")
	}
}

func (s *genstate) GenPair(calloutfile *os.File, checkoutfile *os.File, fidx int, pidx int, b *bytes.Buffer, seed int64, emit bool) int64 {

	verb(1, "gen fidx %d pidx %d", fidx, pidx)

	checkTunables(tunables)
	s.tunables = tunables

	// Generate a function with a random number of params and returns
	fp := s.GenFunc(fidx, pidx)

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
		if imp == "reflect" {
			outf.WriteString("import \"reflect\"\n")
			continue
		}
		outf.WriteString(fmt.Sprintf("import \"%s%s\"\n", ipref, imp))
	}
	outf.WriteString("\n")
	return outf
}

func emitUtils(outf *os.File) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "var FailCount int\n\n")
	fmt.Fprintf(outf, "type UtilsType int\n\n")
	fmt.Fprintf(outf, "//go:noinline\n")
	fmt.Fprintf(outf, "func NoteFailure(fidx int, pkg string, pref string, parmNo int, _ uint64) {\n")
	fmt.Fprintf(outf, "  FailCount += 1\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail on =%%s.Test%%d= %%s %%d\\n\", pkg, fidx, pref, parmNo)\n")
	fmt.Fprintf(outf, "  if (FailCount > 10) {\n")
	fmt.Fprintf(outf, "    os.Exit(1)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "}\n\n")
	fmt.Fprintf(outf, "//go:noinline\n")
	fmt.Fprintf(outf, "func NoteFailureElem(fidx int, pkg string, pref string, parmNo int, elem int, _ uint64) {\n")
	fmt.Fprintf(outf, "  FailCount += 1\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail on =%%s.Test%%d= %%s %%d elem %%d\\n\", pkg, fidx, pref, parmNo, elem)\n")
	fmt.Fprintf(outf, "  if (FailCount > 10) {\n")
	fmt.Fprintf(outf, "    os.Exit(1)\n")
	fmt.Fprintf(outf, "  }\n")
	fmt.Fprintf(outf, "}\n\n")
}

func (s *genstate) emitMain(outf *os.File, numit int, fcnmask map[int]int, pkmask map[int]int) {
	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "func main() {\n")
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, \"starting main\\n\")\n")
	for k := 0; k < s.numtpk; k++ {
		cp := fmt.Sprintf("%sCaller%d", s.tag, k)
		for i := 0; i < numit; i++ {
			if emitFP(i, k, fcnmask, pkmask) {
				fmt.Fprintf(outf, "  %s.Caller%d()\n", cp, i)
			}
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

func emitFP(fn int, pk int, fcnmask map[int]int, pkmask map[int]int) bool {
	emitpk := true
	emitfn := true
	if len(pkmask) != 0 {
		emitpk = false
		if _, ok := pkmask[pk]; ok {
			emitpk = true
		}
	}
	if len(fcnmask) != 0 {
		emitfn = false
		if _, ok := fcnmask[fn]; ok {
			emitfn = true
		}
	}
	doemit := emitpk && emitfn
	verb(2, "emitFP(F=%d,P=%d) returns %v", fn, pk, doemit)
	return doemit
}

func Generate(tag string, outdir string, pkgpath string, numit int, numtpkgs int, seed int64, pragma string, fcnmask map[int]int, pkmask map[int]int, utilsinl bool) int {
	mainpkg := tag + "Main"

	var ipref string
	if len(pkgpath) > 0 {
		ipref = pkgpath + "/"
	}

	s := genstate{
		outdir: outdir,
		ipref:  ipref,
		tag:    tag,
		numtpk: numtpkgs,
		pragma: pragma,
	}

	if outdir != "." {
		makeDir(outdir)
	}
	verb(1, "creating %s", outdir)

	mainimports := []string{}
	for i := 0; i < numtpkgs; i++ {
		makeDir(outdir + "/" + s.callerPkg(i))
		makeDir(outdir + "/" + s.checkerPkg(i))
		makeDir(outdir + "/" + s.utilsPkg())
		if emitFP(-1, i, nil, pkmask) {
			mainimports = append(mainimports, s.callerPkg(i))
		}
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
		callerImports := []string{s.checkerPkg(k), s.utilsPkg()}
		if tunables.doReflectCall {
			callerImports = append(callerImports, "reflect")
		}
		calleroutfile := openOutputFile(s.callerFile(k), s.callerPkg(k),
			callerImports, ipref)
		checkeroutfile := openOutputFile(s.checkerFile(k), s.checkerPkg(k),
			[]string{s.utilsPkg()}, ipref)

		s.newpfuncs = nil
		s.newrfuncs = nil
		s.newgvars = nil
		s.pfuncs = make(map[string]string)
		s.rfuncs = make(map[string]string)
		s.gvars = make(map[string]string)

		var b bytes.Buffer
		for i := 0; i < numit; i++ {
			doemit := emitFP(i, k, fcnmask, pkmask)
			seed = s.GenPair(calleroutfile, checkeroutfile, i, k,
				&b, seed, doemit)
		}

		// When minimization is in effect, we sometimes wind up eliminating
		// all refs to the utils package. Add a dummy to help with this.
		fmt.Fprintf(calleroutfile, "\n// dummy\nvar Dummy %s.UtilsType\n", s.utilsPkg())
		fmt.Fprintf(checkeroutfile, "\n// dummy\nvar Dummy %s.UtilsType\n", s.utilsPkg())
		calleroutfile.Close()
		checkeroutfile.Close()
	}
	s.emitMain(mainoutfile, numit, fcnmask, pkmask)

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
	return s.errs
}
