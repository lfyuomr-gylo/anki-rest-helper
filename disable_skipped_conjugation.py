#!/usr/bin/env python3

import sys
import json

note = json.load(sys.stdin)
print(f"Loaded note: {note}", file=sys.stderr)

modifications = []
for field, value in note.items():
    if value.strip() == "-":
        modifications.append({"add_tag": f"conjugation_skip:{field}"})
        modifications.append({"set_field": {field: ""}})

json.dump(modifications, sys.stdout)
