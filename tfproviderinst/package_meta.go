package tfproviderinst

import (
	"crypto/sha256"
	"net/url"
)

type RemotePackageMeta struct {
	PackageURL    *url.URL
	ArchiveSHA256 [sha256.Size]byte

	// Authority is set only when the metadata originates from the provider's
	// origin registry, and contains the information required to obtain and
	// verify the provider developer's own signed checksums.
	//
	// Packages from network mirrors are non-authoritative and so Authority
	// will always be nil in that case.
	Authority *PackageAuthority
}

type PackageAuthority struct {
	SHA256SumsURL          *url.URL
	SHA256SumsSignatureURL *url.URL
	Filename               string
	GPGKeys                []PackageAuthorityKey
}

type PackageAuthorityKey struct {
	KeyID          string
	ASCIIArmor     string
	TrustSignature string
	Source         string
	SourceURL      string
}
