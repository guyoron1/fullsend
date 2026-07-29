import { describe, expect, it } from "vitest";
import {
  filterTree,
  highlightSegments,
  collectTopLevelDirs,
  filterByFolders,
} from "./filterTree";
import type { ManifestNode } from "virtual:fullsend-docs";

const tree: ManifestNode[] = [
  {
    type: "dir",
    name: "guides",
    children: [
      {
        type: "dir",
        name: "getting-started",
        children: [
          {
            type: "file",
            name: "installation",
            routeKey: "guides/getting-started/installation",
            title: "Installation Guide",
          },
          {
            type: "file",
            name: "config",
            routeKey: "guides/getting-started/config",
            title: "Configuration",
          },
        ],
      },
      {
        type: "file",
        name: "quickstart",
        routeKey: "guides/quickstart",
        title: "Quickstart",
      },
    ],
  },
  {
    type: "file",
    name: "readme",
    routeKey: "readme",
    title: "README",
  },
];

describe("filterTree", () => {
  it("returns full tree when query is empty", () => {
    expect(filterTree(tree, "")).toBe(tree);
    expect(filterTree(tree, "  ")).toBe(tree);
  });

  it("matches file by title", () => {
    const result = filterTree(tree, "quickstart");
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          {
            type: "file",
            name: "quickstart",
            routeKey: "guides/quickstart",
            title: "Quickstart",
          },
        ],
      },
    ]);
  });

  it("keeps ancestor dirs for nested match", () => {
    const result = filterTree(tree, "installation");
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          {
            type: "dir",
            name: "getting-started",
            children: [
              {
                type: "file",
                name: "installation",
                routeKey: "guides/getting-started/installation",
                title: "Installation Guide",
              },
            ],
          },
        ],
      },
    ]);
  });

  it("prunes branches with no matches", () => {
    const result = filterTree(tree, "zzz");
    expect(result).toEqual([]);
  });

  it("matches case-insensitively", () => {
    const result = filterTree(tree, "README");
    expect(result).toEqual([
      { type: "file", name: "readme", routeKey: "readme", title: "README" },
    ]);
  });

  it("matches dir name and keeps full subtree", () => {
    const result = filterTree(tree, "getting-started");
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          {
            type: "dir",
            name: "getting-started",
            children: [
              {
                type: "file",
                name: "installation",
                routeKey: "guides/getting-started/installation",
                title: "Installation Guide",
              },
              {
                type: "file",
                name: "config",
                routeKey: "guides/getting-started/config",
                title: "Configuration",
              },
            ],
          },
        ],
      },
    ]);
  });

  it("matches multiple words separately (fuzzy)", () => {
    const result = filterTree(tree, "install guide");
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          {
            type: "dir",
            name: "getting-started",
            children: [
              {
                type: "file",
                name: "installation",
                routeKey: "guides/getting-started/installation",
                title: "Installation Guide",
              },
            ],
          },
        ],
      },
    ]);
  });

  it("matches against file routeKey (path)", () => {
    const result = filterTree(tree, "started config");
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          {
            type: "dir",
            name: "getting-started",
            children: [
              {
                type: "file",
                name: "config",
                routeKey: "guides/getting-started/config",
                title: "Configuration",
              },
            ],
          },
        ],
      },
    ]);
  });

  it("multi-word query with no combined match returns empty", () => {
    const result = filterTree(tree, "started readme");
    expect(result).toEqual([]);
  });
});

describe("highlightSegments", () => {
  it("returns full text when query is empty", () => {
    expect(highlightSegments("Hello", "")).toEqual([
      { text: "Hello", highlight: false },
    ]);
  });

  it("highlights single word match", () => {
    expect(highlightSegments("Installation Guide", "install")).toEqual([
      { text: "Install", highlight: true },
      { text: "ation Guide", highlight: false },
    ]);
  });

  it("highlights multiple words independently", () => {
    const result = highlightSegments("Installation Guide", "guide inst");
    expect(result).toEqual([
      { text: "Inst", highlight: true },
      { text: "allation ", highlight: false },
      { text: "Guide", highlight: true },
    ]);
  });

  it("merges overlapping highlights", () => {
    const result = highlightSegments("abcdef", "abc bcd");
    expect(result).toEqual([
      { text: "abcd", highlight: true },
      { text: "ef", highlight: false },
    ]);
  });

  it("returns unhighlighted text when no match", () => {
    expect(highlightSegments("Hello", "xyz")).toEqual([
      { text: "Hello", highlight: false },
    ]);
  });

  it("is case-insensitive", () => {
    expect(highlightSegments("README", "read")).toEqual([
      { text: "READ", highlight: true },
      { text: "ME", highlight: false },
    ]);
  });
});

describe("collectTopLevelDirs", () => {
  it("returns names of top-level dir nodes only", () => {
    expect(collectTopLevelDirs(tree)).toEqual(["guides"]);
  });

  it("returns empty array when no dirs exist", () => {
    const flat: ManifestNode[] = [
      { type: "file", name: "readme", routeKey: "readme", title: "README" },
    ];
    expect(collectTopLevelDirs(flat)).toEqual([]);
  });

  it("returns multiple dirs in order", () => {
    const multi: ManifestNode[] = [
      { type: "dir", name: "ADRs", children: [] },
      { type: "dir", name: "guides", children: [] },
      { type: "file", name: "vision", routeKey: "vision", title: "Vision" },
      { type: "dir", name: "plans", children: [] },
    ];
    expect(collectTopLevelDirs(multi)).toEqual(["ADRs", "guides", "plans"]);
  });
});

describe("filterByFolders", () => {
  it("returns full tree when folder set is empty", () => {
    expect(filterByFolders(tree, new Set())).toBe(tree);
  });

  it("keeps only selected top-level dirs", () => {
    const multi: ManifestNode[] = [
      {
        type: "dir",
        name: "ADRs",
        children: [
          { type: "file", name: "001", routeKey: "ADRs/001", title: "ADR 1" },
        ],
      },
      {
        type: "dir",
        name: "guides",
        children: [
          { type: "file", name: "quickstart", routeKey: "guides/quickstart", title: "Quickstart" },
        ],
      },
      { type: "file", name: "readme", routeKey: "readme", title: "README" },
    ];
    const result = filterByFolders(multi, new Set(["guides"]));
    expect(result).toEqual([
      {
        type: "dir",
        name: "guides",
        children: [
          { type: "file", name: "quickstart", routeKey: "guides/quickstart", title: "Quickstart" },
        ],
      },
    ]);
  });

  it("excludes top-level files when a folder is selected", () => {
    const result = filterByFolders(tree, new Set(["guides"]));
    // The top-level "readme" file should not be in results
    const fileNames = result.map((n) => n.name);
    expect(fileNames).not.toContain("readme");
  });

  it("supports multiple selected folders", () => {
    const multi: ManifestNode[] = [
      { type: "dir", name: "ADRs", children: [] },
      { type: "dir", name: "guides", children: [] },
      { type: "dir", name: "plans", children: [] },
    ];
    const result = filterByFolders(multi, new Set(["ADRs", "plans"]));
    expect(result.map((n) => n.name)).toEqual(["ADRs", "plans"]);
  });
});
