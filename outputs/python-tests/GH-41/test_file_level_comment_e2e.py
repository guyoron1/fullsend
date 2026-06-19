"""
File-Level Comment End-to-End Tests -- GH-41

Tests that file-level comments (subject_type: "file") are correctly
handled by the GitHub Pull Request Review API.

STP Reference: outputs/stp/GH-41/GH-41_test_plan.md
Jira: GH-41
"""
import json
import logging
import os
import subprocess
import time

import pytest

logger = logging.getLogger(__name__)

GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN", "")
REPO = os.environ.get("TEST_REPO", "")
PR_NUMBER = os.environ.get("TEST_PR_NUMBER", "")

pytestmark = [
    pytest.mark.e2e,
    pytest.mark.tier2,
    pytest.mark.skipif(
        not all([GITHUB_TOKEN, REPO, PR_NUMBER]),
        reason="Requires GITHUB_TOKEN, TEST_REPO, and TEST_PR_NUMBER environment variables",
    ),
]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _gh_api(method, endpoint, data=None):
    """Call GitHub API via gh CLI and return the CompletedProcess."""
    cmd = ["gh", "api", "-X", method, endpoint]
    if data is not None:
        input_data = json.dumps(data)
        cmd.extend(["--input", "-"])
        result = subprocess.run(
            cmd,
            input=input_data,
            capture_output=True,
            text=True,
            timeout=30,
        )
    else:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
    return result


def _get_pr_files():
    """Return filenames changed in the test PR."""
    result = _gh_api("GET", f"/repos/{REPO}/pulls/{PR_NUMBER}/files")
    assert result.returncode == 0, f"Failed to list PR files: {result.stderr}"
    return [f["filename"] for f in json.loads(result.stdout)]


def _dismiss_review(review_id):
    """Best-effort dismiss a review so tests leave no lasting side-effects."""
    try:
        _gh_api(
            "PUT",
            f"/repos/{REPO}/pulls/{PR_NUMBER}/reviews/{review_id}/dismissals",
            data={"message": "Automated test cleanup"},
        )
        logger.info("Dismissed review %s", review_id)
    except Exception:
        logger.warning("Could not dismiss review %s", review_id, exc_info=True)


def _delete_review_comment(comment_id):
    """Best-effort delete a pull request review comment."""
    try:
        _gh_api(
            "DELETE",
            f"/repos/{REPO}/pulls/comments/{comment_id}",
        )
        logger.info("Deleted review comment %s", comment_id)
    except Exception:
        logger.warning(
            "Could not delete review comment %s", comment_id, exc_info=True
        )


def _submit_file_level_review(file_path, body_text, event="COMMENT"):
    """Submit a PR review containing a single file-level comment.

    A file-level comment has ``subject_type: "file"`` and no ``line`` field.

    Returns:
        tuple: (review_id, parsed response dict)
    """
    payload = {
        "event": event,
        "body": f"GH-41 automated e2e test review ({body_text})",
        "comments": [
            {
                "path": file_path,
                "body": body_text,
                "subject_type": "file",
            }
        ],
    }
    result = _gh_api(
        "POST",
        f"/repos/{REPO}/pulls/{PR_NUMBER}/reviews",
        data=payload,
    )
    logger.info(
        "Review submission rc=%d stdout=%s stderr=%s",
        result.returncode,
        result.stdout[:300],
        result.stderr[:300],
    )
    assert result.returncode == 0, (
        f"Review submission failed (rc={result.returncode}): {result.stderr}"
    )
    response = json.loads(result.stdout)
    review_id = response.get("id")
    assert review_id is not None, "Review response missing 'id' field"
    return review_id, response


def _get_review_comments(review_id):
    """Fetch all comments belonging to a specific review."""
    result = _gh_api(
        "GET",
        f"/repos/{REPO}/pulls/{PR_NUMBER}/reviews/{review_id}/comments",
    )
    assert result.returncode == 0, (
        f"Failed to fetch review comments: {result.stderr}"
    )
    return json.loads(result.stdout)


