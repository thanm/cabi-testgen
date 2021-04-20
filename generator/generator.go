package generator

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
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

	// arrays/slices have between 0 and N elements
	nArrayElements uint8

	// fraction of slices vs arrays
	sliceFraction uint8

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
	// struct/array/map/pointer/int/float/complex/byte/string at the
	// top level. If nesting precludes using a struct, other types
	// are chosen from instead according to same proportions.
	typeFractions [9]uint8

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

	// If true, randomly pick between emitting a value by literal
	// (e.g. "int(1)" vs emitting a call to a function that
	// will produce the same value (e.g. "myHelperEmitsInt1()").
	doFuncCallValues bool

	// Fraction of the time that we emit a function call to create
	// a param value vs emitting a literal.
	funcCallValFraction uint8
}

var defaultTypeFractions = [9]uint8{
	10, // struct
	10, // array
	10, // map
	15, // pointer
	20, // numeric
	15, // float
	5,  // complex
	5,  // byte
	10, // string
}

type typeFractionIndex uint8

const (
	// Param not address taken.
	StructTfIdx = iota
	ArrayTfIdx
	MapTfIdx
	PointerTfIdx
	NumericTfIdx
	FloatTfIdx
	ComplexTfIdx
	ByteTfIdx
	StringTfIdx
)

