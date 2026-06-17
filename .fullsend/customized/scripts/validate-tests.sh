#!/usr/bin/env bash
# validate-tests.sh — Validate that the test generator produced expected output.
# Checks for Go test files, Python test files, or any test output.
set -euo pipefail

OUTPUT_DIR="${1:-${FULLSEND_OUTPUT_DIR:-$(pwd)/output}}"

if [ ! -d "$OUTPUT_DIR" ]; then
    mkdir -p "$OUTPUT_DIR"
fi

errors=0
total_tests=0

# --- Check for Go test files ---
go_tests=$(find "$OUTPUT_DIR" -name "*_test.go" 2>/dev/null | wc -l)
if [ "$go_tests" -gt 0 ]; then
    echo "OK: found $go_tests Go test file(s)"
    total_tests=$((total_tests + go_tests))
else
    # Check target repo fallback locations
    for search_root in /sandbox/workspace/pr-repo/outputs/go-tests /sandbox/workspace/target-repo/outputs/go-tests; do
        if [ -d "$search_root" ]; then
            pr_tests=$(find "$search_root" -name "*_test.go" 2>/dev/null | head -1)
            if [ -n "$pr_tests" ]; then
                pr_out_dir=$(dirname "$pr_tests")
                echo "Found Go tests in target repo: $pr_out_dir"
                cp -r "$pr_out_dir"/* "$OUTPUT_DIR/" 2>/dev/null || true
                go_tests=$(find "$OUTPUT_DIR" -name "*_test.go" 2>/dev/null | wc -l)
                total_tests=$((total_tests + go_tests))
                break
            fi
        fi
    done
fi

# --- Check for Python test files ---
py_tests=$(find "$OUTPUT_DIR" -name "test_*.py" 2>/dev/null | wc -l)
if [ "$py_tests" -gt 0 ]; then
    echo "OK: found $py_tests Python test file(s)"
    total_tests=$((total_tests + py_tests))
else
    # Check target repo fallback locations
    for search_root in /sandbox/workspace/pr-repo/outputs/python-tests /sandbox/workspace/target-repo/outputs/python-tests; do
        if [ -d "$search_root" ]; then
            pr_tests=$(find "$search_root" -name "test_*.py" 2>/dev/null | head -1)
            if [ -n "$pr_tests" ]; then
                pr_out_dir=$(dirname "$pr_tests")
                echo "Found Python tests in target repo: $pr_out_dir"
                cp -r "$pr_out_dir"/* "$OUTPUT_DIR/" 2>/dev/null || true
                py_tests=$(find "$OUTPUT_DIR" -name "test_*.py" 2>/dev/null | wc -l)
                total_tests=$((total_tests + py_tests))
                break
            fi
        fi
    done
fi

# --- Check for any other test files ---
other_tests=$(find "$OUTPUT_DIR" -type f \( -name "*_test.*" -o -name "test_*.*" \) \
    ! -name "*_test.go" ! -name "test_*.py" 2>/dev/null | wc -l)
if [ "$other_tests" -gt 0 ]; then
    echo "OK: found $other_tests other test file(s)"
    total_tests=$((total_tests + other_tests))
fi

# --- Require at least some test output ---
if [ "$total_tests" -eq 0 ]; then
    echo "FAIL: no test files found in output"
    errors=$((errors + 1))
fi

# --- Check for summary ---
if [ -f "$OUTPUT_DIR/summary.yaml" ]; then
    echo "OK: summary.yaml found"
else
    echo "WARN: summary.yaml not found (non-fatal)"
fi

if [ "$errors" -gt 0 ]; then
    echo "FAIL: $errors validation error(s)"
    exit 1
fi

echo "PASS: output validated — $total_tests test file(s) (Go: ${go_tests:-0}, Python: ${py_tests:-0}, Other: ${other_tests:-0})"
