package sharedroot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclarationStandardExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "declaration", "standard-example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	decl, err := ParseDeclaration("shared-root.yaml", data)
	if err != nil {
		t.Fatalf("ParseDeclaration: %v", err)
	}
	if decl.Version != 1 || decl.Consumer != "provision-example" || !decl.AutomationKey || len(decl.Warnings) != 0 {
		t.Fatalf("header: %+v", decl)
	}
	if len(decl.Target) != 1 || decl.Target[0] != (KeyEntry{Name: "GOVC_URL", Required: true,
		Description: "vCenter SDK URL of the target", Example: "https://vcenter.example.test/sdk", Line: 9}) {
		t.Fatalf("target: %+v", decl.Target)
	}
	if len(decl.Host) != 1 || decl.Host[0].Name != "HOST_IP" || !decl.Host[0].Required {
		t.Fatalf("host: %+v", decl.Host)
	}
	if len(decl.Credentials) != 1 || decl.Credentials[0].Name != "EXAMPLE_DB_PASS" {
		t.Fatalf("credentials: %+v", decl.Credentials)
	}
	if len(decl.TargetCredentials) != 1 || decl.TargetCredentials[0].Identity != "vcenter-access-admin" {
		t.Fatalf("target_credentials: %+v", decl.TargetCredentials)
	}
}

func TestDeclarationMinimal(t *testing.T) {
	decl, err := ParseDeclaration("d.yaml", []byte("version: 1\nconsumer: provision-x\n"))
	if err != nil || decl.AutomationKey || decl.Target != nil {
		t.Fatalf("decl %+v, err %v", decl, err)
	}
	// Same name for two identities is allowed. The standard has one active
	// identity today, so the test activates a second one.
	identities["provision-x"] = identityActive
	t.Cleanup(func() { delete(identities, "provision-x") })
	two := "version: 1\nconsumer: provision-x\ntarget_credentials:\n" +
		"  - {name: GOVC_PASSWORD, identity: vcenter-access-admin, required: true, description: a}\n" +
		"  - {name: GOVC_PASSWORD, identity: provision-x, required: true, description: b}\n"
	if _, err := ParseDeclaration("d.yaml", []byte(two)); err != nil {
		t.Fatalf("two identities: %v", err)
	}
}

const head = "version: 1\nconsumer: provision-x\n"

