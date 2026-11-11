#!/usr/bin/env bash
# Line check of the provision-semaphoreui loader, copied verbatim from
# hosts/lib/config.sh config_check_line_format (origin/lab b8aaab2). Prints
# "ok" or "refused" for each line of the file in $1. Used by
# TestShellFileAgreesWithLoader so the Go parser and the loader agree.
sq="'[^']*'"
dq='"[^"$`\\]*"'
bare="[^[:space:]'\"\$\`;&|<>()\\#]*"
assign="^[[:space:]]*(export[[:space:]]+)?[A-Za-z_][A-Za-z0-9_]*=(${sq}|${dq}|${bare})([[:space:]]+#.*)?[[:space:]]*\$"
while IFS= read -r line || [[ -n "${line}" ]]; do
	if [[ "${line}" =~ ^[[:space:]]*(#.*)?$ ]] || [[ "${line}" =~ ${assign} ]]; then
		echo ok
	else
		echo refused
	fi
done <"$1"
