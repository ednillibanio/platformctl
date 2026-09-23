package render

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/ednillibanio/platformctl/internal/config"
)

// platformModulesVariables lists the PM_* variables that platform-modules
// reads, taken from packer/lib/pkr-vars.sh and scripts/gitops-config/lib.sh
// in al-rr/platform-modules (lab, eb8bc97). PM_DEFAULT_TARGET is deliberately
// absent: it makes packerctl source a targets/<name>.sh file, which does not
// exist when the configuration comes from platformctl export.
var platformModulesVariables = map[string]bool{
	"PM_ANSIBLE_KEY": true, "PM_ANSIBLE_PRIVATE_KEY_FILE": true, "PM_ANSIBLE_USERNAME": true,
	"PM_BUILD_PASSWORD": true, "PM_BUILD_PASSWORD_ENCRYPTED": true, "PM_BUILD_USERNAME": true,
	"PM_COMMON_CONTENT_LIBRARY": true, "PM_COMMON_CONTENT_LIBRARY_DESTROY": true,
	"PM_COMMON_CONTENT_LIBRARY_ENABLED": true, "PM_COMMON_CONTENT_LIBRARY_OVF": true,
	"PM_COMMON_CONTENT_LIBRARY_SKIP_EXPORT": true, "PM_COMMON_DATA_SOURCE": true,
	"PM_COMMON_HCP_PACKER_REGISTRY_ENABLED": true, "PM_COMMON_HTTP_IP": true,
	"PM_COMMON_HTTP_PORT_MAX": true, "PM_COMMON_HTTP_PORT_MIN": true,
	"PM_COMMON_IP_SETTLE_TIMEOUT": true, "PM_COMMON_IP_WAIT_TIMEOUT": true,
	"PM_COMMON_ISO_AUTO_DOWNLOAD": true, "PM_COMMON_ISO_CONTENT_LIBRARY": true,
	"PM_COMMON_ISO_CONTENT_LIBRARY_ENABLED": true, "PM_COMMON_ISO_DATASTORE": true,
	"PM_COMMON_ISO_SOURCE_CHECKSUM": true, "PM_COMMON_ISO_SOURCE_URL": true,
	"PM_COMMON_OVF_EXPORT_ENABLED": true, "PM_COMMON_OVF_EXPORT_IMAGE_FILES": true,
	"PM_COMMON_OVF_EXPORT_OVERWRITE": true, "PM_COMMON_REMOVE_CDROM": true,
	"PM_COMMON_SHUTDOWN_TIMEOUT": true, "PM_COMMON_TEMPLATE_CONVERSION": true,
	"PM_COMMON_TOOLS_UPGRADE_POLICY": true, "PM_COMMON_VM_VERSION": true,
	"PM_COMMUNICATOR_PROXY_HOST": true, "PM_COMMUNICATOR_PROXY_PASSWORD": true,
	"PM_COMMUNICATOR_PROXY_PORT": true, "PM_COMMUNICATOR_PROXY_USERNAME": true,
	"PM_GIT_USER_EMAIL": true, "PM_GIT_USER_NAME": true, "PM_GITHUB_TOKEN": true,
	"PM_PACKER_ENABLE_LOG": true, "PM_PACKER_EXECUTION_MODE": true, "PM_PACKER_LOG_PATH": true,
	"PM_VSPHERE_CLUSTER": true, "PM_VSPHERE_DATACENTER": true, "PM_VSPHERE_DATASTORE": true,
	"PM_VSPHERE_ENDPOINT": true, "PM_VSPHERE_FOLDER": true, "PM_VSPHERE_HOST": true,
	"PM_VSPHERE_HOST_PASSWORD": true, "PM_VSPHERE_HOST_USERNAME": true,
	"PM_VSPHERE_INSECURE_CONNECTION": true, "PM_VSPHERE_NETWORK": true,
	"PM_VSPHERE_PASSWORD": true, "PM_VSPHERE_RESOURCE_POOL": true,
	"PM_VSPHERE_SET_HOST_FOR_DATASTORE_UPLOADS": true, "PM_VSPHERE_USERNAME": true,
}

var exportedName = regexp.MustCompile(`(?m)^  export ([A-Z0-9_]+)=`)

func TestPlatformModulesExampleExport(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "examples", "platform-modules-home"))
	bundle, err := config.Load(root, "lab")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveServices(bundle, []string{"vsphere-esxi", "packer"})
	if err != nil {
		t.Fatal(err)
	}
	output, err := EncodeEnv(resolved, EnvOptions{Prefix: "PM_"})
	if err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", "platform-modules-export.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, output, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != string(want) {
		t.Fatalf("export differs from %s (set UPDATE_GOLDEN=1 to refresh):\n%s", golden, output)
	}

	names := exportedName.FindAllStringSubmatch(string(output), -1)
	if len(names) == 0 {
		t.Fatal("example exported no variables")
	}
	for _, match := range names {
		if !platformModulesVariables[match[1]] {
			t.Errorf("%s is not read by platform-modules; fix the example key or the variable list", match[1])
		}
	}
}
