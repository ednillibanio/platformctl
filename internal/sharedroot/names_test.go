package sharedroot

import "testing"

func TestNameValidation(t *testing.T) {
	cases := []struct {
		kind  string
		check func(string) error
		valid []string
		bad   []string
	}{
		{"env", ValidateEnv, []string{"lab", "prod"},
			[]string{"", "homol", "dev", "Lab", "lab-1", "1lab", "../lab", "lab/x", "..", "la b"}},
		{"target", ValidateTarget, []string{"vsphere-lab", "vsphere", "a1-b2"},
			[]string{"", "Vsphere", "vsphere_lab", "-x", "x-", "x--y", "a/b", "..", "../x"}},
		{"consumer", ValidateConsumer, []string{"provision-example", "provision-vsphere-platform"},
			[]string{"", "example", "provision-", "Provision-x", "provision-x/y", "provision-..", "automation", "target-vsphere-lab"}},
		{"identity", ValidateIdentity, []string{"vcenter-access-admin"},
			[]string{"", "provision-semaphoreui", "provision-nexus3", "platform-modules-live", "platform-modules-packer",
				"terraform-provisioner", "packer-builder", "semaphoreui", "unknown-identity", "Vcenter-access-admin", "vcenter/access", ".."}},
		{"key", ValidateKey, []string{"GOVC_URL", "HOST_IP", "A", "X1_2"},
			[]string{"", "govc_url", "1KEY", "_KEY", "KEY-NAME", "KEY/NAME", "..", "KEY NAME"}},
	}
	for _, tc := range cases {
		for _, name := range tc.valid {
			if err := tc.check(name); err != nil {
				t.Errorf("%s %q: unexpected error %v", tc.kind, name, err)
			}
		}
		for _, name := range tc.bad {
			if err := tc.check(name); ExitCode(err) != ExitUsage {
				t.Errorf("%s %q: want usage error, got %v", tc.kind, name, err)
			}
		}
	}
}
