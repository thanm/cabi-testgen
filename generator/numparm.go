package generator

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
)

// numparm describes a numeric parameter type; it implements the
// "parm" interface.
type numparm struct {
	tag         string
	widthInBits uint32
	ctl         bool
	isBlank
	addrTakenHow
}

var f32parm *numparm = &numparm{
	tag:         "float",
	widthInBits: uint32(32),
	ctl:         false,
}
var f64parm *numparm = &numparm{
	tag:         "float",
	widthInBits: uint32(64),
	ctl:         false,
}

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
	var rp parm
	rp = &p
	return path, rp
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
		nrange := 1 << (p.widthInBits - 2)
		v := rand.Intn(nrange)
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
	r, nv := p.genRandNum(value)
	verb(5, "numparm.GenValue(%d) = %s", value, r)
	return r, nv
}
