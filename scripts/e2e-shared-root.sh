#!/usr/bin/env bash
# End-to-end check of the shared-root commands (bootstrap-config, credential,
# doctor) against a scratch root, never the operator's tree. It follows the
# "Validação Esperada" of iterations/004-shared-root/README.md.
#
# Usage: scripts/e2e-shared-root.sh [--bin <platformctl>]
#
#   --bin  binary to test (default: build ./cmd/platformctl with
#          scripts/go.sh into the scratch directory)
#
# It uses only the neutral declaration scripts/testdata/e2e/shared-root.yaml;
# whether a consumer's own loader accepts the tree is that consumer's test.
# Prints one line per check and exits 1 when any check fails. Linux and macOS;
# needs python3 or jq to parse the doctor JSON.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
decl="${repo}/scripts/testdata/e2e/shared-root.yaml"
bin=""
while (($# > 0)); do
	case "$1" in
	--bin)
		bin="${2:?--bin needs a path}"
		shift
		;;
	-h | --help)
		sed -n '2,15p' "$0"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		exit 2
		;;
	esac
	shift
done

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
if [[ -z "${bin}" ]]; then
	(cd "${repo}" && ./scripts/go.sh build -o "${work}/platformctl" ./cmd/platformctl)
	bin="${work}/platformctl"
fi
unset ASSEMBLEIARR_CONFIG_DIR

failures=0
# Secret values of the scratch tree. Captured output is printed on a failure
# only through show, which masks them, so a regression that leaks a secret
# fails the check without printing it.
secrets=()
add_secret_file() {
	local line
	[[ -f "$1" ]] || return 0
	while IFS= read -r line || [[ -n "${line}" ]]; do
		if [[ -n "${line}" && "${line}" != -----* ]]; then
			secrets+=("${line}")
		fi
	done <"$1"
	return 0
}
show() {
	local text s
	text="$(cat "${work}/$1.out")"
	for s in ${secrets[@]+"${secrets[@]}"}; do
		text="${text//"${s}"/<redacted>}"
	done
	printf '%s\n' "${text}" | sed 's/^/      /'
}
pass() { printf 'PASS  %s\n' "$1"; }
fail() {
	printf 'FAIL  %s\n' "$1"
	failures=$((failures + 1))
}

# run <name> <command>...: runs the command with output in ${work}/<name>.out
# and sets rc to its exit status.
run() {
	local name="$1"
	shift
	rc=0
	"$@" >"${work}/${name}.out" 2>&1 || rc=$?
}
out() { cat "${work}/$1.out"; }

# expect <description> <want-rc> <name>: checks the status of the last run.
expect() {
	if [[ "${rc}" == "$2" ]]; then pass "$1"; else
		fail "$1 (exit ${rc}, want $2)"
		show "$3"
	fi
}

# snapshot <root>: paths, types, modes, sizes, mtimes and content hashes.
# The tree holds only names this script and platformctl created (no newline
# or space), so a newline-separated listing and ls are safe; plain sort keeps
# it portable to the BSD sort of macOS.
# shellcheck disable=SC2012
snapshot() {
	(
		cd "$1" && find . | LC_ALL=C sort | while IFS= read -r p; do
			if [[ -f "${p}" ]]; then
				printf '%s %s %s\n' "${p}" "$(ls -ld "${p}" | awk '{print $1, $5}')" "$(cksum <"${p}")"
			else
				printf '%s %s\n' "${p}" "$(ls -ld "${p}" | awk '{print $1}')"
			fi
		done
		find . -newer "${work}/stamp" -print | LC_ALL=C sort
	)
}

# set_key <file> <key> <value>: turns the placeholder "# KEY=" into KEY=value.
set_key() {
	local tmp="$1.e2e"
	awk -v k="$2" -v v="$3" '$0 == "# " k "=" { print k "=" v; next } { print }' "$1" >"${tmp}"
	cat "${tmp}" >"$1"
	rm -f "${tmp}"
}

pc() { "${bin}" "$@" --config-dir "${root}"; }

