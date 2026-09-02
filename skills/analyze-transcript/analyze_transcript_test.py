#!/usr/bin/env python3
"""Tests for analyze-transcript.py."""

from __future__ import annotations

# The script uses a hyphenated filename, so use importlib to load it.
import importlib.util
import json
import os

import pytest

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
_SCRIPT_PATH = os.path.join(_SCRIPT_DIR, "analyze-transcript.py")
_spec = importlib.util.spec_from_file_location("analyze_transcript", _SCRIPT_PATH)
assert _spec and _spec.loader
at = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(at)

TESTDATA = os.path.join(_SCRIPT_DIR, "testdata")


def _fixture(name: str) -> str:
    return os.path.join(TESTDATA, name)


# ---------------------------------------------------------------------------
# parse_line_range
# ---------------------------------------------------------------------------


class TestParseLineRange:
    def test_none_input(self):
        assert at.parse_line_range(None) is None

    def test_empty_string(self):
        assert at.parse_line_range("") is None

    def test_single_number(self):
        assert at.parse_line_range("5") == (5, 5)

    def test_range_with_both_bounds(self):
        assert at.parse_line_range("3-10") == (3, 10)

    def test_range_open_end(self):
        assert at.parse_line_range("7-") == (7, None)

    def test_range_open_start(self):
        assert at.parse_line_range("-5") == (0, 5)


# ---------------------------------------------------------------------------
# truncate
# ---------------------------------------------------------------------------


class TestTruncate:
    def test_short_text_unchanged(self):
        assert at.truncate("hello", 10) == "hello"

    def test_exact_length_unchanged(self):
        assert at.truncate("hello", 5) == "hello"

    def test_long_text_truncated(self):
        assert at.truncate("hello world", 5) == "hello..."

    def test_none_max_width_no_truncation(self):
        assert at.truncate("hello world", None) == "hello world"

    def test_zero_max_width_no_truncation(self):
        assert at.truncate("hello world", 0) == "hello world"


# ---------------------------------------------------------------------------
# extract_content_blocks
# ---------------------------------------------------------------------------


class TestExtractContentBlocks:
    def test_string_content(self):
        msg = {"content": "hello"}
        blocks = list(at.extract_content_blocks(msg))
        assert blocks == [("text", "hello")]

    def test_empty_string_content(self):
        msg = {"content": "   "}
        blocks = list(at.extract_content_blocks(msg))
        assert blocks == []

    def test_no_content_key(self):
        msg = {}
        blocks = list(at.extract_content_blocks(msg))
        assert blocks == []

    def test_list_content_with_dicts(self):
        msg = {
            "content": [
                {"type": "text", "text": "hello"},
                {"type": "tool_use", "name": "Read", "input": {}},
            ]
        }
        blocks = list(at.extract_content_blocks(msg))
        assert len(blocks) == 2
        assert blocks[0][0] == "text"
        assert blocks[1][0] == "tool_use"

    def test_list_content_with_strings(self):
        msg = {"content": ["hello", "  ", "world"]}
        blocks = list(at.extract_content_blocks(msg))
        assert blocks == [("text", "hello"), ("text", "world")]

    def test_dict_without_type_yields_unknown(self):
        msg = {"content": [{"data": "something"}]}
        blocks = list(at.extract_content_blocks(msg))
        assert blocks == [("unknown", {"data": "something"})]


# ---------------------------------------------------------------------------
# get_tool_result_text
# ---------------------------------------------------------------------------


class TestGetToolResultText:
    def test_string_content(self):
        block = {"content": "result text"}
        assert at.get_tool_result_text(block) == "result text"

    def test_list_content_with_text_blocks(self):
        block = {
            "content": [
                {"type": "text", "text": "line one"},
                {"type": "text", "text": "line two"},
            ]
        }
        assert at.get_tool_result_text(block) == "line one\nline two"

    def test_list_content_skips_non_text(self):
        block = {
            "content": [
                {"type": "image", "data": "..."},
                {"type": "text", "text": "visible"},
            ]
        }
        assert at.get_tool_result_text(block) == "visible"

    def test_empty_content(self):
        block = {"content": ""}
        assert at.get_tool_result_text(block) == ""

    def test_no_content_key(self):
        block = {}
        assert at.get_tool_result_text(block) == ""

    def test_non_list_non_string_content(self):
        block = {"content": 42}
        assert at.get_tool_result_text(block) == ""


