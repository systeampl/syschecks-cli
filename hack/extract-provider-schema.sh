#!/usr/bin/env bash
# extract-provider-schema.sh — list the writable (Required or Optional)
# attributes of a terraform-provider-systeam resource, in schema order, so
# internal/generate/schema.go's `specs` map can be re-derived by hand when
# the provider schema changes.
#
# It does NOT regenerate schema.go automatically — schema.go also carries
# hand-picked data (Secret flags, SecretMapKeys, AttrKind) that isn't fully
# recoverable from a line-oriented grep. Use this script's output as the
# worklist: for each name printed, look at resource.go to decide its
# AttrKind and whether it should be Secret (Sensitive: true in the
# provider ~ a candidate for Secret: true / SecretMapKeys here).
#
# Usage:
#   hack/extract-provider-schema.sh <provider-repo-path> <resource-dir>
#   hack/extract-provider-schema.sh /home/destine/GIT/wlasne/terraform-provider-systeam check
#   hack/extract-provider-schema.sh /home/destine/GIT/wlasne/terraform-provider-systeam notification_channel
#   hack/extract-provider-schema.sh /home/destine/GIT/wlasne/terraform-provider-systeam team
#
# How it works: resource.go declares each attribute as a top-level entry in
# the Schema()'s `Attributes: map[string]schema.Attribute{...}` literal —
# `"<name>": schema.<Kind>Attribute{`. We scan from that line to the matching
# closing brace of the attribute's own literal (tracking `{`/`}` depth) and
# check whether Required/Optional/Computed appear at that depth. An attribute
# is writable iff it has `Required: true` or `Optional: true`; pure-Computed
# attributes (id, created_at, updated_at, status, uuid, member_count, ...)
# are dropped.
set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "usage: $0 <provider-repo-path> <resource-dir>" >&2
	exit 2
fi

repo="$1"
res="$2"
file="$repo/internal/resources/$res/resource.go"

if [[ ! -f "$file" ]]; then
	echo "no such file: $file" >&2
	exit 1
fi

awk '
	# Match the start of an attribute entry: "name": schema.XAttribute{
	/^\t+"[a-z0-9_]+":[[:space:]]*schema\.[A-Za-z0-9]+Attribute\{/ {
		match($0, /"[a-z0-9_]+"/)
		name = substr($0, RSTART+1, RLENGTH-2)
		match($0, /schema\.[A-Za-z0-9]+Attribute/)
		kind = substr($0, RSTART, RLENGTH)
		depth = gsub(/\{/, "{") - gsub(/\}/, "}")
		required = 0
		optional = 0
		computed = 0
		if ($0 ~ /Required:[[:space:]]*true/) required = 1
		if ($0 ~ /Optional:[[:space:]]*true/) optional = 1
		if ($0 ~ /Computed:[[:space:]]*true/) computed = 1
		if (depth <= 0) {
			# single-line attribute literal (e.g. the compact check.go fields)
			if (required || optional) print name "\t" kind "\twritable"
			next
		}
		in_attr = 1
		next
	}
	in_attr {
		depth += gsub(/\{/, "{") - gsub(/\}/, "}")
		if ($0 ~ /Required:[[:space:]]*true/) required = 1
		if ($0 ~ /Optional:[[:space:]]*true/) optional = 1
		if ($0 ~ /Computed:[[:space:]]*true/) computed = 1
		if (depth <= 0) {
			in_attr = 0
			if (required || optional) print name "\t" kind "\twritable"
		}
	}
' "$file"
