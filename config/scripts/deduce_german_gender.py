#!/usr/bin/env python3

"""
Usage:
  ./deduce_german_gender.py 'die Frage'

  {"set_field": {"Gender": "Femininum"}}
"""

import sys
import json

def deduce_gender(word):
  if word.startswith('der'):
    return 'Maskulinum'
  elif word.startswith('die'):
    return 'Femininum'
  elif word.startswith('das'):
    return 'Neutrum'
  else:
    return None

commands = []
if len(sys.argv) >= 2:
  word = sys.argv[1]
  gender = deduce_gender(sys.argv[1])
  if gender is not None:
    commands.append({"set_field": {"Gender": gender}})

json.dump(commands, fp=sys.stdout)