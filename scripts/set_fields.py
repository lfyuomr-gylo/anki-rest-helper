#!/usr/bin/env python3

import sys
import json

commands = []
for i in range(1, len(sys.argv), 2):
    if i+1 < len(sys.argv):
        commands.append({"set_field": {sys.argv[i]: sys.argv[i+1]}})
json.dump(commands, fp=sys.stdout)