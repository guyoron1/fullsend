#!/usr/bin/env bash
# upload-traces — upload OTLP traces to a collector endpoint via otelcol-contrib
#
# Usage: upload-traces <run-dir> --endpoint <url> [--header key=value]...
#
# Headers are written directly into the collector YAML config under
# exporters.otlphttp.headers. The OTEL_EXPORTER_OTLP_HEADERS env var
# is an SDK concept that otelcol-contrib ignores, so this script injects
# headers into the config file instead.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE="${SCRIPT_DIR}/upload-traces-otelcol-config.yaml"

usage() {
  echo "Usage: $0 <run-dir> --endpoint <url> [--header key=value]..." >&2
  exit 1
}

endpoint=""
headers=()
run_dir=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --endpoint)
      endpoint="${2:?--endpoint requires a value}"
      shift 2
      ;;
    --header)
      headers+=("${2:?--header requires a value}")
      shift 2
      ;;
    --help|-h)
      usage
      ;;
    -*)
      echo "Unknown flag: $1" >&2
      usage
      ;;
    *)
      if [[ -z "${run_dir}" ]]; then
        run_dir="$1"
        shift
      else
        echo "Unexpected argument: $1" >&2
        usage
      fi
      ;;
  esac
done

if [[ -z "${run_dir}" ]] || [[ -z "${endpoint}" ]]; then
  usage
fi

if [[ ! -d "${run_dir}" ]]; then
  echo "Error: ${run_dir} is not a directory" >&2
  exit 1
fi

if ! command -v otelcol-contrib &>/dev/null; then
  echo "Error: otelcol-contrib is not installed" >&2
  exit 1
fi

export REPLAY_ENDPOINT="${endpoint}"
REPLAY_TRACE_DIR="$(cd "${run_dir}" && pwd)"
export REPLAY_TRACE_DIR

# generate_config writes the collector config to stdout, injecting any
# --header flags as YAML keys under exporters.otlphttp.headers.
generate_config() {
  while IFS= read -r line; do
    printf '%s\n' "${line}"
    # Insert headers block right after the endpoint line.
    if [[ "${line}" == *"endpoint:"*"REPLAY_ENDPOINT"* ]] \
        && [[ ${#headers[@]} -gt 0 ]]; then
      echo "    headers:"
      for h in "${headers[@]}"; do
        local key="${h%%=*}"
        local value="${h#*=}"
        echo "      ${key}: \"${value}\""
      done
    fi
  done < "${TEMPLATE}"
}

config="${TEMPLATE}"

if [[ ${#headers[@]} -gt 0 ]]; then
  tmpconfig="$(mktemp /tmp/otelcol-config-XXXXXX.yaml)"
  trap 'rm -f "${tmpconfig}"' EXIT
  generate_config > "${tmpconfig}"
  config="${tmpconfig}"
fi

echo "Uploading traces from ${run_dir} to ${endpoint}..."
otelcol-contrib --config "${config}"
