import { describe, expect, it } from "vitest";
import {
  filePathToRouteKey,
  listDocMarkdownFiles,
  pathnameToRouteKey,
  routeKeyToUrl,
} from "./paths";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

describe("paths", () => {
  it("filePathToRouteKey strips docs/ and .md", () => {
    expect(filePathToRouteKey("docs/guides/getting-started/installation.md")).toBe(
      "guides/getting-started/installation",
    );
    expect(filePathToRouteKey("docs/README.md")).toBe("README");
  });

  it("routeKeyToUrl", () => {
    expect(routeKeyToUrl("guides/getting-started/installation")).toBe(
      "/docs/guides/getting-started/installation",
    );
  });

  it("pathnameToRouteKey", () => {
    expect(pathnameToRouteKey("/docs/guides/getting-started/installation")).toBe(
      "guides/getting-started/installation",
    );
    expect(pathnameToRouteKey("/docs/")).toBe("");
  });

  it("listDocMarkdownFiles respects filter", () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "fullsend-docs-"));
    fs.mkdirSync(path.join(tmp, "docs", "a"), { recursive: true });
    fs.writeFileSync(path.join(tmp, "docs", "a", "x.md"), "# x\n");
    fs.writeFileSync(path.join(tmp, "docs", "skip.md"), "# s\n");

    const all = listDocMarkdownFiles(tmp);
    expect(all).toContain("docs/a/x.md");
    expect(all).toContain("docs/skip.md");

    const filtered = listDocMarkdownFiles(tmp, (p) => p !== "docs/skip.md");
    expect(filtered).toContain("docs/a/x.md");
    expect(filtered).not.toContain("docs/skip.md");
  });

  it("listDocMarkdownFiles excludes non-content files", () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "fullsend-docs-"));
    fs.mkdirSync(path.join(tmp, "docs", "guides"), { recursive: true });
    fs.writeFileSync(path.join(tmp, "docs", "guides", "intro.md"), "# Intro\n");
    fs.writeFileSync(path.join(tmp, "docs", "AGENTS.md"), "# Agents\n");
    fs.writeFileSync(path.join(tmp, "docs", "CLAUDE.md"), "# Claude\n");
    fs.writeFileSync(path.join(tmp, "docs", "CONTRIBUTING.md"), "# Contrib\n");

    const result = listDocMarkdownFiles(tmp);
    expect(result).toContain("docs/guides/intro.md");
    expect(result).not.toContain("docs/AGENTS.md");
    expect(result).not.toContain("docs/CLAUDE.md");
    expect(result).not.toContain("docs/CONTRIBUTING.md");
  });

  it("listDocMarkdownFiles excludes template directories", () => {
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "fullsend-docs-"));
    fs.mkdirSync(path.join(tmp, "docs", "experiments", "0000-experiment-template"), {
      recursive: true,
    });
    fs.mkdirSync(path.join(tmp, "docs", "experiments", "0001-real"), {
      recursive: true,
    });
    fs.writeFileSync(
      path.join(tmp, "docs", "experiments", "0000-experiment-template", "README.md"),
      "# Template\n",
    );
    fs.writeFileSync(
      path.join(tmp, "docs", "experiments", "0001-real", "README.md"),
      "# Real\n",
    );

    const result = listDocMarkdownFiles(tmp);
    expect(result).toContain("docs/experiments/0001-real/README.md");
    expect(result).not.toContain(
      "docs/experiments/0000-experiment-template/README.md",
    );
  });
});