func TestDeclarationRefusals(t *testing.T) {
	cases := []struct {
		name, yaml, want string
	}{
		{"empty", "", "empty declaration"},
		{"syntax", "version: [1\n", "invalid YAML"},
		{"two documents", head + "---\n" + head, "only one YAML document"},
		{"not a mapping", "- a\n", "must be a mapping"},
		{"missing version", "consumer: provision-x\n", "version: required field is missing"},
		{"missing consumer", "version: 1\n", "consumer: required field is missing"},
		{"string version", "version: \"1\"\nconsumer: provision-x\n", "version: must be an integer"},
		{"unsupported version", "version: 2\nconsumer: provision-x\n", "unsupported version 2"},
		{"old version", "version: 0\nconsumer: provision-x\n", "unsupported version 0"},
		{"bad consumer", "version: 1\nconsumer: semaphoreui\n", "invalid consumer name"},
		{"unknown top field", head + "extra: 1\n", "extra: unknown field"},
		{"duplicate top field", head + "consumer: provision-y\n", "consumer: field set twice"},
		{"automation_key string", head + "automation_key: yes-please\n", "automation_key: must be true or false"},
		{"target not a list", head + "target: GOVC_URL\n", "target: must be a list"},
		{"entry not a mapping", head + "target:\n  - GOVC_URL\n", "each entry must be a mapping"},
		{"unknown entry field", head + "target:\n  - {name: GOVC_URL, required: true, description: d, secret: x}\n", "target.secret: unknown field"},
		{"example in credentials", head + "credentials:\n  - {name: A_PASS, required: true, description: d, example: x}\n", "credentials.example: unknown field"},
		{"identity in target", head + "target:\n  - {name: GOVC_URL, required: true, description: d, identity: vcenter-access-admin}\n", "target.identity: unknown field"},
		{"missing name", head + "target:\n  - {required: true, description: d}\n", "target.name: required field is missing"},
		{"bad name", head + "target:\n  - {name: govc_url, required: true, description: d}\n", "invalid name"},
		{"missing required", head + "target:\n  - {name: GOVC_URL, description: d}\n", "GOVC_URL.required: required field is missing"},
		{"required string", head + "target:\n  - {name: GOVC_URL, required: \"true\", description: d}\n", "GOVC_URL.required: must be true or false"},
		{"missing description", head + "target:\n  - {name: GOVC_URL, required: true}\n", "GOVC_URL.description: required field is missing"},
		{"empty description", head + "target:\n  - {name: GOVC_URL, required: true, description: \" \"}\n", "must not be empty"},
		{"multiline description", head + "target:\n  - name: GOVC_URL\n    required: true\n    description: |\n      a\n      b\n", "must be one line"},
		{"target prefix", head + "target:\n  - {name: HOST_IP, required: true, description: d}\n", "target keys start with"},
		{"host prefix", head + "host:\n  - {name: GOVC_URL, required: true, description: d}\n", "host keys start with HOST_"},
		{"duplicate target", head + "target:\n  - {name: GOVC_URL, required: true, description: d}\n  - {name: GOVC_URL, required: false, description: e}\n", "declared twice in target (first on line 4)"},
		{"duplicate credential", head + "credentials:\n  - {name: A, required: true, description: d}\n  - {name: A, required: true, description: d}\n", "declared twice in credentials"},
		{"duplicate target credential", head + "target_credentials:\n  - {name: P, identity: vcenter-access-admin, required: true, description: d}\n  - {name: P, identity: vcenter-access-admin, required: true, description: d}\n", "declared twice for the same identity"},
		{"missing identity", head + "target_credentials:\n  - {name: P, required: true, description: d}\n", "P.identity: required field is missing"},
		{"reserved identity", head + "target_credentials:\n  - {name: P, identity: provision-semaphoreui, required: true, description: d}\n", "is reserved"},
		{"deprecated identity", head + "target_credentials:\n  - {name: P, identity: terraform-provisioner, required: true, description: d}\n", "P.identity: identity is deprecated"},
		{"invalid identity name", head + "target_credentials:\n  - {name: P, identity: Admin, required: true, description: d}\n", "invalid identity name"},
		{"alias as a section", head + "x: &items\n  - {name: GOVC_URL, required: true, description: d}\ntarget: *items\n", "x: unknown field"},
		{"alias value", head + "target: *missing\n", "invalid YAML"},
		{"merge key", head + "target:\n  - <<: {name: GOVC_URL, required: true}\n    description: d\n", "target.<<: unknown field"},
		{"null required", head + "target:\n  - {name: GOVC_URL, required: null, description: d}\n", "GOVC_URL.required: must be true or false"},
		{"tagged string", head + "target:\n  - {name: !!binary R09WQw==, required: true, description: d}\n", "target.name: must be a string"},
		{"unknown identity", head + "target_credentials:\n  - {name: P, identity: someone, required: true, description: d}\n", "not an active identity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDeclaration("shared-root.yaml", []byte(tc.yaml))
			if err == nil {
				t.Fatalf("accepted:\n%s", tc.yaml)
			}
			if ExitCode(err) != ExitProblem {
				t.Fatalf("want a problem, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
			if !strings.HasPrefix(err.Error(), "shared-root.yaml") {
				t.Fatalf("error does not name the file: %q", err)
			}
		})
	}
}

func TestDeclarationReportsEveryProblem(t *testing.T) {
	yaml := "version: 3\nconsumer: bad\nextra: 1\ntarget:\n  - {name: HOST_IP, required: true, description: d}\n"
	_, err := ParseDeclaration("d.yaml", []byte(yaml))
	if err == nil {
		t.Fatal("want problems")
	}
	problems := err.(Problems)
	if len(problems) != 4 {
		t.Fatalf("want 4 problems, got:\n%v", err)
	}
	for i, line := range []int{1, 2, 3, 5} {
		if problems[i].Line != line {
			t.Errorf("problem %d on line %d, want %d: %v", i, problems[i].Line, line, problems[i])
		}
	}
}

func TestDeclarationErrorsHoldNoValue(t *testing.T) {
	cases := []string{
		"version: s3cr3t-value\nconsumer: provision-x\n",
		head + "target:\n  - {name: GOVC_URL, required: s3cr3t-value, description: d}\n",
		head + "target: s3cr3t-value\n",
		"version: 1\nconsumer: provision-x\nkey: \"s3cr3t-value\n",
		head + "target_credentials:\n  - {name: P, identity: s3cr3t-value, required: true, description: d}\n",
		head + "target_credentials:\n  - {name: P, identity: S3cr3t/value, required: true, description: d}\n",
	}
	for _, yaml := range cases {
		_, err := ParseDeclaration("d.yaml", []byte(yaml))
		if err == nil || strings.Contains(strings.ToLower(err.Error()), "s3cr3t") {
			t.Errorf("error holds the value or is missing: %v", err)
		}
	}
}
