package tfproviderinst

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform represents a target platform that a provider is or might be
// available for.
type Platform struct {
	OS, Arch string
}

func (p Platform) String() string {
	return p.OS + "_" + p.Arch
}

// LessThan returns true if the receiver should sort before the other given
// Platform in an ordered list of platforms.
//
// The ordering is lexical first by OS and then by Architecture.
// This ordering is primarily just to ensure that results of
// functions in this package will be deterministic. The ordering is not
// intended to have any semantic meaning and is subject to change in future.
func (p Platform) LessThan(other Platform) bool {
	switch {
	case p.OS != other.OS:
		return p.OS < other.OS
	default:
		return p.Arch < other.Arch
	}
}

// ParsePlatform parses a string representation of a platform, like
// "linux_amd64", or returns an error if the string is not valid.
func ParsePlatform(str string) (Platform, error) {
	parts := strings.Split(str, "_")
	if len(parts) != 2 {
		return Platform{}, fmt.Errorf("must be two words separated by an underscore")
	}

	os, arch := parts[0], parts[1]
	if !validPlatformPart(os) {
		return Platform{}, fmt.Errorf("OS portion must not contain whitespace")
	}
	if !validPlatformPart(arch) {
		return Platform{}, fmt.Errorf("architecture portion must not contain whitespace")
	}

	return Platform{
		OS:   os,
		Arch: arch,
	}, nil
}

func validPlatformPart(str string) bool {
	return !strings.ContainsAny(str, " \t\n\r")
}

// CurrentPlatform is the platform where the current program is running.
//
// If attempting to install providers for use on the same system where the
// installation process is running, this is the right platform to use.
var CurrentPlatform = Platform{
	OS:   runtime.GOOS,
	Arch: runtime.GOARCH,
}