# ---------------------------------------------------------------------------
# detect_file_type
# ---------------------------------------------------------------------------


class TestDetectFileType:
    def test_valid_transcript_returns_none(self):
        assert at.detect_file_type(_fixture("valid_transcript.jsonl")) is None

    def test_otlp_telemetry_detected(self):
        result = at.detect_file_type(_fixture("otlp_telemetry.jsonl"))
        assert result is not None
        assert "OTLP" in result

    def test_non_dict_json_warns(self):
        result = at.detect_file_type(_fixture("non_dict_json.jsonl"))
        assert result is not None
        assert "No recognizable transcript lines" in result

    def test_non_json_lines_warns(self):
        result = at.detect_file_type(_fixture("non_json_lines.jsonl"))
        assert result is not None
        assert "No recognizable transcript lines" in result

    def test_unreadable_file(self, tmp_path):
        bad_path = str(tmp_path / "does_not_exist.jsonl")
        result = at.detect_file_type(bad_path)
        assert result is not None
        assert "Cannot read file" in result

    def test_stdin_returns_none(self):
        # stdin path "-" always returns None (no detection)
        assert at.detect_file_type("-") is None

    def test_blank_line_heavy_file(self):
        # The file has blank lines around a valid transcript line;
        # blank lines are skipped, but the valid line is examined.
        result = at.detect_file_type(_fixture("blank_heavy.jsonl"))
        assert result is None

    def test_empty_file_warns(self, tmp_path):
        empty = tmp_path / "empty.jsonl"
        empty.write_text("")
        result = at.detect_file_type(str(empty))
        assert result is not None
        assert "No recognizable transcript lines" in result

    def test_scopespans_detected_as_otlp(self, tmp_path):
        f = tmp_path / "scope.jsonl"
        f.write_text('{"scopeSpans":[{"scope":{"name":"x"}}]}\n')
        result = at.detect_file_type(str(f))
        assert result is not None
        assert "OTLP" in result


# ---------------------------------------------------------------------------
# _parse_source / parse_lines
# ---------------------------------------------------------------------------


class TestParseLines:
    def test_skips_blank_lines(self):
        results = list(at.parse_lines(_fixture("blank_heavy.jsonl")))
        assert len(results) == 1
        _, obj = results[0]
        assert obj["type"] == "assistant"

    def test_skips_non_json(self):
        results = list(at.parse_lines(_fixture("non_json_lines.jsonl")))
        assert results == []

    def test_skips_non_dict_json(self):
        results = list(at.parse_lines(_fixture("non_dict_json.jsonl")))
        assert results == []

    def test_line_range_filtering(self):
        # valid_transcript.jsonl has lines at indices 0-4
        results = list(at.parse_lines(_fixture("valid_transcript.jsonl"), line_range=(1, 2)))
        line_nums = [i for i, _ in results]
        assert all(1 <= n <= 2 for n in line_nums)

    def test_line_range_open_end(self):
        results = list(at.parse_lines(_fixture("valid_transcript.jsonl"), line_range=(3, None)))
        line_nums = [i for i, _ in results]
        assert all(n >= 3 for n in line_nums)

    def test_non_utf8_resilience(self):
        # Should not crash on non-UTF-8 bytes (opened with errors="replace")
        results = list(at.parse_lines(_fixture("non_utf8.jsonl")))
        # Fixture has exactly 3 parseable JSON dict lines (indices 0, 1, 3);
        # line 2 is raw bad bytes and is skipped.
        assert len(results) == 3

    def test_line_range_stops_early(self):
        # Only first 2 lines (0 and 1)
        results = list(at.parse_lines(_fixture("valid_transcript.jsonl"), line_range=(0, 1)))
        line_nums = [i for i, _ in results]
        assert all(n <= 1 for n in line_nums)

    def test_no_line_range_returns_all(self):
        results = list(at.parse_lines(_fixture("valid_transcript.jsonl")))
        assert len(results) == 5


