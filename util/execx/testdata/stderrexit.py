#!/usr/bin/env python3

import sys

print("test message written to stderr", file=sys.stderr)
sys.exit(1)