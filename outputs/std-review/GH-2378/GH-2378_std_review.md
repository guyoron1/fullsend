# STD Review Report: GH-2378

**Date:** 2026-06-21
**Verdict:** BLOCKED
**Reason:** STD artifact not found

## Summary

The STD review for GH-2378 cannot proceed because the required STD YAML file does not exist.

**Expected path:** `outputs/std/GH-2378/GH-2378_test_description.yaml`
**Actual state:** The `outputs/std/` directory does not exist in the repository.

## Artifacts Checked

| Artifact | Path | Status |
|:---------|:-----|:-------|
| STD YAML | `outputs/std/GH-2378/GH-2378_test_description.yaml` | NOT FOUND |
| Go stubs | `outputs/std/GH-2378/go-tests/*_stubs_test.go` | NOT FOUND |
| Python stubs | `outputs/std/GH-2378/python-tests/test_*_stubs.py` | NOT FOUND |
| STP | `outputs/stp/GH-2378/GH-2378_test_plan.md` | FOUND |

## Recommendation

The STP exists at `outputs/stp/GH-2378/GH-2378_test_plan.md`, but the STD has not yet been generated. Run the `std-builder` command for GH-2378 before requesting an STD review.