# ---------------------------------------------------------------------------
# iter_messages
# ---------------------------------------------------------------------------


class TestIterMessages:
    def test_yields_correct_roles(self):
        messages = list(at.iter_messages(_fixture("valid_transcript.jsonl")))
        roles = [role for _, role, _, _ in messages]
        assert roles == ["meta", "user", "assistant", "user", "assistant"]

    def test_agent_setting_yields_meta(self):
        messages = list(at.iter_messages(_fixture("valid_transcript.jsonl")))
        first = messages[0]
        assert first[1] == "meta"  # role
        assert first[2].get("type") == "agent-setting"

    def test_skips_non_transcript_types(self):
        # non_dict_json.jsonl has no transcript type lines
        messages = list(at.iter_messages(_fixture("non_dict_json.jsonl")))
        assert messages == []

    def test_line_range_respected(self):
        # Only lines 0-1 (agent-setting and first user message)
        messages = list(at.iter_messages(_fixture("valid_transcript.jsonl"), line_range=(0, 1)))
        assert len(messages) == 2

    def test_queue_operation_yields_queue_role(self, tmp_path):
        f = tmp_path / "queue.jsonl"
        f.write_text('{"type":"queue-operation","timestamp":"2025-01-01T00:00:00Z"}\n')
        messages = list(at.iter_messages(str(f)))
        assert len(messages) == 1
        assert messages[0][1] == "queue"

    def test_last_prompt_yields_meta(self, tmp_path):
        f = tmp_path / "lp.jsonl"
        f.write_text('{"type":"last-prompt","prompt":"do stuff"}\n')
        messages = list(at.iter_messages(str(f)))
        assert len(messages) == 1
        assert messages[0][1] == "meta"


# ---------------------------------------------------------------------------
# _accumulate_stats
# ---------------------------------------------------------------------------


class TestAccumulateStats:
    def test_basic_stats(self):
        s = at._accumulate_stats(_fixture("valid_transcript.jsonl"))
        assert s["agent"] == "code"
        assert "sess-001" in s["session_ids"]
        assert "claude-sonnet-4-20250514" in s["models"]
        assert s["tokens"]["input"] == 300
        assert s["tokens"]["output"] == 150
        assert s["tokens"]["cache_read"] == 30
        assert s["tokens"]["cache_create"] == 15
        assert s["messages"]["user"] == 2
        assert s["messages"]["assistant"] == 2

    def test_stop_reasons(self):
        s = at._accumulate_stats(_fixture("valid_transcript.jsonl"))
        assert s["stop_reasons"]["tool_use"] == 1
        assert s["stop_reasons"]["end_turn"] == 1

    def test_tool_calls_counted(self):
        s = at._accumulate_stats(_fixture("valid_transcript.jsonl"))
        assert s["tool_calls"]["Read"] == 1

    def test_duration_calculated(self):
        s = at._accumulate_stats(_fixture("valid_transcript.jsonl"))
        # From 00:00:00 to 00:01:00 = 60 seconds
        assert s["duration_seconds"] is not None
        assert s["duration_seconds"] == pytest.approx(60.0, abs=1)

    def test_accepts_pre_iterated_messages(self):
        messages = list(at.iter_messages(_fixture("valid_transcript.jsonl")))
        s = at._accumulate_stats(_fixture("valid_transcript.jsonl"), messages=messages)
        assert s["agent"] == "code"
        assert s["tokens"]["input"] == 300

    def test_no_timestamps_no_duration(self, tmp_path):
        f = tmp_path / "no_ts.jsonl"
        f.write_text(
            '{"type":"assistant","message":{"role":"assistant","content":"hi",'
            '"usage":{"input_tokens":1,"output_tokens":1},'
            '"stop_reason":"end_turn"}}\n'
        )
        s = at._accumulate_stats(str(f))
        assert s["duration_seconds"] is None

    def test_queue_timestamp_contributes_to_duration(self, tmp_path):
        f = tmp_path / "queue_ts.jsonl"
        f.write_text(
            '{"type":"queue-operation","timestamp":"2025-01-01T00:00:00Z"}\n'
            '{"type":"queue-operation","timestamp":"2025-01-01T00:05:00Z"}\n'
        )
        s = at._accumulate_stats(str(f))
        assert s["duration_seconds"] == pytest.approx(300.0, abs=1)

    def test_empty_file(self, tmp_path):
        f = tmp_path / "empty.jsonl"
        f.write_text("")
        s = at._accumulate_stats(str(f))
        assert s["agent"] is None
        assert s["tokens"]["input"] == 0
        assert s["duration_seconds"] is None


