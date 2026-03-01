# Standalone CLI Usage

## Compare two files

```bash
differ before.json after.json
```

## Inline JSON

```bash
differ -left-input '{"name":"Alice","age":30}' \
       -right-input '{"name":"Bob","age":30,"email":"bob@x.com"}' \
       -pretty
```

Output:
```json
{
  "equal": false,
  "stats": {
    "added": 1,
    "removed": 0,
    "replaced": 1,
    "total": 2
  },
  "changes": [
    {
      "path": "$.email",
      "type": "added",
      "value": "bob@x.com"
    },
    {
      "path": "$.name",
      "type": "replaced",
      "from": "Alice",
      "to": "Bob"
    }
  ]
}
```

## Pipe from stdin

```bash
curl -s https://api.example.com/config | differ -right expected.json
```

## RFC 6902 JSON Patch

```bash
differ before.json after.json -format patch
```

```json
[{"op":"replace","path":"/name","value":"Bob"},{"op":"add","path":"/email","value":"bob@x.com"}]
```

## Stats only

```bash
differ before.json after.json -format stat
```

```json
{"equal":false,"stats":{"added":1,"removed":0,"replaced":1,"total":2}}
```

## Filter specific paths

```bash
# Only diff the users subtree
differ before.json after.json -only '$.users'

# Ignore metadata and timestamps
differ before.json after.json -ignore '$.metadata,$.timestamps'
```

## Exit codes in scripts

```bash
differ expected.json actual.json -format stat
case $? in
  0) echo "No drift detected" ;;
  1) echo "Configuration drift!" ;;
  2) echo "Error comparing files" ;;
esac
```

## Write output to file

```bash
differ before.json after.json -output diff.json -pretty
```
