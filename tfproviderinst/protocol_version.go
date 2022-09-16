package tfproviderinst

import (
	"fmt"
	"strconv"
	"strings"
)

type ProtocolVersion struct {
	Major, Minor int64
}

func ParseProtocolVersion(str string) (ProtocolVersion, error) {
	const errMsg = "protocol version must be two decimal integers separated by a dot"

	dot := strings.Index(str, ".")
	if dot < 0 {
		return ProtocolVersion{}, fmt.Errorf(errMsg)
	}
	majorStr := str[:dot]
	minorStr := str[dot+1:]

	major, err := strconv.ParseInt(majorStr, 10, 64)
	if err != nil {
		return ProtocolVersion{}, fmt.Errorf(errMsg)
	}
	minor, err := strconv.ParseInt(minorStr, 10, 64)
	if err != nil {
		return ProtocolVersion{}, fmt.Errorf(errMsg)
	}

	return ProtocolVersion{major, minor}, nil
}
