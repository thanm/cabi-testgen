package generator

import (
	"bytes"
	"fmt"
	"math/rand"
)

// typedefparm describes a parameter that is a typedef of some other
// type; it implements the "parm" interface
type typedefparm struct {
	aname  string
	qname  string
	target parm
	isBlank
	addrTakenHow
}

func (p typedefparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	n := p.aname
	if caller {
		n = p.qname
	}
	b.WriteString(fmt.Sprintf("%s %s%s", prefix, n, suffix))
}

func (p typedefparm) GenElemRef(elidx int, path string) (string, parm) {
	_, isarr := p.target.(*arrayparm)
	_, isstruct := p.target.(*structparm)
	rv, rp := p.target.GenElemRef(elidx, path)
	// this is hacky, but I don't see a nicer way to do this
	if isarr || isstruct {
		return rv, rp
	}
	rp = &p
	return rv, rp
}

func (p typedefparm) GenValue(s *genstate, value int, caller bool) (string, int) {
	n := p.aname
	if caller {
		n = p.qname
	}
	rv, v := p.target.GenValue(s, value, caller)
	rv = n + "(" + rv + ")"
	return rv, v
}

func (p typedefparm) IsControl() bool {
	return false
}

func (p typedefparm) NumElements() int {
	return p.target.NumElements()
}

func (p typedefparm) String() string {
	return fmt.Sprintf("%s typedef of %s", p.aname, p.target.String())

}

func (p typedefparm) TypeName() string {
	return p.aname

}

func (p typedefparm) QualName() string {
	return p.qname
}

func (s *genstate) makeTypedefParm(f *funcdef, target parm, pidx int) parm {
	var tdp typedefparm
	ns := len(f.typedefs)
	tdp.aname = fmt.Sprintf("MyTypeF%dS%d", f.idx, ns)
	tdp.qname = fmt.Sprintf("%s.MyTypeF%dS%d", s.checkerPkg(pidx), f.idx, ns)
	tdp.target = target
	tdp.SetBlank(uint8(rand.Intn(100)) < tunables.blankPerc)
	f.typedefs = append(f.typedefs, tdp)
	return &tdp
}
