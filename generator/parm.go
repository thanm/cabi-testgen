package generator

import "bytes"

// parm is an interface describing an abstract parameter or return type.
type parm interface {
	Declare(b *bytes.Buffer, prefix string, suffix string, caller bool)
	GenElemRef(elidx int, path string) (string, parm)
	GenValue(value int, caller bool) (string, int)
	IsControl() bool
	NumElements() int
	String() string
	TypeName() string
	QualName() string
	IsBlank() bool
	SetBlank(v bool)
}
