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

// Identities of the standard ("Identities"). Only an active identity may be
// declared or have a directory; a reserved one is rejected.
var (
	activeIdentities   = map[string]bool{"vcenter-access-admin": true, "terraform-provisioner": true}
	reservedIdentities = map[string]bool{"packer-builder": true, "semaphoreui": true}
)

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
	if reservedIdentities[name] {
		return Usagef("identity %q is reserved and cannot be used yet", name)
	}
	if !activeIdentities[name] {
		return Usagef("identity %q is not an active identity of the shared-root standard", name)
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
