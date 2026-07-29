import type { ManifestNode } from "virtual:fullsend-docs";

/**
 * Return the names of every top-level directory in the manifest.
 * These are the folder pills shown in the search filter bar.
 */
export function collectTopLevelDirs(nodes: ManifestNode[]): string[] {
  return nodes
    .filter((n): n is Extract<ManifestNode, { type: "dir" }> => n.type === "dir")
    .map((n) => n.name);
}

/**
 * Filter `nodes` so only top-level directories whose name is in
 * `folders` (plus top-level files, which have no folder) are kept.
 * An empty set means "all" — no filtering.
 */
export function filterByFolders(
  nodes: ManifestNode[],
  folders: ReadonlySet<string>,
): ManifestNode[] {
  if (folders.size === 0) return nodes;
  return nodes.filter((n) => {
    if (n.type === "dir") return folders.has(n.name);
    // Top-level files are only shown when no folder filter is active
    // (already handled by the size === 0 early return above).
    return false;
  });
}

export function filterTree(
  nodes: ManifestNode[],
  query: string,
): ManifestNode[] {
  const q = query.trim().toLowerCase();
  if (!q) return nodes;
  const words = q.split(/\s+/).filter(Boolean);
  if (words.length === 0) return nodes;
  return pruneNodes(nodes, words, "");
}

function matchesAllWords(text: string, words: string[]): boolean {
  const lower = text.toLowerCase();
  return words.every((w) => lower.includes(w));
}

function pruneNodes(
  nodes: ManifestNode[],
  words: string[],
  parentPath: string,
): ManifestNode[] {
  const out: ManifestNode[] = [];
  for (const node of nodes) {
    if (node.type === "file") {
      const searchText = `${node.title} ${node.routeKey}`;
      if (matchesAllWords(searchText, words)) out.push(node);
    } else {
      const dirPath = parentPath
        ? `${parentPath}/${node.name}`
        : node.name;
      if (matchesAllWords(dirPath, words)) {
        out.push(node);
      } else {
        const children = pruneNodes(node.children, words, dirPath);
        if (children.length > 0) {
          out.push({ type: "dir", name: node.name, children });
        }
      }
    }
  }
  return out;
}

export type TextSegment = { text: string; highlight: boolean };

export function highlightSegments(
  text: string,
  query: string,
): TextSegment[] {
  const q = query.trim().toLowerCase();
  if (!q) return [{ text, highlight: false }];
  const words = q.split(/\s+/).filter(Boolean);
  if (words.length === 0) return [{ text, highlight: false }];

  const ranges: [number, number][] = [];
  const lower = text.toLowerCase();
  for (const word of words) {
    let start = 0;
    for (;;) {
      const idx = lower.indexOf(word, start);
      if (idx === -1) break;
      ranges.push([idx, idx + word.length]);
      start = idx + 1;
    }
  }

  if (ranges.length === 0) return [{ text, highlight: false }];

  ranges.sort((a, b) => a[0] - b[0]);
  const merged: [number, number][] = [ranges[0]];
  for (let i = 1; i < ranges.length; i++) {
    const last = merged[merged.length - 1];
    if (ranges[i][0] <= last[1]) {
      last[1] = Math.max(last[1], ranges[i][1]);
    } else {
      merged.push([...ranges[i]]);
    }
  }

  const segments: TextSegment[] = [];
  let pos = 0;
  for (const [s, e] of merged) {
    if (pos < s) segments.push({ text: text.slice(pos, s), highlight: false });
    segments.push({ text: text.slice(s, e), highlight: true });
    pos = e;
  }
  if (pos < text.length)
    segments.push({ text: text.slice(pos), highlight: false });

  return segments;
}
