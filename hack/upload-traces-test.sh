#!/usr/bin/env bash
# upload-traces-test.sh — verify that --header flags are injected into the
# collector YAML config, not set as OTEL_EXPORTER_OTLP_HEADERS.
#
# Run from the repo root: bash hack/upload-traces-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/upload-traces.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# Create a fake run directory.
mkdir -p "${TMPDIR}/run"

# Mock otelcol-contrib: capture the --config path and exit.
MOCK_BIN="${TMPDIR}/bin"
mkdir -p "${MOCK_BIN}"
cat > "${MOCK_BIN}/otelcol-contrib" <<'EOF'
#!/usr/bin/env bash
# Save the config path for assertion.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --config) cp "$2" "${OTELCOL_CONFIG_CAPTURE}"; exit 0 ;;
    *) shift ;;
  esac
done
exit 1
EOF
chmod +x "${MOCK_BIN}/otelcol-contrib"

# --- Test 1: single header ---
echo "TEST 1: single header appears in config"
CAPTURE="${TMPDIR}/config-single.yaml"
OTELCOL_CONFIG_CAPTURE="${CAPTURE}" \
  PATH="${MOCK_BIN}:${PATH}" \
  bash "${SCRIPT}" "${TMPDIR}/run" --endpoint http://localhost:4318 \
    --header x-mlflow-experiment-id=42 > /dev/null

if grep -q 'x-mlflow-experiment-id: "42"' "${CAPTURE}"; then
  echo "  PASS"
else
  echo "  FAIL: header not found in generated config"
  cat "${CAPTURE}"
  FAILURES=$((FAILURES + 1))
fi

if grep -q 'headers:' "${CAPTURE}"; then
  echo "  PASS: headers block present"
else
  echo "  FAIL: headers block missing"
  FAILURES=$((FAILURES + 1))
fi

# --- Test 2: multiple headers ---
echo "TEST 2: multiple headers appear in config"
CAPTURE="${TMPDIR}/config-multi.yaml"
OTELCOL_CONFIG_CAPTURE="${CAPTURE}" \
  PATH="${MOCK_BIN}:${PATH}" \
  bash "${SCRIPT}" "${TMPDIR}/run" --endpoint http://localhost:4318 \
    --header x-mlflow-experiment-id=42 --header x-custom=val > /dev/null

if grep -q 'x-mlflow-experiment-id: "42"' "${CAPTURE}" \
    && grep -q 'x-custom: "val"' "${CAPTURE}"; then
  echo "  PASS"
else
  echo "  FAIL: not all headers found"
  cat "${CAPTURE}"
  FAILURES=$((FAILURES + 1))
fi

# --- Test 3: no headers (backward compat) ---
echo "TEST 3: no headers — config has no headers block"
CAPTURE="${TMPDIR}/config-none.yaml"
OTELCOL_CONFIG_CAPTURE="${CAPTURE}" \
  PATH="${MOCK_BIN}:${PATH}" \
  bash "${SCRIPT}" "${TMPDIR}/run" --endpoint http://localhost:4318 > /dev/null

if grep -q 'headers:' "${CAPTURE}"; then
  echo "  FAIL: unexpected headers block in config"
  cat "${CAPTURE}"
  FAILURES=$((FAILURES + 1))
else
  echo "  PASS"
fi

# --- Test 4: header value with equals sign ---
echo "TEST 4: header value containing equals sign"
CAPTURE="${TMPDIR}/config-equals.yaml"
OTELCOL_CONFIG_CAPTURE="${CAPTURE}" \
  PATH="${MOCK_BIN}:${PATH}" \
  bash "${SCRIPT}" "${TMPDIR}/run" --endpoint http://localhost:4318 \
    --header Authorization=Bearer=abc123 > /dev/null

if grep -q 'Authorization: "Bearer=abc123"' "${CAPTURE}"; then
  echo "  PASS"
else
  echo "  FAIL: header with equals in value not handled"
  cat "${CAPTURE}"
  FAILURES=$((FAILURES + 1))
fi

# --- Summary ---
echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "FAILED: ${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