# ---------------------------------------------------------------------------
# Error detection (_collect_errors, _check_block_error, _is_error_result)
# ---------------------------------------------------------------------------


class TestErrorDetection:
    def test_is_error_result_with_is_error_flag(self):
        block = {"is_error": True, "content": "oops"}
        assert at._is_error_result(block) is True

    def test_is_error_result_with_tool_use_error_tag(self):
        block = {"content": "<tool_use_error>fail</tool_use_error>"}
        assert at._is_error_result(block) is True

    def test_is_error_result_with_error_tag(self):
        block = {"content": "<error>fail</error>"}
        assert at._is_error_result(block) is True

    def test_is_error_result_normal_content(self):
        block = {"content": "all good"}
        assert at._is_error_result(block) is False

    def test_collect_errors_from_fixture(self):
        errors, mentions = at._collect_errors(_fixture("errors_transcript.jsonl"), max_w=400)
        # Errors: is_error tool_result, tool_use_error tool_result,
        # Exit code 1 tool_result, <error> in user text, <error> in
        # tool_result
        assert len(errors) == 5
        # Mentions: assistant text containing "fatal error" and
        # "api error"
        assert len(mentions) == 2

    def test_result_error_patterns_match(self):
        """_RESULT_ERROR_PATTERNS matches expected error prefixes."""
        for text in [
            "Error: something failed",
            "error: could not find",
            "Exit code 1",
            "FAIL test_foo",
            "fatal: not a git repo",
            "panic: runtime error",
            "Traceback (most recent call last)",
        ]:
            assert at._RESULT_ERROR_PATTERNS.search(text), f"Should match: {text}"

    def test_result_error_patterns_no_false_positive(self):
        """Normal text containing 'error' should not match."""
        text = "The function handles the error gracefully."
        assert not at._RESULT_ERROR_PATTERNS.search(text)

    def test_check_block_error_user_error_tag(self):
        errors, mentions = [], []
        at._check_block_error("user", "text", "<error>bad</error>", 5, 400, errors, mentions)
        assert len(errors) == 1
        assert errors[0][0] == 5

    def test_check_block_error_assistant_keyword(self):
        errors, mentions = [], []
        at._check_block_error(
            "assistant",
            "text",
            "There was a permission denied issue",
            10,
            400,
            errors,
            mentions,
        )
        assert len(mentions) == 1
        assert mentions[0][0] == 10

    def test_check_block_error_tool_result_not_error(self):
        """Normal tool_result should not be flagged."""
        errors, mentions = [], []
        block = {"content": "all good here"}
        at._check_block_error("user", "tool_result", block, 1, 400, errors, mentions)
        assert len(errors) == 0
        assert len(mentions) == 0

    def test_check_block_error_tool_result_with_exit_code(self):
        """tool_result matching _RESULT_ERROR_PATTERNS is an error."""
        errors, mentions = [], []
        block = {"content": "Exit code 1\nfailed"}
        at._check_block_error("user", "tool_result", block, 7, 400, errors, mentions)
        assert len(errors) == 1

    def test_check_block_error_assistant_eacces(self):
        errors, mentions = [], []
        at._check_block_error(
            "assistant",
            "text",
            "Got EACCES on the path",
            3,
            400,
            errors,
            mentions,
        )
        assert len(mentions) == 1


