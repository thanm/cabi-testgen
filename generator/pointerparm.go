package generator

import (
	"bytes"
	"fmt"
)

// pointerparm describes a parameter of pointer type; it implements the
// "parm" interface.
type pointerparm struct {
	tag    string
	totype parm
	isBlank
	addrTakenHow
}

func (p pointerparm) Declare(b *bytes.Buffer, prefix string, suffix string, caller bool) {
	n := p.totype.TypeName()
	if caller {
		n = p.totype.QualName()
	}
	b.WriteString(fmt.Sprintf("%s *%s%s", prefix, n, suffix))
}

func (p pointerparm) GenElemRef(elidx int, path string) (string, parm) {
	return path, &p
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

func mkPointerParm(to parm) pointerparm {
	var pp pointerparm
	pp.tag = "pointer"
	pp.totype = to
	return pp
}
