# Usage

```
nix-shell -p quicktype
cd events
go generate ./...
```

cloud events.json -> removed data_base64


allof/anyof is not properly supported for golang:
https://github.com/glideapps/quicktype/issues/493


TODO:
removed data


No workie worke:
```
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "description": "OpenMeter Events Schema",
  "type": "object",
  "$ref": "cloudevents.json",
  "properties": {
    "data_base64": {
      "description": "The event payload encoded in base64.",
      "type": "string",
      "maxLength": 0
    }
  },
  "if": {
    "properties": {
      "type": {
        "const": "balance.snapshot"
      }
    }
  },
  "then": {
    "properties": {
      "data": {
        "$ref": "payload-entitlements-balance-snapshot-v1.json"
      }
    }
  },
  "required": [
    "id",
    "source",
    "specversion",
    "type",
    "time",
    "data"
  ]
}
```


TODO:
config support?!


latest version works

nix version: quicktype  --src-lang schema *.json --lang go -o events.go --package events

ok
