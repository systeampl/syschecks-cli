#!/usr/bin/env bash
# extract-provider-schema.sh — list the writable (Required or Optional)
# attributes of a terraform-provider-systeam resource, in schema order, along
# with whether the provider marks each Sensitive: true, so
# internal/generate/schema.go's `specs` map can be re-derived by hand when
# the provider schema changes.
#
# It does NOT regenerate schema.go automatically — schema.go also carries
# hand-picked data (SecretMapKeys, AttrKind) that isn't fully recoverable from
# a line-oriented grep. Use this script's output as the worklist: for each
# name printed, look at resource.go to decide its AttrKind, and set
# Secret: true in schema.go for every row whose 4th column is "sensitive"
# (Attr.Secret must mirror the provider's Sensitive: true 1:1 — see the
# package doc comment in schema.go for why).
#
# EXCLUDED (Phase 1): this script will list check's api_scenario_steps,
# oidc_config, dns_records_config, and response_assertions as writable
# string attrs, but do NOT add them to schema.go's check Attrs. The provider
# declares them jsontypes.Normalized StringAttribute, but the syschecks SDK
# returns them already JSON-decoded (map[string]any/[]any, not a JSON
# string), so rendering them via schema.go's plain AttrString path hands
# renderResource an HCL object/tuple for a string attribute — which
# `terraform plan` hard-errors on for every check that has one set. Phase 1
# defers complex/nested attrs anyway, so they're intentionally omitted from
# the spec until render.go supports jsonencode(...)-wrapping a JSON-blob
# attr kind.
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
# check whether Required/Optional/Computed/Sensitive appear at that depth. An
# attribute is writable iff it has `Required: true` or `Optional: true`;
# pure-Computed attributes (id, created_at, updated_at, status, uuid,
# member_count, ...) are dropped. The last column is "sensitive" iff
# `Sensitive: true` appears anywhere in the attribute's own literal.
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
		sensitive = 0
		if ($0 ~ /Required:[[:space:]]*true/) required = 1
		if ($0 ~ /Optional:[[:space:]]*true/) optional = 1
		if ($0 ~ /Computed:[[:space:]]*true/) computed = 1
		if ($0 ~ /Sensitive:[[:space:]]*true/) sensitive = 1
		if (depth <= 0) {
			# single-line attribute literal (e.g. the compact check.go fields)
			if (required || optional) print name "\t" kind "\twritable\t" (sensitive ? "sensitive" : "-")
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
		if ($0 ~ /Sensitive:[[:space:]]*true/) sensitive = 1
		if (depth <= 0) {
			in_attr = 0
			if (required || optional) print name "\t" kind "\twritable\t" (sensitive ? "sensitive" : "-")
		}
	}
' "$file"
