package sharedroot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureLines(t *testing.T, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "shellfile", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestShellFileFixtures(t *testing.T) {
	for _, line := range fixtureLines(t, "accepted.txt") {
		if _, err := ParseShellFile("vars.sh", VarsFile, []byte(line+"\n")); err != nil {
			t.Errorf("accepted line refused: %q: %v", line, err)
		}
	}
	for _, name := range []string{"refused.txt", "refused-tilde.txt", "refused-comment.txt"} {
		for _, line := range fixtureLines(t, name) {
			_, err := ParseShellFile("vars.sh", VarsFile, []byte(line+"\n"))
			if err == nil {
				t.Errorf("%s: line accepted: %q", name, line)
				continue
			}
			if ExitCode(err) != ExitProblem {
				t.Errorf("%s: %q: want a problem, got %v", name, line, err)
			}
			for _, secret := range []string{"hidden", "touch", "HOME", "unterminated", "operator", "path"} {
				if strings.Contains(line, secret) && strings.Contains(err.Error(), secret) {
					t.Errorf("%s: error prints part of the line (%q): %v", name, secret, err)
				}
			}
		}
	}
}

func TestShellFileValuesAndPlaceholders(t *testing.T) {
	content := strings.Join([]string{
		"# vCenter SDK URL of the target",
		"# GOVC_URL=",
		"GOVC_USERNAME='terraform provisioner'",
		`export TARGET_FOLDER="/dc/vm/services"`,
		"DISPOSABLE_POOL=pool-1",
		"",
		"# TARGET_DATASTORE=example",
	}, "\n")
	got, err := ParseShellFile("vars.sh", VarsFile, []byte(content))
	if err != nil {
		t.Fatalf("ParseShellFile: %v", err)
	}
	want := []Assignment{
		{Key: "GOVC_USERNAME", Value: "terraform provisioner", Line: 3},
		{Key: "TARGET_FOLDER", Value: "/dc/vm/services", Line: 4},
		{Key: "DISPOSABLE_POOL", Value: "pool-1", Line: 5},
	}
	if len(got.Assignments) != len(want) {
		t.Fatalf("assignments %+v, want %+v", got.Assignments, want)
	}
	for i := range want {
		if got.Assignments[i] != want[i] {
			t.Errorf("assignment %d: %+v, want %+v", i, got.Assignments[i], want[i])
		}
	}
	if len(got.Placeholders) != 2 || got.Placeholders[0] != (Placeholder{Key: "GOVC_URL", Line: 2}) ||
		got.Placeholders[1] != (Placeholder{Key: "TARGET_DATASTORE", Line: 7}) {
		t.Fatalf("placeholders %+v", got.Placeholders)
	}
}

func TestShellFileKeyRules(t *testing.T) {
	cases := []struct {
		kind FileKind
		line string
		ok   bool
	}{
		{VarsFile, "GOVC_URL=x", true},
		{VarsFile, "TARGET_FOLDER=x", true},
		{VarsFile, "DISPOSABLE_POOL=x", true},
		{VarsFile, "HOST_IP=192.0.2.1", false},
		{VarsFile, "OTHER_KEY=x", false},
		{VarsFile, "govc_url=x", false},
		{HostsFile, "HOST_IP=192.0.2.1", true},
		{HostsFile, "GOVC_URL=x", false},
		{HostsFile, "DISPOSABLE_POOL=x", false},
		{HostsFile, "host_ip=x", false},
		{FileKind(2), "HOST_IP=x", false},
		{FileKind(-1), "GOVC_URL=x", false},
	}
	for _, tc := range cases {
		_, err := ParseShellFile("f.sh", tc.kind, []byte(tc.line))
		if (err == nil) != tc.ok {
			t.Errorf("kind %d %q: ok=%v, err %v", tc.kind, tc.line, tc.ok, err)
		}
	}
}

func TestShellFileReportsEveryProblemWithLocation(t *testing.T) {
	content := "GOVC_URL=a\nHOST_IP=x\nGOVC_URL=b\nGOVC_PASSWORD=$(cat secret-value)\nTARGET_X=y\r\n"
	_, err := ParseShellFile("/root/lab/targets/t/vars.sh", VarsFile, []byte(content))
	if err == nil {
		t.Fatal("want problems")
	}
	problems := err.(Problems)
	want := []string{
		"/root/lab/targets/t/vars.sh:2: HOST_IP: vars.sh holds only",
		"/root/lab/targets/t/vars.sh:3: GOVC_URL: key set twice (first on line 1)",
		"/root/lab/targets/t/vars.sh:4: GOVC_PASSWORD: only KEY=value lines",
		"/root/lab/targets/t/vars.sh:5: TARGET_X: carriage return",
	}
	if len(problems) != len(want) {
		t.Fatalf("problems:\n%v", err)
	}
	for i, w := range want {
		if !strings.HasPrefix(problems[i].String(), w) {
			t.Errorf("problem %d: %q, want prefix %q", i, problems[i].String(), w)
		}
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("error prints a value: %v", err)
	}
}

func TestShellFileTrailingCommentMessage(t *testing.T) {
	_, err := ParseShellFile("vars.sh", VarsFile, []byte("GOVC_URL=https://vc.example.invalid/sdk # lab vCenter\n"))
	if err == nil {
		t.Fatal("trailing comment accepted")
	}
	if !strings.Contains(err.Error(), "a comment goes on its own line") {
		t.Errorf("want the trailing-comment message, got %v", err)
	}
	if strings.Contains(err.Error(), "vc.example.invalid") {
		t.Errorf("error prints the value: %v", err)
	}
}