var tunables = TunableParams{
	nParmRange:            15,
	nReturnRange:          7,
	nStructFields:         7,
	nArrayElements:        5,
	sliceFraction:         50,
	intBitRanges:          [4]uint8{30, 20, 20, 30},
	floatBitRanges:        [2]uint8{50, 50},
	unsignedRanges:        [2]uint8{50, 50},
	blankPerc:             15,
	structDepth:           3,
	typeFractions:         defaultTypeFractions,
	recurPerc:             20,
	methodPerc:            10,
	pointerMethodCallPerc: 50,
	doReflectCall:         true,
	doDefer:               true,
	takeAddress:           true,
	doFuncCallValues:      true,
	takenFraction:         20,
	deferFraction:         30,
	funcCallValFraction:   5,
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
		panic(errors.New("typeFractions tunable does not sum to 100"))
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
	if t.sliceFraction > 100 {
		log.Fatal(errors.New("sliceFraction not between 0 and 100"))
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
	idx         int
	structdefs  []structparm
	arraydefs   []arrayparm
	typedefs    []typedefparm
	mapdefs     []mapparm
	mapkeytypes []parm
	mapkeytmps  []string
	mapkeyts    string
	receiver    parm
	params      []parm
	returns     []parm
	values      []int
	dodefc      uint8
	dodefp      []uint8
	rstack      int
	recur       bool
	method      bool
}

type genstate struct {
	outdir         string
	ipref          string
	tag            string
	numtpk         int
	pkidx          int
	errs           int
	pragma         string
	sforce         bool
	randctl        int
	tunables       TunableParams
	tstack         []TunableParams
	derefFuncs     map[string]string
	newDerefFuncs  []funcdesc
	assignFuncs    map[string]string
	newAssignFuncs []funcdesc
	allocFuncs     map[string]string
	newAllocFuncs  []funcdesc
	genvalFuncs    map[string]string
	newGenvalFuncs []funcdesc
	globVars       map[string]string
	newGlobVars    []funcdesc
	wr             *wraprand
}

func (s *genstate) intFlavor() string {
	which := uint8(s.wr.Intn(100))
	if which < s.tunables.unsignedRanges[0] {
		return "uint"
	}
	return "int"
}

func (s *genstate) intBits() uint32 {
	which := uint8(s.wr.Intn(100))
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
	which := uint8(s.wr.Intn(100))
	if which < s.tunables.floatBitRanges[0] {
		return uint32(32)
	}
	return uint32(64)
}

func (s *genstate) genAddrTaken() addrTakenHow {
	which := uint8(s.wr.Intn(100))
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

func (s *genstate) dumpTypeFraction(tag string) {
	fmt.Fprintf(os.Stderr, "type fractions at %s:\n", tag)
	sum := uint8(0)
	d := func(sl int, tag string) {
		amt := s.tunables.typeFractions[sl]
		sum += amt
		fmt.Fprintf(os.Stderr, "%10s: %d\n", tag, amt)
	}
	d(StructTfIdx, "struct")
	d(ArrayTfIdx, "array")
	d(MapTfIdx, "map")
	d(PointerTfIdx, "pointer")
	d(NumericTfIdx, "numeric")
	d(FloatTfIdx, "float")
	d(ComplexTfIdx, "complex")
	d(ByteTfIdx, "byte")
	d(StringTfIdx, "string")
	fmt.Fprintf(os.Stderr, "sum: %d\n", sum)
}

func (s *genstate) redistributeFraction(f uint8, avoid []int) {
	inavoid := func(j int) bool {
		for _, k := range avoid {
			if j == k {
				return true
			}
		}
		return false
	}

	doredis := func() {
		for {
			for i := range s.tunables.typeFractions {
				if inavoid(i) {
					continue
				}
				s.tunables.typeFractions[i]++
				f--
				if f == 0 {
					return
				}
			}
		}
	}
	doredis()
	checkTunables(s.tunables)
}

func (s *genstate) precludeSelectedTypes(t int, t2 ...int) {
	avoid := []int{t}
	avoid = append(avoid, t2...)
	f := uint8(0)
	for _, idx := range avoid {
		f += s.tunables.typeFractions[idx]
		s.tunables.typeFractions[idx] = 0
	}
	s.redistributeFraction(f, avoid)
}

func (s *genstate) GenMapKeyType(f *funcdef, depth int, pidx int) parm {
	s.pushTunables()
	defer s.popTunables()
	// maps we can't allow at all; pointers might be possible but
	//  would be too much work to arrange. Avoid slices as well.
	s.tunables.sliceFraction = 0
	s.precludeSelectedTypes(MapTfIdx, PointerTfIdx)
	return s.GenParm(f, depth+1, false, pidx)
}

func (s *genstate) GenParm(f *funcdef, depth int, mkctl bool, pidx int) parm {

	// Enforcement for struct/array/map/pointer array nesting depth.
	toodeep := depth >= int(s.tunables.structDepth)
	if toodeep {
		s.pushTunables()
		defer s.popTunables()
		s.precludeSelectedTypes(StructTfIdx, ArrayTfIdx, MapTfIdx, PointerTfIdx)
	}

	// Convert tf into a cumulative sum
	tf := s.tunables.typeFractions
	sum := uint8(0)
	for i := 0; i < len(tf); i++ {
		sum += tf[i]
		tf[i] = sum
	}

	isblank := uint8(s.wr.Intn(100)) < s.tunables.blankPerc
	addrTaken := notAddrTaken
	if depth == 0 && tunables.takeAddress && !isblank {
		addrTaken = s.genAddrTaken()
	}
	isGenValFunc := tunables.doFuncCallValues &&
		uint8(s.wr.Intn(100)) < s.tunables.funcCallValFraction

	// Make adjusted selection (pick a bucket within tf)
	which := uint8(s.wr.Intn(100))
	verb(3, "which=%d", which)
	var retval parm
	switch {
	case which < tf[StructTfIdx]:
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
			nf := s.wr.Intn(tnf)
			for fi := 0; fi < nf; fi++ {
				fp := s.GenParm(f, depth+1, false, pidx)
				sp.fields = append(sp.fields, fp)
			}
			f.structdefs[ns] = sp
			retval = &sp
		}
	case which < tf[ArrayTfIdx]:
		{
			if toodeep {
				panic("should not be here")
			}
			var ap arrayparm
			ns := len(f.arraydefs)
			nel := uint8(s.wr.Intn(int(s.tunables.nArrayElements)))
			issl := uint8(s.wr.Intn(100)) < s.tunables.sliceFraction
			ap.aname = fmt.Sprintf("ArrayF%dS%dE%d", f.idx, ns, nel)
			ap.qname = fmt.Sprintf("%s.ArrayF%dS%dE%d", s.checkerPkg(pidx),
				f.idx, ns, nel)
			f.arraydefs = append(f.arraydefs, ap)
			ap.nelements = nel
			ap.slice = issl
			ap.eltype = s.GenParm(f, depth+1, false, pidx)
			ap.eltype.SetBlank(false)
			f.arraydefs[ns] = ap
			retval = &ap
		}
	case which < tf[MapTfIdx]:
		{
			if toodeep {
				panic("should not be here")
			}
			var mp mapparm
			ns := len(f.mapdefs)

			// append early, since calls below might also append
			f.mapdefs = append(f.mapdefs, mp)
			f.mapkeytmps = append(f.mapkeytmps, "")
			f.mapkeytypes = append(f.mapkeytypes, mp.keytype)
			mp.aname = fmt.Sprintf("MapF%dM%d", f.idx, ns)
			if f.mapkeyts == "" {
				f.mapkeyts = fmt.Sprintf("MapKeysF%d", f.idx)
			}
			mp.qname = fmt.Sprintf("%s.MapF%dM%d", s.checkerPkg(pidx),
				f.idx, ns)
			mkt := fmt.Sprintf("Mk%dt%d", f.idx, ns)
			mp.keytmp = mkt
			mk := s.GenMapKeyType(f, depth+1, pidx)
			mp.keytype = mk
			mp.valtype = s.GenParm(f, depth+1, false, pidx)
			mp.valtype.SetBlank(false)
			mp.keytype.SetBlank(false)
			// now update the previously appended placeholders
			f.mapdefs[ns] = mp
			f.mapkeytypes[ns] = mk
			f.mapkeytmps[ns] = mkt
			retval = &mp
		}
	case which < tf[PointerTfIdx]:
		{
			if toodeep {
				panic("should not be here")
			}
			pp := mkPointerParm(s.GenParm(f, depth+1, false, pidx))
			retval = &pp
		}
	case which < tf[NumericTfIdx]:
		{
			var ip numparm
			ip.tag = s.intFlavor()
			ip.widthInBits = s.intBits()
			if mkctl {
				ip.ctl = true
			}
			retval = &ip
		}
	case which < tf[FloatTfIdx]:
		{
			var fp numparm
			fp.tag = "float"
			fp.widthInBits = s.floatBits()
			retval = &fp
		}
	case which < tf[ComplexTfIdx]:
		{
			var fp numparm
			fp.tag = "complex"
			fp.widthInBits = s.floatBits() * 2
			retval = &fp
		}
	case which < tf[ByteTfIdx]:
		{
			var bp numparm
			bp.tag = "byte"
			bp.widthInBits = 8
			retval = &bp
		}
	case which < tf[StringTfIdx]:
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
	retval.SetIsGenVal(isGenValFunc)
	return retval
}

func (s *genstate) GenReturn(f *funcdef, depth int, pidx int) parm {
	return s.GenParm(f, depth, false, pidx)
}

func (s *genstate) GenFunc(fidx int, pidx int) *funcdef {
	f := new(funcdef)
	f.idx = fidx
	numParams := s.wr.Intn(1 + int(s.tunables.nParmRange))
	numReturns := s.wr.Intn(1 + int(s.tunables.nReturnRange))
	f.recur = uint8(s.wr.Intn(100)) < s.tunables.recurPerc
	f.method = uint8(s.wr.Intn(100)) < s.tunables.methodPerc
	if f.method {
		// Receiver type can't be pointer type. Temporarily update
		// tunables to eliminate that possibility.
		s.pushTunables()
		s.precludeSelectedTypes(PointerTfIdx)
		target := s.GenParm(f, 0, false, pidx)
		target.SetBlank(false)
		s.popTunables()
		f.receiver = s.makeTypedefParm(f, target, pidx)
		if f.receiver.IsBlank() {
			f.recur = false
		}
	}
	needControl := f.recur
	f.dodefc = uint8(s.wr.Intn(100))
	pTaken := uint8(s.wr.Intn(100)) < s.tunables.takenFraction
	for pi := 0; pi < numParams; pi++ {
		newparm := s.GenParm(f, 0, needControl, pidx)
		if !pTaken {
			newparm.SetAddrTaken(notAddrTaken)
		}
		if newparm.IsControl() {
			needControl = false
		}
		f.params = append(f.params, newparm)
		f.dodefp = append(f.dodefp, uint8(s.wr.Intn(100)))
	}
	if f.recur && needControl {
		f.recur = false
	}

	rTaken := uint8(s.wr.Intn(100)) < s.tunables.takenFraction
	for ri := 0; ri < numReturns; ri++ {
		r := s.GenReturn(f, 0, pidx)
		if !rTaken {
			r.SetAddrTaken(notAddrTaken)
		}
		f.returns = append(f.returns, r)
	}
	spw := uint(s.wr.Intn(11))
	rstack := 1 << spw
	if rstack < 4 {
		rstack = 4
	}
	f.rstack = rstack
	return f
}

func genDeref(p parm) (parm, string) {
	curp := p
	star := ""
	for {
		if pp, ok := curp.(*pointerparm); ok {
			star += "*"
			curp = pp.totype
		} else {
			return curp, star
		}
	}
}

func (s *genstate) eqFuncRef(f *funcdef, t parm, caller bool) string {
	cp := ""
	if f.mapkeyts != "" {
		cp = "mkt."
	} else if caller {
		cp = s.checkerPkg(s.pkidx) + "."
	}
	return cp + "Equal" + t.TypeName()
}

func (s *genstate) emitCompareFunc(f *funcdef, b *bytes.Buffer, p parm) {
	if !p.HasPointer() {
		return
	}

	tn := p.TypeName()
	b.WriteString(fmt.Sprintf("// equal func for %s\n", tn))
	b.WriteString("//go:noinline\n")
	rcvr := ""
	if f.mapkeyts != "" {
		rcvr = fmt.Sprintf("(mkt *%s) ", f.mapkeyts)
	}
	b.WriteString(fmt.Sprintf("func %sEqual%s(left %s, right %s) bool {\n", rcvr, tn, tn, tn))
	b.WriteString("  return ")
	numel := p.NumElements()
	ncmp := 0
	for i := 0; i < numel; i++ {
		lelref, lelparm := p.GenElemRef(i, "left")
		relref, _ := p.GenElemRef(i, "right")
		if lelref == "" || lelref == "_" {
			continue
		}
		basep, star := genDeref(lelparm)
		// Handle *p where p is an empty struct.
		if basep.NumElements() == 0 {
			continue
		}
		if ncmp != 0 {
			b.WriteString("  && ")
		}
		ncmp++
		if basep.HasPointer() {
			efn := s.eqFuncRef(f, basep, false)
			b.WriteString(fmt.Sprintf(" %s(%s%s, %s%s)", efn, star, lelref, star, relref))
		} else {
			b.WriteString(fmt.Sprintf("%s%s == %s%s", star, lelref, star, relref))
		}
	}
	if ncmp == 0 {
		b.WriteString("true")
	}
	b.WriteString("\n}\n\n")
}

func (s *genstate) emitStructAndArrayDefs(f *funcdef, b *bytes.Buffer) {
	for _, str := range f.structdefs {
		b.WriteString(fmt.Sprintf("type %s struct {\n", str.sname))
		for fi, sp := range str.fields {
			sp.Declare(b, "  "+str.FieldName(fi), "\n", false)
		}
		b.WriteString("}\n\n")
		s.emitCompareFunc(f, b, &str)
	}
	for _, a := range f.arraydefs {
		elems := fmt.Sprintf("%d", a.nelements)
		if a.slice {
			elems = ""
		}
		b.WriteString(fmt.Sprintf("type %s [%s]%s\n\n", a.aname,
			elems, a.eltype.TypeName()))
		s.emitCompareFunc(f, b, &a)
	}
	for _, a := range f.mapdefs {
		b.WriteString(fmt.Sprintf("type %s map[%s]%s\n\n", a.aname,
			a.keytype.TypeName(), a.valtype.TypeName()))
		s.emitCompareFunc(f, b, &a)
	}
	for _, td := range f.typedefs {
		b.WriteString(fmt.Sprintf("type %s %s\n\n", td.aname,
			td.target.TypeName()))
		s.emitCompareFunc(f, b, &td)
	}
	if f.mapkeyts != "" {
		b.WriteString(fmt.Sprintf("type %s struct {\n", f.mapkeyts))
		for i := range f.mapkeytypes {
			f.mapkeytypes[i].Declare(b, "  "+f.mapkeytmps[i], "\n", false)
		}
		b.WriteString("}\n\n")
	}
}

// GenValue method of genstate wraps the parm method of the same
// name, but optionally returns a call to a function to produce
// the value as opposed to a literal value.
func (s *genstate) GenValue(f *funcdef, p parm, value int, caller bool) (string, int) {
	var valstr string
	valstr, value = p.GenValue(s, f, value, caller)
	if !s.tunables.doFuncCallValues || !p.IsGenVal() || caller {
		return valstr, value
	}

	mkInvoc := func(fname string) string {
		meth := ""
		if f.mapkeyts != "" {
			meth = "mkt."
		}
		return fmt.Sprintf("%s%s()", meth, fname)
	}

	b := bytes.NewBuffer(nil)
	p.Declare(b, "x", "", false)
	h := sha1.New()
	h.Write([]byte(valstr))
	h.Write([]byte(b.String()))
	if f.mapkeyts != "" {
		h.Write([]byte(f.mapkeyts))
	}
	h.Write([]byte(b.String()))
	bs := h.Sum(nil)
	hashstr := fmt.Sprintf("%x", bs)
	b.WriteString(hashstr)
	tag := b.String()
	fname, ok := s.genvalFuncs[tag]
	if ok {
		return mkInvoc(fname), value
	}

	fname = fmt.Sprintf("genval_%d", len(s.genvalFuncs))
	s.newGenvalFuncs = append(s.newGenvalFuncs, funcdesc{p: p, name: fname, tag: tag, payload: valstr})
	s.genvalFuncs[tag] = fname
	return mkInvoc(fname), value
}

func (s *genstate) emitMapKeyTmps(f *funcdef, b *bytes.Buffer, pidx int, value int, caller bool) int {
	if f.mapkeyts == "" {
		return value
	}
	// map key tmps
	cp := ""
	if caller {
		cp = s.checkerPkg(pidx) + "."
	}
	b.WriteString("  var mkt " + cp + f.mapkeyts + "\n")
	for i, t := range f.mapkeytypes {
		var keystr string
		keystr, value = s.GenValue(f, t, value, caller)
		tname := f.mapkeytmps[i]
		b.WriteString(fmt.Sprintf("  %s := %s\n", tname, keystr))
		b.WriteString(fmt.Sprintf("  mkt.%s = %s\n", tname, tname))
	}
	return value
}

func (s *genstate) emitCaller(f *funcdef, b *bytes.Buffer, pidx int) {

	b.WriteString(fmt.Sprintf("func Caller%d(mode string) {\n", f.idx))

	b.WriteString(fmt.Sprintf("  %s.BeginFcn()\n", s.utilsPkg()))

	var value int = 1

	s.wr.Checkpoint("before mapkeytmps")
	value = s.emitMapKeyTmps(f, b, pidx, value, true)

	// generate return constants
	s.wr.Checkpoint("before return constants")
	for ri, r := range f.returns {
		rc := fmt.Sprintf("c%d", ri)
		value = s.emitVarAssign(f, b, r, rc, value, true)
	}

	// generate param constants
	s.wr.Checkpoint("before param constants")
	for pi, p := range f.params {
		verb(4, "emitCaller gen p%d value=%d", pi, value)
		if p.IsControl() {
			_ = uint8(s.wr.Intn(100)) < 50
			p.Declare(b, fmt.Sprintf("  var p%d ", pi), " = 10\n", true)
		} else {
			pc := fmt.Sprintf("p%d", pi)
			value = s.emitVarAssign(f, b, p, pc, value, true)
		}
		f.values = append(f.values, value)
	}

	// generate receiver constant if applicable
	if f.method {
		s.wr.Checkpoint("before receiver constant")
		f.receiver.Declare(b, "  var rcvr", "\n", true)
		valstr, value := s.GenValue(f, f.receiver, value, true)
		b.WriteString(fmt.Sprintf("  rcvr = %s\n", valstr))
		f.values = append(f.values, value)
	}

	b.WriteString(fmt.Sprintf("  %s.Mode = \"\"\n", s.utilsPkg()))

	// calling code
	b.WriteString(fmt.Sprintf("  // %d returns %d params\n",
		len(f.returns), len(f.params)))
	if s.sforce {
		b.WriteString("  hackStack() // force stack growth on next call\n")
	}
	b.WriteString("  if mode == \"normal\" {\n")
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
	cm := f.complexityMeasure()
	for ri, rp := range f.returns {
		star := ""
		pfc := ""
		curp, star := genDeref(rp)
		// Handle *p where p is an empty struct.
		if curp.NumElements() == 0 {
			b.WriteString(fmt.Sprintf("  _, _ = r%d, c%d // zero size\n", ri, ri))
			continue
		}
		if star != "" {
			pfc = fmt.Sprintf("%s.ParamFailCount == 0 && ", s.utilsPkg())
		}
		if curp.HasPointer() {
			efn := "!" + s.eqFuncRef(f, curp, true)
			b.WriteString(fmt.Sprintf("  if %s%s(%sr%d, %sc%d) {\n", pfc, efn, star, ri, star, ri))
		} else {
			b.WriteString(fmt.Sprintf("  if %s%sr%d != %sc%d {\n", pfc, star, ri, star, ri))
		}
		b.WriteString(fmt.Sprintf("    %s.NoteFailure(%d, %d, %d, \"%s\", \"return\", %d, true, uint64(0))\n", s.utilsPkg(), cm, pidx, f.idx, s.checkerPkg(pidx), ri))
		b.WriteString("  }\n")
	}
	b.WriteString("  }")
	if s.tunables.doReflectCall {
		b.WriteString("else {\n")
		// now make the same call via reflection
		b.WriteString("  // same call via reflection\n")
		b.WriteString(fmt.Sprintf("  %s.Mode = \"reflect\"\n", s.utilsPkg()))
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

			star := ""
			pfc := ""
			curp, star := genDeref(r)
			// Handle *p where p is an empty struct.
			if curp.NumElements() == 0 {
				b.WriteString(fmt.Sprintf("  _, _ = rr%dv, c%d // zero size\n", ri, ri))
				continue
			}
			if star != "" {
				pfc = fmt.Sprintf("%s.ParamFailCount == 0 && ", s.utilsPkg())
			}
			if curp.HasPointer() {
				efn := "!" + s.eqFuncRef(f, curp, true)
				b.WriteString(fmt.Sprintf("  if %s%s(%srr%dv, %sc%d) {\n", pfc, efn, star, ri, star, ri))
			} else {
				b.WriteString(fmt.Sprintf("  if %s%srr%dv != %sc%d {\n", pfc, star, ri, star, ri))
			}
			b.WriteString(fmt.Sprintf("    %s.NoteFailure(%d, %d, %d, \"%s\", \"reflect return\", %d, true, uint64(0))\n", s.utilsPkg(), cm, pidx, f.idx, s.checkerPkg(pidx), ri))
			b.WriteString("  }\n")
		}
		b.WriteString("}\n")
	}

	b.WriteString(fmt.Sprintf("\n  %s.EndFcn()\n", s.utilsPkg()))

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

// funcdesc describes an auto-generated helper function or global
// variable, such as an allocation function (returns new(T)) or a
// pointer assignment function (assigns value of T to type *T). Here
// 'p' is a param type T, 'pp' is a pointer type *T, 'name' is the
// name within the generated code of the function or variable and
// 'tag' is a descriptive tag used to look up the entity in a map (so
// that we don't have to emit multiple copies of a function that
// assigns int to *int, for exampkle).
type funcdesc struct {
	p       parm
	pp      parm
	name    string
	tag     string
	payload string
}

func (s *genstate) emitDerefFuncs(b *bytes.Buffer, emit bool) {
	b.WriteString("// dereference helpers\n")
	for _, fd := range s.newDerefFuncs {
		if !emit {
			b.WriteString(fmt.Sprintf("\n// skip derefunc %s\n", fd.name))
			delete(s.derefFuncs, fd.tag)
			continue
		}
		b.WriteString("\n//go:noinline\n")
		b.WriteString(fmt.Sprintf("func %s(", fd.name))
		fd.pp.Declare(b, "x", "", false)
		b.WriteString(") ")
		fd.p.Declare(b, "", "", false)
		b.WriteString(" {\n")
		b.WriteString("  return *x\n")
		b.WriteString("}\n")
	}
	s.newDerefFuncs = nil
}

func (s *genstate) emitAssignFuncs(b *bytes.Buffer, emit bool) {
	b.WriteString("// assign helpers\n")
	for _, fd := range s.newAssignFuncs {
		if !emit {
			b.WriteString(fmt.Sprintf("\n// skip assignfunc %s\n", fd.name))
			delete(s.assignFuncs, fd.tag)
			continue
		}
		b.WriteString("\n//go:noinline\n")
		b.WriteString(fmt.Sprintf("func %s(", fd.name))
		fd.pp.Declare(b, "x", "", false)
		b.WriteString(", ")
		fd.p.Declare(b, "v", "", false)
		b.WriteString(") {\n")
		b.WriteString("  *x = v\n")
		b.WriteString("}\n")
	}
	s.newAssignFuncs = nil
}

func (s *genstate) emitNewFuncs(b *bytes.Buffer, emit bool) {
	b.WriteString("// 'new' funcs\n")
	for _, fd := range s.newAllocFuncs {
		if !emit {
			b.WriteString(fmt.Sprintf("\n// skip newfunc %s\n", fd.name))
			delete(s.allocFuncs, fd.tag)
			continue
		}
		b.WriteString("\n//go:noinline\n")
		b.WriteString(fmt.Sprintf("func %s(", fd.name))
		fd.p.Declare(b, "i", "", false)
		b.WriteString(") ")
		fd.pp.Declare(b, "", "", false)
		b.WriteString(" {\n")
		b.WriteString("  x := new(")
		fd.p.Declare(b, "", "", false)
		b.WriteString(")\n")
		b.WriteString("  *x = i\n")
		b.WriteString("  return x\n")
		b.WriteString("}\n\n")
	}
	s.newAllocFuncs = nil
}

func (s *genstate) emitGlobalVars(b *bytes.Buffer, emit bool) {
	b.WriteString("// global vars\n")
	for _, fd := range s.newGlobVars {
		if !emit {
			b.WriteString(fmt.Sprintf("\n// skip gvar %s\n", fd.name))
			delete(s.globVars, fd.tag)
			continue
		}
		b.WriteString("var ")
		fd.pp.Declare(b, fd.name, "", false)
		b.WriteString("\n")
	}
	s.newGlobVars = nil
	b.WriteString("\n")
}

func (s *genstate) emitGenValFuncs(f *funcdef, b *bytes.Buffer, emit bool) {
	b.WriteString("// genval helpers\n")
	for _, fd := range s.newGenvalFuncs {
		if !emit {
			b.WriteString(fmt.Sprintf("\n// skip genvalfunc %s\n", fd.name))
			delete(s.genvalFuncs, fd.tag)
			continue
		}
		b.WriteString("\n//go:noinline\n")
		rcvr := ""
		if f.mapkeyts != "" {
			rcvr = fmt.Sprintf("(mkt *%s) ", f.mapkeyts)
		}
		b.WriteString(fmt.Sprintf("func %s%s() ", rcvr, fd.name))
		fd.p.Declare(b, "", "", false)
		b.WriteString(" {\n")
		if f.mapkeyts != "" {
			contained := containedParms(fd.p)
			for _, cp := range contained {
				mp, ismap := cp.(*mapparm)
				if ismap {
					b.WriteString(fmt.Sprintf("  %s := mkt.%s\n",
						mp.keytmp, mp.keytmp))
				}
			}
		}
		b.WriteString(fmt.Sprintf("  return %s\n", fd.payload))
		b.WriteString("}\n")
	}
	s.newGenvalFuncs = nil
}

func (s *genstate) emitAddrTakenHelpers(f *funcdef, b *bytes.Buffer, emit bool) {
	b.WriteString("// begin addr taken helpers\n")
	s.emitDerefFuncs(b, emit)
	s.emitAssignFuncs(b, emit)
	s.emitNewFuncs(b, emit)
	s.emitGlobalVars(b, emit)
	s.emitGenValFuncs(f, b, emit)
	b.WriteString("// end addr taken helpers\n")
}

func (s *genstate) genGlobVar(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "gv", "", false)
	tag := b.String()
	gv, ok := s.globVars[tag]
	if ok {
		return gv
	}
	gv = fmt.Sprintf("gvar_%d", len(s.globVars))
	s.newGlobVars = append(s.newGlobVars, funcdesc{pp: pp, p: p, name: gv, tag: tag})
	s.globVars[tag] = gv
	return gv
}

func (s *genstate) genParamDerefFunc(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "x", "", false)
	tag := b.String()
	f, ok := s.derefFuncs[tag]
	if ok {
		return f
	}
	f = fmt.Sprintf("deref_%d", len(s.derefFuncs))
	s.newDerefFuncs = append(s.newDerefFuncs, funcdesc{pp: pp, p: p, name: f, tag: tag})
	s.derefFuncs[tag] = f
	return f
}

