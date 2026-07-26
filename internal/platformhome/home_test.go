package platformhome

import "testing"

func TestResolveFor(t *testing.T) {
	tests := []struct {
		name, goos, override, base, want string
		wantErr                          bool
	}{
		{name: "linux default", goos: "linux", base: "/home/operator/.config", want: "/home/operator/.config/platformctl"},
		{name: "windows default", goos: "windows", base: `C:\Users\operator\AppData\Roaming`, want: `C:\Users\operator\AppData\Roaming\platformctl`},
		{name: "macos default", goos: "darwin", base: "/Users/operator/Library/Application Support", want: "/Users/operator/Library/Application Support/platformctl"},
		{name: "linux override", goos: "linux", override: "/srv/platform", base: "/ignored", want: "/srv/platform"},
		{name: "windows override", goos: "windows", override: `D:\platform`, base: `C:\ignored`, want: `D:\platform`},
		{name: "relative rejected", goos: "linux", override: "relative", base: "/ignored", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveFor(test.goos, test.override, test.base)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