# ---------------------------------------------------------------------------
# Host matching (_host_matches, _match_host, _match_http_entry)
# ---------------------------------------------------------------------------


class TestHostMatching:
    def test_exact_match(self):
        assert at._host_matches("api.github.com", "api.github.com") is True

    def test_parent_domain_match(self):
        assert at._host_matches("api.github.com", "github.com") is True

    def test_no_match(self):
        assert at._host_matches("api.github.com", "gitlab.com") is False

    def test_empty_candidate(self):
        assert at._host_matches("", "github.com") is False

    def test_none_candidate(self):
        assert at._host_matches(None, "github.com") is False

    def test_case_insensitive(self):
        assert at._host_matches("API.GitHub.COM", "api.github.com") is True

    def test_filter_val_not_lowered(self):
        # _host_matches lowercases the candidate but not filter_val;
        # callers are expected to pre-lowercase the filter.
        assert at._host_matches("api.github.com", "GitHub.COM") is False

    def test_match_host_no_filter(self):
        e = {"host": "api.github.com"}
        assert at._match_host(e, None) is True

    def test_match_host_via_host_field(self):
        e = {"host": "api.github.com"}
        assert at._match_host(e, "github.com") is True

    def test_match_host_via_http_url(self):
        e = {"http_url": "https://api.github.com/repos/foo/bar"}
        assert at._match_host(e, "github.com") is True

    def test_match_host_no_match(self):
        e = {"host": "registry.npmjs.org"}
        assert at._match_host(e, "github.com") is False

    def test_match_http_entry_requires_http_method(self):
        e = {"host": "api.github.com"}
        assert at._match_http_entry(e, None, None) is False

    def test_match_http_entry_method_filter(self):
        e = {"http_method": "GET", "host": "api.github.com"}
        assert at._match_http_entry(e, {"POST"}, None) is False
        assert at._match_http_entry(e, {"GET"}, None) is True

    def test_match_http_entry_host_filter(self):
        e = {"http_method": "GET", "host": "api.github.com"}
        assert at._match_http_entry(e, None, "github.com") is True
        assert at._match_http_entry(e, None, "gitlab.com") is False

    def test_match_http_entry_combined_filters(self):
        e = {"http_method": "POST", "host": "api.github.com"}
        assert at._match_http_entry(e, {"POST"}, "github.com") is True
        assert at._match_http_entry(e, {"GET"}, "github.com") is False
        assert at._match_http_entry(e, {"POST"}, "gitlab.com") is False


# ---------------------------------------------------------------------------
# parse_sandbox_log
# ---------------------------------------------------------------------------


class TestParseSandboxLog:
    def test_parses_all_entries(self):
        entries = list(at.parse_sandbox_log(_fixture("sandbox_network.log")))
        assert len(entries) == 5

    def test_entry_fields(self):
        entries = list(at.parse_sandbox_log(_fixture("sandbox_network.log")))
        first = entries[0]
        assert first["ts"] == 1000.0
        assert first["level"] == "info"
        assert first["event"] == "network_activity"
        assert first["host"] == "api.github.com"
        assert first["port"] == 443
        assert first["verdict"] == "ALLOWED"
        assert first["http_method"] == "GET"
        assert "api.github.com" in first["http_url"]
        assert first["policy"] == "github-api"
        assert first["process"] == "/usr/bin/curl"
        assert first["pid"] == 1234

    def test_denied_entry(self):
        entries = list(at.parse_sandbox_log(_fixture("sandbox_network.log")))
        denied = [e for e in entries if e.get("verdict") == "DENIED"]
        assert len(denied) == 1
        assert denied[0]["host"] == "evil.example.com"

    def test_empty_log(self, tmp_path):
        log_file = tmp_path / "empty.log"
        log_file.write_text("")
        entries = list(at.parse_sandbox_log(str(log_file)))
        assert entries == []

    def test_non_ocsf_lines_skipped(self, tmp_path):
        log_file = tmp_path / "other.log"
        log_file.write_text(
            "[1000.0] [sandbox] [info ] [other] something happened\nrandom log line\n"
        )
        entries = list(at.parse_sandbox_log(str(log_file)))
        assert entries == []


