package common

// Sealed is a common type intentionally _not_ exported in the public
// API, because we rely on the fact that only our own packages can reach it
// in order to make tfprovider.Provider be a sealed interface that no other
// module can implement.
//
// (We need to be able to expand it in future when there are new Terraform
// plugin protocol features.)
type Sealed struct{}
