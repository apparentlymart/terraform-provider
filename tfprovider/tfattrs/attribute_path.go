package tfattrs

import (
	"fmt"
)

type Path []PathStep

type PathStep interface {
	pathStep()
	String() string
}

type Name string

func (n Name) pathStep() {}

func (n Name) String() string {
	return fmt.Sprintf(".%s", string(n))
}

type MapKey string

func (k MapKey) pathStep() {}

func (k MapKey) String() string {
	return fmt.Sprintf("[%q]", string(k))
}

type ListIndex int64

func (i ListIndex) pathStep() {}

func (i ListIndex) String() string {
	return fmt.Sprintf("[%d]", int64(i))
}