# ---------------------------------------------------------------------------
# cmd_summary (end-to-end via capsys)
# ---------------------------------------------------------------------------


class TestCmdSummary:
    def _make_args(self, path, json_output=False, line_range=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.json_output = json_output
        a.line_range = line_range
        a.max_width = 400
        return a

    def test_text_output(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"))
        at.cmd_summary(args)
        out = capsys.readouterr().out
        assert "Agent:" in out
        assert "code" in out
        assert "Tokens:" in out
        assert "300 in" in out

    def test_json_output(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"), json_output=True)
        at.cmd_summary(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        assert data["agent"] == "code"
        assert data["tokens"]["input"] == 300
        assert data["tokens"]["output"] == 150
        assert "Read" in data["tool_calls"]


# ---------------------------------------------------------------------------
# cmd_errors (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdErrors:
    def _make_args(self, path, max_width=400, line_range=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.max_width = max_width
        a.line_range = line_range
        return a

    def test_finds_errors(self, capsys):
        args = self._make_args(_fixture("errors_transcript.jsonl"))
        at.cmd_errors(args)
        out = capsys.readouterr().out
        assert "ERROR:" in out

    def test_finds_mentions(self, capsys):
        args = self._make_args(_fixture("errors_transcript.jsonl"))
        at.cmd_errors(args)
        out = capsys.readouterr().out
        assert "MENTION:" in out

    def test_no_errors_message(self, capsys):
        args = self._make_args(_fixture("blank_heavy.jsonl"))
        at.cmd_errors(args)
        out = capsys.readouterr().out
        assert "No errors found" in out


# ---------------------------------------------------------------------------
# cmd_audit (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdAudit:
    def _make_args(self, path, max_width=400, line_range=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.max_width = max_width
        a.line_range = line_range
        return a

    def test_audit_combines_summary_and_errors(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"))
        at.cmd_audit(args)
        out = capsys.readouterr().out
        # Summary section
        assert "Agent:" in out
        assert "Tokens:" in out
        # Tool table
        assert "Read" in out
        # Error section
        assert "Errors: none" in out

    def test_audit_with_errors(self, capsys):
        args = self._make_args(_fixture("errors_transcript.jsonl"))
        at.cmd_audit(args)
        out = capsys.readouterr().out
        assert "Errors (" in out


# ---------------------------------------------------------------------------
# cmd_tools (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdTools:
    def _make_args(self, path, json_output=False, line_range=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.json_output = json_output
        a.line_range = line_range
        return a

    def test_text_output(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"))
        at.cmd_tools(args)
        out = capsys.readouterr().out
        assert "Read" in out

    def test_json_output(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"), json_output=True)
        at.cmd_tools(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        assert "Read" in data
        assert data["Read"]["count"] == 1

    def test_no_tools_message(self, capsys):
        args = self._make_args(_fixture("blank_heavy.jsonl"))
        at.cmd_tools(args)
        out = capsys.readouterr().out
        assert "No tool calls found" in out


# ---------------------------------------------------------------------------
# cmd_conversation (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdConversation:
    def _make_args(self, path, max_width=400, line_range=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.max_width = max_width
        a.line_range = line_range
        return a

    def test_conversation_output(self, capsys):
        args = self._make_args(_fixture("valid_transcript.jsonl"))
        at.cmd_conversation(args)
        out = capsys.readouterr().out
        assert "USER:" in out
        assert "ASSISTANT:" in out
        assert "TOOL CALL:" in out
        assert "RESULT:" in out


# ---------------------------------------------------------------------------
# cmd_search (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdSearch:
    def _make_args(self, pattern, path, max_width=400, line_range=None):
        class Args:
            pass

        a = Args()
        a.pattern = pattern
        a.file = path
        a.max_width = max_width
        a.line_range = line_range
        return a

    def test_search_finds_match(self, capsys):
        args = self._make_args("bug", _fixture("valid_transcript.jsonl"))
        at.cmd_search(args)
        out = capsys.readouterr().out
        assert "bug" in out.lower()

    def test_search_no_match(self, capsys):
        args = self._make_args("zzzznotfound", _fixture("valid_transcript.jsonl"))
        at.cmd_search(args)
        out = capsys.readouterr().out
        assert "No matches" in out


# ---------------------------------------------------------------------------
# cmd_network (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdNetwork:
    def _make_args(self, path, json_output=False, http=False, method=None, host=None):
        class Args:
            pass

        a = Args()
        a.file = path
        a.json_output = json_output
        a.http = http
        a.method = method
        a.host = host
        return a

    def test_text_summary(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"))
        at.cmd_network(args)
        out = capsys.readouterr().out
        assert "Duration:" in out
        assert "Events:" in out
        assert "Hosts:" in out
        assert "DENIED" in out
        assert "evil.example.com" in out

    def test_http_flag_lists_requests(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"), http=True)
        at.cmd_network(args)
        out = capsys.readouterr().out
        assert "HTTP requests" in out
        assert "GET" in out

    def test_method_filter(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"), method="POST")
        at.cmd_network(args)
        out = capsys.readouterr().out
        assert "HTTP requests" in out
        assert "POST" in out

    def test_host_filter(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"), host="github.com")
        at.cmd_network(args)
        out = capsys.readouterr().out
        assert "HTTP requests" in out

    def test_json_output(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"), json_output=True)
        at.cmd_network(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        assert isinstance(data, list)
        assert len(data) == 5

    def test_json_with_http_filter(self, capsys):
        args = self._make_args(_fixture("sandbox_network.log"), json_output=True, http=True)
        at.cmd_network(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        http_entries = [e for e in data if e.get("http_method")]
        denied_entries = [e for e in data if e.get("verdict") == "DENIED"]
        assert len(http_entries) == 4
        assert len(denied_entries) == 1

    def test_json_with_method_filter(self, capsys):
        args = self._make_args(
            _fixture("sandbox_network.log"),
            json_output=True,
            method="POST",
        )
        at.cmd_network(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        http_entries = [e for e in data if e.get("http_method")]
        for e in http_entries:
            assert e["http_method"] == "POST"

    def test_json_with_host_filter(self, capsys):
        args = self._make_args(
            _fixture("sandbox_network.log"),
            json_output=True,
            host="github.com",
        )
        at.cmd_network(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        # Fixture has 5 entries; only 3 match *.github.com (the npmjs and
        # evil.example.com entries should be excluded).
        assert len(data) == 3
        all_hosts = [e.get("host", "") for e in data]
        for h in all_hosts:
            assert "github.com" in h
        # Verify excluded hosts are truly absent.
        assert not any("npmjs" in h for h in all_hosts)

    def test_empty_log(self, tmp_path, capsys):
        log_file = tmp_path / "empty.log"
        log_file.write_text("")
        args = self._make_args(str(log_file))
        at.cmd_network(args)
        out = capsys.readouterr().out
        assert "No OCSF events found" in out

    def test_denied_entries_in_host_filtered_json(self, capsys):
        """DENIED entries that match the host filter are included."""
        args = self._make_args(
            _fixture("sandbox_network.log"),
            json_output=True,
            host="evil.example.com",
        )
        at.cmd_network(args)
        out = capsys.readouterr().out
        data = json.loads(out)
        denied = [e for e in data if e.get("verdict") == "DENIED"]
        assert len(denied) == 1


# ---------------------------------------------------------------------------
# cmd_network_search (end-to-end)
# ---------------------------------------------------------------------------


class TestCmdNetworkSearch:
    def _make_args(self, pattern, path):
        class Args:
            pass

        a = Args()
        a.pattern = pattern
        a.file = path
        return a

    def test_search_finds_host(self, capsys):
        args = self._make_args("github", _fixture("sandbox_network.log"))
        at.cmd_network_search(args)
        out = capsys.readouterr().out
        assert "github" in out.lower()

    def test_search_no_match(self, capsys):
        args = self._make_args("zzzznotfound", _fixture("sandbox_network.log"))
        at.cmd_network_search(args)
        out = capsys.readouterr().out
        assert "No matches" in out
