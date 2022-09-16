// Package common contains some types and functions that both the public
// tfprovider package and the internal protocol-version-specific implementations
// need to refer to.
//
// It exists only to avoid a dependency cycle between the version-agnostic
// public package and the version-specific packages. Most exported items in
// this package should be re-exported from the tfprovider package for use by
// external callers.
package common
