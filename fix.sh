#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
FILE="internal/engine/http_executor.go"

echo "=== Fixing HTTP template context ==="

if [[ ! -f "$FILE" ]]; then
  echo "ERROR: $FILE not found"
  exit 1
fi

echo "-> Patching $FILE"

perl -0777 -i -pe '
s@type HTTPExecutor struct \{@type HTTPExecutor struct {@g;

# Insert TemplateContext struct if missing
unless (/type TemplateContext struct/) {
  s@(\n)@${1}type TemplateContext struct {\n\tEvent    string                 `json:"event"`\n\tActionID string                 `json:"actionId,omitempty"`\n\tMetadata map[string]interface{} `json:"metadata"`\n}\n\n@;
}

# Replace template execution context
s@tmpl.Execute\([^,]+,\s*([^)]+)\)@tmpl.Execute($1, TemplateContext{
\tEvent:    string(event),
\tMetadata: map[string]interface{}{
\t\t"name":      obj.GetName(),
\t\t"namespace": obj.GetNamespace(),
\t\t"uid":       string(obj.GetUID()),
\t\t"labels":    obj.GetLabels(),
\t},
})@gs;

' "$FILE"

echo "=== gofmt ==="
gofmt -w "$FILE"

echo "=== DONE ==="
echo
echo "Now run:"
echo "  make generate && make run"

