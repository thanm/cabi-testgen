package generator

import "bytes"

// parm is an interface describing an abstract parameter or return type.
type parm interface {
	Declare(b *bytes.Buffer, prefix string, suffix string, caller bool)
	GenElemRef(elidx int, path string) (string, parm)
	GenValue(s *genstate, value int, caller bool) (string, int)
	IsControl() bool
	NumElements() int
	String() string
	TypeName() string
	QualName() string
	IsBlank() bool
	HasPointer() bool
	SetBlank(v bool)
	AddrTaken() addrTakenHow
	SetAddrTaken(val addrTakenHow)
}

type addrTakenHow uint8

const (
	// Param not address taken.
	notAddrTaken addrTakenHow = 0

	// Param address is taken and used for simple reads/writes.
	addrTakenSimple addrTakenHow = 1

	// Param address is taken and passed to a well-behaved function.
	addrTakenPassed addrTakenHow = 2

	// Param address is taken and stored to a global var.
	addrTakenHeap addrTakenHow = 3
)

func (a *addrTakenHow) AddrTaken() addrTakenHow {
	return *a
}

func (a *addrTakenHow) SetAddrTaken(val addrTakenHow) {
	*a = val
}

type isBlank bool

func (b *isBlank) IsBlank() bool {
	return bool(*b)
}

func (b *isBlank) SetBlank(val bool) {
	*b = isBlank(val)
}
