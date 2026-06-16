#!/bin/bash
# install-lsp.sh — Set up LSP language servers in the fullsend sandbox.
# Runs as pre_script before the agent starts.
# Expects gopls binary at /tmp/workspace/lsp-bin/gopls (copied via host_files).

set -e

# --- Go: gopls ---
if [ -f /tmp/workspace/lsp-bin/gopls ]; then
  cp /tmp/workspace/lsp-bin/gopls /usr/local/bin/gopls 2>/dev/null || \
    cp /tmp/workspace/lsp-bin/gopls /tmp/gopls
  chmod +x /usr/local/bin/gopls 2>/dev/null || chmod +x /tmp/gopls
  GOPLS_PATH=$(which gopls 2>/dev/null || echo /tmp/gopls)
  echo "gopls installed at: $GOPLS_PATH"
else
  echo "WARN: gopls binary not found at /tmp/workspace/lsp-bin/gopls"
  GOPLS_PATH=""
fi

# --- Python: pyright ---
if command -v npm &>/dev/null; then
  npm install -g pyright 2>/dev/null && echo "pyright installed" || echo "WARN: pyright install failed"
  PYRIGHT_PATH=$(which pyright-langserver 2>/dev/null || echo "")
else
  echo "WARN: npm not found, skipping pyright"
  PYRIGHT_PATH=""
fi

# --- Configure Claude Code LSP plugins ---
CLAUDE_PLUGINS_DIR="${CLAUDE_CONFIG_DIR:-/tmp/claude-config}/plugins"

if [ -n "$GOPLS_PATH" ]; then
  mkdir -p "$CLAUDE_PLUGINS_DIR/gopls-lsp"
  cat > "$CLAUDE_PLUGINS_DIR/gopls-lsp/.lsp.json" << LSPEOF
{
  "go": {
    "command": "$GOPLS_PATH",
    "args": ["serve"],
    "extensionToLanguage": { ".go": "go" }
  }
}
LSPEOF
  echo "gopls LSP plugin configured at $CLAUDE_PLUGINS_DIR/gopls-lsp/.lsp.json"
fi

if [ -n "$PYRIGHT_PATH" ]; then
  mkdir -p "$CLAUDE_PLUGINS_DIR/pyright-lsp"
  cat > "$CLAUDE_PLUGINS_DIR/pyright-lsp/.lsp.json" << LSPEOF
{
  "python": {
    "command": "pyright-langserver",
    "args": ["--stdio"],
    "extensionToLanguage": { ".py": "python" }
  }
}
LSPEOF
  echo "pyright LSP plugin configured at $CLAUDE_PLUGINS_DIR/pyright-lsp/.lsp.json"
fi

echo "LSP setup complete."