func (s *genstate) genAssignFunc(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "x", "", false)
	tag := b.String()
	f, ok := s.assignFuncs[tag]
	if ok {
		return f
	}
	f = fmt.Sprintf("retassign_%d", len(s.assignFuncs))
	s.newAssignFuncs = append(s.newAssignFuncs, funcdesc{pp: pp, p: p, name: f, tag: tag})
	s.assignFuncs[tag] = f
	return f
}

func (s *genstate) genAllocFunc(p parm) string {
	var pp parm
	ppp := mkPointerParm(p)
	pp = &ppp
	b := bytes.NewBuffer(nil)
	pp.Declare(b, "x", "", false)
	tag := b.String()
	f, ok := s.allocFuncs[tag]
	if ok {
		return f
	}
	f = fmt.Sprintf("New_%d", len(s.allocFuncs))
	s.newAllocFuncs = append(s.newAllocFuncs, funcdesc{pp: pp, p: p, name: f, tag: tag})
	s.allocFuncs[tag] = f
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

func (s *genstate) emitParamElemCheck(f *funcdef, b *bytes.Buffer, p parm, pvar string, cvar string, paramidx int, elemidx int) {
	basep, star := genDeref(p)
	// Handle *p where p is an empty struct.
	if basep.NumElements() == 0 {
		return
	}
	if basep.HasPointer() {
		efn := s.eqFuncRef(f, basep, false)
		b.WriteString(fmt.Sprintf("  if !%s(%s%s, %s%s) {\n", efn, star, pvar, star, cvar))
	} else {
		b.WriteString(fmt.Sprintf("  if %s%s != %s%s {\n", star, pvar, star, cvar))
	}
	cm := f.complexityMeasure()
	b.WriteString(fmt.Sprintf("    %s.NoteFailureElem(%d, %d, %d, \"%s\", \"parm\", %d, %d, false, pad[0])\n", s.utilsPkg(), cm, s.pkidx, f.idx, s.checkerPkg(s.pkidx), paramidx, elemidx))
	b.WriteString("    return\n")
	b.WriteString("  }\n")
}

func (s *genstate) emitParamChecks(f *funcdef, b *bytes.Buffer, pidx int, value int) (int, bool) {
	var valstr string
	haveControl := false
	dangling := []int{}
	for pi, p := range f.params {
		verb(4, "emitting parmcheck p%d numel=%d pt=%s value=%d",
			pi, p.NumElements(), p.TypeName(), value)
		// To balance code in caller
		_ = uint8(s.wr.Intn(100)) < 50
		if p.IsControl() {
			b.WriteString(fmt.Sprintf("  if %s == 0 {\n",
				s.genParamRef(p, pi)))
			s.emitReturn(f, b, false)
			b.WriteString("  }\n")
			haveControl = true

		} else if p.IsBlank() {
			valstr, value = s.GenValue(f, p, value, false)
			if f.recur {
				b.WriteString(fmt.Sprintf("  brc%d := %s\n", pi, valstr))
			} else {
				b.WriteString(fmt.Sprintf("  _ = %s\n", valstr))
			}
		} else {
			numel := p.NumElements()
			cel := checkableElements(p)
			for i := 0; i < numel; i++ {
				verb(4, "emitting check-code for p%d el %d value=%d", pi, i, value)
				elref, elparm := p.GenElemRef(i, s.genParamRef(p, pi))
				valstr, value = s.GenValue(f, elparm, value, false)
				if elref == "" || elref == "_" || cel == 0 {
					b.WriteString(fmt.Sprintf("  // skip: %s\n", valstr))
					continue
				} else {
					basep, _ := genDeref(elparm)
					// Handle *p where p is an empty struct.
					if basep.NumElements() == 0 {
						continue
					}
					cvar := fmt.Sprintf("p%df%dc", pi, i)
					b.WriteString(fmt.Sprintf("  %s := %s\n", cvar, valstr))
					s.emitParamElemCheck(f, b, elparm, elref, cvar, pi, i)
				}
			}
			if p.AddrTaken() != notAddrTaken {
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
	if f.method {
		numel := f.receiver.NumElements()
		for i := 0; i < numel; i++ {
			verb(4, "emitting check-code for rcvr el %d value=%d", i, value)
			elref, elparm := f.receiver.GenElemRef(i, "rcvr")
			valstr, value = s.GenValue(f, elparm, value, false)
			if elref == "" || strings.HasPrefix(elref, "_") || f.receiver.IsBlank() {
				verb(4, "empty skip rcvr el %d", i)
				continue
			} else {

				basep, _ := genDeref(elparm)
				// Handle *p where p is an empty struct.
				if basep.NumElements() == 0 {
					continue
				}
				cvar := fmt.Sprintf("rcvrf%dc", i)
				b.WriteString(fmt.Sprintf("  %s := %s\n", cvar, valstr))
				s.emitParamElemCheck(f, b, elparm, elref, cvar, -1, i)
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
// where we randomly choose to either pass a param through to the
// function literal, or have the param captured by the closure, then
// check its value in the defer.
func (s *genstate) emitDeferChecks(f *funcdef, b *bytes.Buffer, pidx int, value int) int {

	if len(f.params) == 0 {
		return value
	}

	// make a pass through the params and randomly decide which will be passed into the func.
	passed := []bool{}
	for i := range f.params {
		p := f.dodefp[i] < 50
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
			elref, elparm := p.GenElemRef(i, s.genParamRef(p, pi))
			if elref == "" || elref == "_" || cel == 0 {
				verb(4, "empty skip p%d el %d", pi, i)
				continue
			} else {
				basep, _ := genDeref(elparm)
				// Handle *p where p is an empty struct.
				if basep.NumElements() == 0 {
					continue
				}
				cvar := fmt.Sprintf("p%df%dc", pi, i)
				s.emitParamElemCheck(f, b, elparm, elref, cvar, pi, i)
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

func (s *genstate) emitVarAssign(f *funcdef, b *bytes.Buffer, r parm, rname string, value int, caller bool) int {
	var valstr string
	isassign := uint8(s.wr.Intn(100)) < 50
	if rmp, ismap := r.(*mapparm); ismap && isassign {
		// emit: var m ... ; m[k] = v
		r.Declare(b, "  "+rname+" := make(", ")\n", caller)
		valstr, value = s.GenValue(f, rmp.valtype, value, caller)
		b.WriteString(fmt.Sprintf("  %s[mkt.%s] = %s\n",
			rname, rmp.keytmp, valstr))
	} else {
		// emit r = c
		valstr, value = s.GenValue(f, r, value, caller)
		b.WriteString(fmt.Sprintf("  %s := %s\n", rname, valstr))
	}
	return value
}

func (s *genstate) emitChecker(f *funcdef, b *bytes.Buffer, pidx int, emit bool) {
	verb(4, "emitting struct and array defs")
	s.emitStructAndArrayDefs(f, b)
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
	b.WriteString(fmt.Sprintf("  var pad [%d]uint64\n", f.rstack))
	b.WriteString(fmt.Sprintf("  pad[%s.FailCount&0x1]++\n", s.utilsPkg()))

	value := 1

	// generate map key tmps
	s.wr.Checkpoint("before map key temps")
	value = s.emitMapKeyTmps(f, b, pidx, value, false)

	// generate return constants
	s.wr.Checkpoint("before return constants")
	for ri, r := range f.returns {
		rc := fmt.Sprintf("rc%d", ri)
		value = s.emitVarAssign(f, b, r, rc, value, false)
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
				gv := s.genGlobVar(p)
				b.WriteString(fmt.Sprintf("  %s = a%s%d\n", gv, n, pi))
			}
		}
	}

	// parameter checking code
	var haveControl bool
	s.wr.Checkpoint("before param checks")
	value, haveControl = s.emitParamChecks(f, b, pidx, value)

	// defer testing
	if s.tunables.doDefer && f.dodefc < s.tunables.deferFraction {
		s.wr.Checkpoint("before defer checks")
		_ = s.emitDeferChecks(f, b, pidx, value)
	}

	// returns
	s.emitReturn(f, b, haveControl)

	b.WriteString(fmt.Sprintf("  // %d addr-taken params, %d addr-taken returns\n",
		aCounts[0], aCounts[1]))

	b.WriteString("}\n\n")

	// emit any new helper funcs referenced by this test function
	s.emitAddrTakenHelpers(f, b, emit)
}

// complexityMeasure returns an integer that estimates how complex a given test function
// is relative to some other function. The more parameters + returns and the more complicated
// the types of the params/returns, the higher the number returned here.
func (f *funcdef) complexityMeasure() int {
	v := int(0)
	if f.method {
		v += f.receiver.NumElements()
	}
	for _, p := range f.params {
		v += p.NumElements()
	}
	for _, r := range f.returns {
		v += r.NumElements()
	}
	return v
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
		if s.sforce {
			b.WriteString("  hackStack() // force stack growth on next call\n")
		}
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
	s.wr = NewWrapRand(seed, s.randctl)
	s.wr.tag = "genfunc"
	fp := s.GenFunc(fidx, pidx)

	// Emit caller side
	wrcaller := NewWrapRand(seed, s.randctl)
	s.wr = wrcaller
	s.wr.tag = "caller"
	s.emitCaller(fp, b, pidx)
	if emit {
		b.WriteTo(calloutfile)
	}
	b.Reset()

	// Emit checker side
	wrchecker := NewWrapRand(seed, s.randctl)
	s.wr = wrchecker
	s.wr.tag = "checker"
	s.emitChecker(fp, b, pidx, emit)
	if emit {
		b.WriteTo(checkoutfile)
	}
	b.Reset()
	wrchecker.Check(wrcaller)

	return seed + 1
}

func (s *genstate) openOutputFile(filename string, pk string, imports []string, ipref string) *os.File {
	verb(1, "opening %s", filename)
	outf, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	haveunsafe := false
	outf.WriteString(fmt.Sprintf("package %s\n\n", pk))
	for _, imp := range imports {
		if imp == "reflect" {
			outf.WriteString("import \"reflect\"\n")
			continue
		}
		if imp == "unsafe" {
			outf.WriteString("import _ \"unsafe\"\n")
			haveunsafe = true
			continue
		}
		outf.WriteString(fmt.Sprintf("import \"%s%s\"\n", ipref, imp))
	}
	outf.WriteString("\n")
	if s.sforce && haveunsafe {
		outf.WriteString("// Hack: reach into runtime to grab this testing hook.\n")
		outf.WriteString("//go:linkname hackStack runtime.gcTestMoveStackOnNextCall\n")
		outf.WriteString("func hackStack()\n\n")
	}
	return outf
}

func emitUtils(outf *os.File, maxfail int) {
	countfail := `
  if isret {
    if ParamFailCount != 0 {
      return
    }
    ReturnFailCount++
  } else {
    ParamFailCount++
  }
`
	earlyexit := fmt.Sprintf(`
  if (ParamFailCount + FailCount + ReturnFailCount > %d) {
    os.Exit(1)
  }
`, maxfail)

	fmt.Fprintf(outf, "import \"fmt\"\n")
	fmt.Fprintf(outf, "import \"os\"\n\n")
	fmt.Fprintf(outf, "var ParamFailCount int\n\n")
	fmt.Fprintf(outf, "var ReturnFailCount int\n\n")
	fmt.Fprintf(outf, "var FailCount int\n\n")
	fmt.Fprintf(outf, "var Mode string\n\n")
	fmt.Fprintf(outf, "type UtilsType int\n\n")
	fmt.Fprintf(outf, "//go:noinline\n")
	fmt.Fprintf(outf, "func NoteFailure(cm int, pidx int, fidx int, pkg string, pref string, parmNo int, isret bool,_ uint64) {")
	outf.WriteString(countfail)
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail %%s |%%d|%%d|%%d| =%%s.Test%%d= %%s %%d\\n\", Mode, cm, pidx, fidx, pkg, fidx, pref, parmNo)\n")
	outf.WriteString(earlyexit)
	fmt.Fprintf(outf, "}\n\n")
	fmt.Fprintf(outf, "//go:noinline\n")
	fmt.Fprintf(outf, "func NoteFailureElem(cm int, pidx int, fidx int, pkg string, pref string, parmNo int, elem int, isret bool, _ uint64) {\n")
	outf.WriteString(countfail)
	fmt.Fprintf(outf, "  fmt.Fprintf(os.Stderr, ")
	fmt.Fprintf(outf, "\"Error: fail %%s |%%d|%%d|%%d| =%%s.Test%%d= %%s %%d elem %%d\\n\", Mode, cm, pidx, fidx, pkg, fidx, pref, parmNo, elem)\n")
	outf.WriteString(earlyexit)
	fmt.Fprintf(outf, "}\n\n")
	fmt.Fprintf(outf, "func BeginFcn() {\n")
	fmt.Fprintf(outf, "  ParamFailCount = 0\n")
	fmt.Fprintf(outf, "  ReturnFailCount = 0\n")
	fmt.Fprintf(outf, "}\n\n")
	fmt.Fprintf(outf, "func EndFcn() {\n")
	fmt.Fprintf(outf, "  FailCount += ParamFailCount\n")
	fmt.Fprintf(outf, "  FailCount += ReturnFailCount\n")
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
				fmt.Fprintf(outf, "  %s.Caller%d(\"normal\")\n", cp, i)
				if s.tunables.doReflectCall {
					fmt.Fprintf(outf, "  %s.Caller%d(\"reflect\")\n", cp, i)
				}
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

func Generate(tag string, outdir string, pkgpath string, numit int, numtpkgs int, seed int64, pragma string, fcnmask map[int]int, pkmask map[int]int, utilsinl bool, maxfail int, forcestackgrowth bool, randctl int) int {
	mainpkg := tag + "Main"

	var ipref string
	if len(pkgpath) > 0 {
		ipref = pkgpath + "/"
	}

	s := genstate{
		outdir:  outdir,
		ipref:   ipref,
		tag:     tag,
		numtpk:  numtpkgs,
		pragma:  pragma,
		sforce:  forcestackgrowth,
		randctl: randctl,
	}

	if outdir != "." {
		makeDir(outdir)
	}
	verb(1, "creating %s", outdir)

	mainimports := []string{}
	for i := 0; i < numtpkgs; i++ {
		if emitFP(-1, i, nil, pkmask) {
			makeDir(outdir + "/" + s.callerPkg(i))
			makeDir(outdir + "/" + s.checkerPkg(i))
			makeDir(outdir + "/" + s.utilsPkg())
			mainimports = append(mainimports, s.callerPkg(i))
		}
	}
	mainimports = append(mainimports, s.utilsPkg())

	utilsfile := outdir + "/" + s.utilsPkg() + "/" + s.utilsPkg() + ".go"
	utilsoutfile := s.openOutputFile(utilsfile, s.utilsPkg(), []string{}, "")
	verb(1, "emit utils")
	emitUtils(utilsoutfile, maxfail)
	utilsoutfile.Close()

	mainfile := outdir + "/" + mainpkg + ".go"
	mainoutfile := s.openOutputFile(mainfile, "main", mainimports, ipref)

	for k := 0; k < numtpkgs; k++ {
		callerImports := []string{s.checkerPkg(k), s.utilsPkg()}
		checkerImports := []string{s.utilsPkg()}
		if tunables.doReflectCall {
			callerImports = append(callerImports, "reflect")
		}
		if s.sforce {
			callerImports = append(callerImports, "unsafe")
			checkerImports = append(checkerImports, "unsafe")
		}
		var calleroutfile, checkeroutfile *os.File
		if emitFP(-1, k, nil, pkmask) {
			calleroutfile = s.openOutputFile(s.callerFile(k), s.callerPkg(k),
				callerImports, ipref)
			checkeroutfile = s.openOutputFile(s.checkerFile(k), s.checkerPkg(k),
				checkerImports, ipref)
		}

		s.pkidx = k
		s.newDerefFuncs = nil
		s.newAssignFuncs = nil
		s.newGlobVars = nil
		s.derefFuncs = make(map[string]string)
		s.assignFuncs = make(map[string]string)
		s.allocFuncs = make(map[string]string)
		s.globVars = make(map[string]string)
		s.genvalFuncs = make(map[string]string)

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