def _get_pr_review_comments():
    """Fetch all review comments on the PR."""
    result = _gh_api(
        "GET",
        f"/repos/{REPO}/pulls/{PR_NUMBER}/comments",
    )
    assert result.returncode == 0, (
        f"Failed to fetch PR review comments: {result.stderr}"
    )
    return json.loads(result.stdout)


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestFileLevelCommentPersistence:
    """TS-GH-41-003: File-level comments survive review re-submission.

    Verifies that file-level comments persist when a PR review is
    re-submitted. This tests the full GitHub API lifecycle for
    file-level comments created via subject_type: "file".
    """

    def test_file_level_comments_persist_after_resubmission(self):
        """[test_id:TS-GH-41-003] File-level comments survive review re-submission.

        Steps:
        1. Identify a file in the PR diff.
        2. Submit a review with a file-level comment (subject_type: "file").
        3. Verify the file-level comment exists on the review.
        4. Submit a second review with another file-level comment.
        5. Verify that both sets of file-level comments are present on the PR.
        """
        files = _get_pr_files()
        assert len(files) > 0, "Test PR has no changed files"
        target_file = files[0]
        logger.info("Using target file: %s", target_file)

        review_ids = []
        comment_ids = []

        try:
            # -- Step 1: Submit first review with a file-level comment ------
            first_body = (
                "TS-GH-41-003 first submission: "
                "file-level comment persistence test"
            )
            review_id_1, _ = _submit_file_level_review(target_file, first_body)
            review_ids.append(review_id_1)
            logger.info("First review created: %s", review_id_1)

            # Allow a brief propagation window
            time.sleep(2)

            # Verify the file-level comment exists on the first review
            comments_1 = _get_review_comments(review_id_1)
            assert len(comments_1) > 0, (
                "First review has no comments"
            )
            file_comments_1 = [
                c for c in comments_1
                if c.get("subject_type") == "file" or c.get("path") == target_file
            ]
            assert len(file_comments_1) > 0, (
                "No file-level comment found on first review"
            )
            for c in file_comments_1:
                comment_ids.append(c["id"])

            # -- Step 2: Submit second review with another file-level comment
            second_body = (
                "TS-GH-41-003 second submission: "
                "verifying persistence across reviews"
            )
            review_id_2, _ = _submit_file_level_review(target_file, second_body)
            review_ids.append(review_id_2)
            logger.info("Second review created: %s", review_id_2)

            time.sleep(2)

            # -- Step 3: Verify both reviews' comments are present ----------
            # Check that the first review's comments are still accessible
            comments_1_after = _get_review_comments(review_id_1)
            assert len(comments_1_after) > 0, (
                "First review comments disappeared after second submission"
            )

            # Check the second review has its own file-level comment
            comments_2 = _get_review_comments(review_id_2)
            assert len(comments_2) > 0, (
                "Second review has no comments"
            )
            for c in comments_2:
                comment_ids.append(c["id"])

            # Verify file-level comments from both reviews are on the PR
            all_comments = _get_pr_review_comments()
            our_comments = [
                c for c in all_comments
                if "TS-GH-41-003" in c.get("body", "")
            ]
            assert len(our_comments) >= 2, (
                f"Expected at least 2 file-level comments from both reviews, "
                f"found {len(our_comments)}"
            )
            logger.info(
                "Verified %d file-level comments persist across submissions",
                len(our_comments),
            )

        finally:
            # Cleanup: dismiss reviews and delete comments
            for rid in review_ids:
                _dismiss_review(rid)
            for cid in comment_ids:
                _delete_review_comment(cid)


class TestGitHubAPIAcceptsFileLevelPayload:
    """TS-GH-41-014: GitHub API accepts file-level comment payload.

    Verifies that the GitHub Pull Request Review API accepts a review
    payload containing a comment with subject_type: "file" and no line
    field. This confirms the API contract that the application relies on.
    """

    def test_github_api_accepts_file_level_comment(self):
        """[test_id:TS-GH-41-014] GitHub API accepts file-level comment payload.

        Steps:
        1. Build a review payload with a comment that has subject_type: "file"
           and no ``line`` field.
        2. Submit the payload via POST /repos/{owner}/{repo}/pulls/{pr}/reviews.
        3. Assert the API returns successfully (rc 0 from gh, HTTP 200).
        4. Query the review's comments and verify the file-level comment exists.
        """
        files = _get_pr_files()
        assert len(files) > 0, "Test PR has no changed files"
        target_file = files[0]
        logger.info("Using target file: %s", target_file)

        review_id = None
        comment_ids = []

        try:
            # -- Step 1: Build and submit a file-level comment payload ------
            comment_body = (
                "TS-GH-41-014: Verifying GitHub API accepts "
                "subject_type file payload"
            )
            payload = {
                "event": "COMMENT",
                "body": "GH-41 e2e: file-level comment API acceptance test",
                "comments": [
                    {
                        "path": target_file,
                        "body": comment_body,
                        "subject_type": "file",
                    }
                ],
            }
            result = _gh_api(
                "POST",
                f"/repos/{REPO}/pulls/{PR_NUMBER}/reviews",
                data=payload,
            )

            # -- Step 2: Assert HTTP 200 (gh returns rc 0 for 200) ----------
            assert result.returncode == 0, (
                f"GitHub API rejected file-level comment payload "
                f"(rc={result.returncode}): {result.stderr}"
            )
            response = json.loads(result.stdout)
            review_id = response.get("id")
            assert review_id is not None, "Response missing review id"
            logger.info(
                "Review %s created successfully with file-level comment",
                review_id,
            )

            # Verify response state is as expected
            assert response.get("state") is not None, (
                "Response missing 'state' field"
            )

            # -- Step 3: Verify the file-level comment is visible -----------
            time.sleep(2)
            comments = _get_review_comments(review_id)
            assert len(comments) > 0, (
                "Review has no comments despite successful submission"
            )

            file_level_found = False
            for comment in comments:
                comment_ids.append(comment["id"])
                # GitHub returns subject_type for file-level comments
                if (
                    comment.get("subject_type") == "file"
                    or (
                        comment.get("path") == target_file
                        and comment.get("line") is None
                    )
                ):
                    file_level_found = True
                    logger.info(
                        "File-level comment confirmed: id=%s path=%s subject_type=%s",
                        comment["id"],
                        comment.get("path"),
                        comment.get("subject_type"),
                    )

            assert file_level_found, (
                "No file-level comment found in the review. "
                f"Comments returned: {json.dumps(comments, indent=2)[:500]}"
            )

        finally:
            # Cleanup: dismiss review and delete comments
            if review_id is not None:
                _dismiss_review(review_id)
            for cid in comment_ids:
                _delete_review_comment(cid)