# --- bootstrap-config ------------------------------------------------------
root="${work}/root"
run boot1 pc bootstrap-config --env lab --target vsphere-lab --declaration "${decl}" --automation-key
expect "bootstrap-config creates the skeleton" 0 boot1
add_secret_file "${root}/lab/credentials/automation/ssh_ed25519"
touch "${work}/stamp"
sleep 1
before="$(snapshot "${root}")"
run boot2 pc bootstrap-config --env lab --target vsphere-lab --declaration "${decl}" --automation-key
expect "bootstrap-config runs again" 0 boot2
if [[ "$(snapshot "${root}")" == "${before}" ]] && ! grep -q '^created' "${work}/boot2.out"; then
	pass "second bootstrap-config changes nothing"
else
	fail "second bootstrap-config changed the tree"
fi
if [[ ! -e "${root}/lab/state" && ! -e "${root}/lab/backups" ]]; then
	pass "no state/ or backups/ created"
else
	fail "state/ or backups/ created"
fi

nodecl="${work}/nodecl"
pushd "${work}" >/dev/null
run nodecl "${bin}" bootstrap-config --env lab --target vsphere-lab --config-dir "${nodecl}"
popd >/dev/null
expect "bootstrap-config without a declaration succeeds" 0 nodecl
if grep -qi 'warn' "${work}/nodecl.out"; then pass "missing declaration is a warning"; else fail "no warning without a declaration"; fi

printf 'version: 1\nconsumer: provision-x\ntarget:\n  - {name: HOST_IP, required: true, description: d}\n' >"${work}/bad.yaml"
run baddecl "${bin}" bootstrap-config --env lab --target vsphere-lab --declaration "${work}/bad.yaml" --config-dir "${work}/badroot"
if [[ "${rc}" != 0 && ! -e "${work}/badroot" ]]; then
	pass "invalid declaration is an error and writes nothing (exit ${rc})"
else
	fail "invalid declaration: exit ${rc}, root exists: $([[ -e "${work}/badroot" ]] && echo yes || echo no)"
fi

vars="${root}/lab/targets/vsphere-lab/vars.sh"
hostsf="${root}/lab/targets/vsphere-lab/hosts/provision-example.sh"
grep -v '^# GOVC_DATACENTER=$' "${vars}" >"${work}/vars.trim"
cp -p "${vars}" "${work}/vars.orig"
cat "${work}/vars.trim" >"${vars}"
sum="$(cksum <"${vars}")"
run diverge pc bootstrap-config --env lab --target vsphere-lab --declaration "${decl}"
if [[ "${rc}" == 0 && "$(cksum <"${vars}")" == "${sum}" ]] && grep -q '# GOVC_DATACENTER=' "${work}/diverge.out"; then
	pass "divergent file reported with the lines to add, not overwritten"
else
	fail "divergent file (exit ${rc})"
	show diverge
fi
cat "${work}/vars.orig" >"${vars}"

# --- fill the skeleton ---------------------------------------------------
set_key "${vars}" GOVC_URL https://vcenter.e2e.invalid/sdk
set_key "${vars}" GOVC_DATACENTER dc1
set_key "${hostsf}" HOST_IP 192.0.2.21

# --- credential ----------------------------------------------------------
secret="e2e-stdin-secret-$$-value"
secrets+=("${secret}")
run cred1 pc credential --env lab --consumer provision-example EXAMPLE_DB_USER=example EXAMPLE_DB_PASS
add_secret_file "${root}/lab/credentials/provision-example/EXAMPLE_DB_PASS"
expect "credential creates consumer files" 0 cred1
# The value goes through a 0600 file and standard input, never an argument.
(umask 077 && printf '%s\n' "${secret}" >"${work}/secret.in")
run cred2 pc credential --env lab --target vsphere-lab --identity vcenter-access-admin GOVC_PASSWORD=- <"${work}/secret.in"
expect "credential reads a target secret from standard input" 0 cred2
creds="${root}/lab/credentials"
user_sum="$(cksum <"${creds}/provision-example/EXAMPLE_DB_USER")"
run cred3 pc credential --env lab --consumer provision-example EXAMPLE_DB_USER=other
if [[ "${rc}" == 0 && "$(cksum <"${creds}/provision-example/EXAMPLE_DB_USER")" == "${user_sum}" ]] && grep -q '^exists' "${work}/cred3.out"; then
	pass "credential never overwrites"
else
	fail "credential overwrote or failed (exit ${rc})"
