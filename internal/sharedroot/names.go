package sharedroot

import "regexp"

// Name patterns of the standard ("Names"). None of them can hold a path
// separator or "..", so every path built from them stays under <root>/<env>/.
var (
	envPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	targetPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	consumerPattern = regexp.MustCompile(`^provision-[a-z0-9]+(-[a-z0-9]+)*$`)
	identityPattern = targetPattern
	keyPattern      = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

// identityStatus is the status of an identity in the standard's
// "Identities" table.
type identityStatus int

const (
	identityUnknown identityStatus = iota
	identityActive
	identityReserved
	identityDeprecated
)

// identities mirrors the "Identities" table of the platform-catalog
// provision-shared-root-standard.md (main c2696ab, ADR-0007). Only an active
// identity may be declared or have a directory; a reserved one is kept for a
// later consumer and a deprecated one is only used by the legacy layout of
// its consumers, so both are rejected like an unknown name.
var identities = map[string]identityStatus{
	"vcenter-access-admin":    identityActive,
	"provision-nexus3":        identityReserved,
	"provision-semaphoreui":   identityReserved,
	"platform-modules-live":   identityReserved,
	"platform-modules-packer": identityReserved,
	"terraform-provisioner":   identityDeprecated,
}

// identityRejection returns why an identity name that matches the pattern
// may not be declared or have a directory, or "" when it is active.
func identityRejection(name string) string {
	switch identities[name] {
	case identityActive:
		return ""
	case identityReserved:
		return "is reserved and cannot be used yet"
	case identityDeprecated:
		return "is deprecated: no new declaration and no directory (ADR-0007)"
	default:
		return "is not an active identity of the shared-root standard"
	}
}

// knownEnvironments mirrors the platform-catalog catalog/environments/
// entries (main 561f0ca). The catalog is not available at run time, so this
// list is kept here and changes in the same pull request that follows a new
// catalog environment.
var knownEnvironments = map[string]bool{"lab": true, "prod": true}

// ValidateEnv checks an environment name: its pattern, and that it is one of
// the catalog environments.
func ValidateEnv(name string) error {
	if err := match("environment", name, envPattern); err != nil {
		return err
	}
	if !knownEnvironments[name] {
		return Usagef("unknown environment %q: not an entry of platform-catalog catalog/environments/", name)
	}
	return nil
}

// ValidateTarget checks a target name.
func ValidateTarget(name string) error {
	return match("target", name, targetPattern)
}

// ValidateConsumer checks a consumer (provision-* repository) name.
func ValidateConsumer(name string) error {
	return match("consumer", name, consumerPattern)
}

// ValidateIdentity checks an identity name and that it is active.
func ValidateIdentity(name string) error {
	if err := match("identity", name, identityPattern); err != nil {
		return err
	}
	if why := identityRejection(name); why != "" {
		return Usagef("identity %q %s", name, why)
	}
	return nil
}

// ValidateKey checks a key or credential file name.
func ValidateKey(name string) error {
	return match("key name", name, keyPattern)
}

func match(kind, name string, pattern *regexp.Regexp) error {
	if !pattern.MatchString(name) {
		return Usagef("invalid %s name %q: must match %s", kind, name, pattern.String())
	}
	return nil
}
