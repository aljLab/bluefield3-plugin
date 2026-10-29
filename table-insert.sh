```bash
#!/usr/bin/env bash
set -euo pipefail

IP="${1:?Usage: $0 <ip>}"

URL="http://${IP}:8080/api/tables"

curl --silent --show-error --fail-with-body \
  --request POST \
  "$URL" \
  --header "Content-Type: application/json" \
  --header "Accept: application/json" \
  --data-binary @- <<'JSON'
{
  "table_entry": {
    "table_id": 39384870,
    "match": [{"field_id":1,"FieldMatchType":{"Exact":{"value":"AA=="}}}],
    "action": {
      "action_id": 19407568,
      "params": []
    }
  }
}
JSON
echo
```