fi
run cred4 pc credential --env lab --target vsphere-lab --identity terraform-provisioner GOVC_PASSWORD
expect "deprecated identity is a usage error" 2 cred4

# --- doctor --------------------------------------------------------------
run doc1 pc doctor --env lab --declaration "${decl}"
expect "doctor accepts the filled tree" 0 doc1
run docjson pc doctor --env lab --declaration "${decl}" --format json
expect "doctor --format json" 0 docjson
# json_ok <file>: the file is one JSON object with ok == true and an
# automation_key.fingerprint of the form SHA256:<43 base64 characters>.
json_ok() {
	if command -v python3 >/dev/null; then
		python3 -I -c 'import json, re, sys
d = json.load(open(sys.argv[1]))
fp = (d.get("automation_key") or {}).get("fingerprint", "")
sys.exit(0 if d.get("ok") is True and re.fullmatch(r"SHA256:[A-Za-z0-9+/]{43}", fp) else 1)' "$1"
	elif command -v jq >/dev/null; then
		jq -e '.ok == true and (.automation_key.fingerprint | test("^SHA256:[A-Za-z0-9+/]{43}$"))' "$1" >/dev/null
	else
		echo "python3 or jq is required to parse the doctor JSON" >&2
		return 1
	fi
}
if json_ok "${work}/docjson.out"; then
	pass "JSON parses, ok is true, fingerprint is SHA256"
else
	fail "doctor JSON does not parse or lacks ok/fingerprint"
	show docjson
fi
leaked=0
for f in "${creds}/provision-example/EXAMPLE_DB_PASS" "${creds}/target-vsphere-lab/vcenter-access-admin/GOVC_PASSWORD"; do
	grep -qF -- "$(cat "${f}")" "${work}/docjson.out" "${work}/doc1.out" && leaked=1
done
grep -qF -- "$(sed -n 2p "${creds}/automation/ssh_ed25519")" "${work}/docjson.out" "${work}/doc1.out" && leaked=1
if ((leaked == 0)); then pass "doctor output holds no secret"; else fail "doctor output holds a secret"; fi

# broken <description> <expected text> <command>...: copies the healthy tree,
# breaks it with the command (run with the copy as $1), and expects doctor
# to fail with the text.
broken() {
	local desc="$1" want="$2" copy="${work}/broken"
	shift 2
	rm -rf "${copy}"
	cp -Rp "${root}" "${copy}"
	"$@" "${copy}/lab"
	run broken "${bin}" doctor --env lab --declaration "${decl}" --config-dir "${copy}"
	if [[ "${rc}" == 1 ]] && grep -qF -- "${want}" "${work}/broken.out"; then
		pass "doctor fails on ${desc}"
	else
		fail "doctor on ${desc} (exit ${rc}, want 1 and '${want}')"
		show broken
	fi
}
append() { printf '%s\n' "$2" >>"$1/targets/vsphere-lab/$3"; }
# shellcheck disable=SC2016 # literal $(...) written into the tree
b_subst() { append "$1" 'GOVC_FOLDER=$(id)' vars.sh; }
b_tilde() { append "$1" 'HOST_DATA_DIR=~/data' hosts/provision-example.sh; }
b_dup() { append "$1" 'GOVC_DATACENTER=dc2' vars.sh; }
b_missing() { grep -v '^GOVC_DATACENTER=' "$1/targets/vsphere-lab/vars.sh" >"$1/v" && cat "$1/v" >"$1/targets/vsphere-lab/vars.sh" && rm "$1/v"; }
b_loose() { (umask 077 && printf 'x' >"$1/credentials/target-vsphere-lab/GOVC_PASSWORD"); }
# shellcheck disable=SC2016 # description text
broken '$(...)' 'GOVC_FOLDER: only KEY=value lines' b_subst
broken 'an unquoted ~' 'HOST_DATA_DIR: only KEY=value lines' b_tilde
broken 'a duplicate key' 'GOVC_DATACENTER: key set twice' b_dup
broken 'a declared key that is missing' 'GOVC_DATACENTER: required key' b_missing
broken 'a file directly under credentials/target-<t>/' 'a file directly under credentials/target-vsphere-lab/' b_loose

if ((failures > 0)); then
	printf '%d check(s) failed\n' "${failures}"
	exit 1
fi
printf 'all checks passed\n'
