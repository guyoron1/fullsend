"""
File-Level Comment End-to-End Tests

STP Reference: outputs/stp/GH-41/GH-41_test_plan.md
Jira: GH-41
"""


class TestFileLevelCommentGitHubAPI:
    """
    Tests for file-level comment integration with the GitHub Pull Request Review API.

    Preconditions:
        - GitHub test repository with an open PR containing files with out-of-hunk changes
        - GitHub token with pull request review permissions
        - fullsend binary built from PR #41 branch
    """

    __test__ = False

    def test_file_level_comments_survive_review_resubmission(self):
        """
        Test that file-level comments survive review re-submission.

        Preconditions:
            - Open PR with at least one file whose findings fall outside diff hunks

        Steps:
            1. Run fullsend post-review against the test PR with out-of-hunk findings
            2. Verify file-level comments exist on the PR via GitHub API
            3. Re-run fullsend post-review against the same PR

        Expected:
            - File-level comments are present after re-submission
        """
        pass

    def test_github_api_accepts_file_level_comment_payload(self):
        """
        Test that GitHub API accepts file-level comment payload with subject_type 'file'.

        Preconditions:
            - Review payload constructed with subject_type: "file" for Line=0 comments

        Steps:
            1. Submit PR review containing a file-level comment via GitHub API
            2. Query PR comments via GitHub API

        Expected:
            - API returns HTTP 200 for the review submission
            - File-level comment is visible on the PR
        """
        pass
