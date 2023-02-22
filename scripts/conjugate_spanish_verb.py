#!/usr/bin/env python3

"""
  Spanish verb conjugation script.

  This script accepts spanish verb and a list of Anki note tags, and produces note modification commands
  that fill out the note with verb conjugation. Field X is not populated if the note has a tag "conjugation_skip:X".

  CLI arguments:
    1. verb in infinitive form
    2. list of tags formatted as JSON array

  NOTE: for this script to work properly, sd-conjugate executable from https://github.com/librehat/sdapi
        must be present in #PATH
"""

from dataclasses import dataclass
import json
import subprocess
import sys

@dataclass
class ConjugationRule:
  note_field: str
  sd_pronoun: str
  sd_paradigm: str
  sd_tense: str = None

  def produce_note_modification(self, sd_conjugation, note_tags):
    skip_tag = f"conjugation_skip:{self.note_field}"
    if skip_tag in note_tags:
      return None

    for conj in sd_conjugation:
      if self._matches(conj):
        return {'set_field_if_not_empty': {self.note_field: conj.get('word', '')}}
    return None

  def _matches(self, sd_conj):
    if self.sd_pronoun is not None and self.sd_pronoun != sd_conj.get('pronoun', ''):
      return False
    if self.sd_paradigm is not None and self.sd_paradigm != sd_conj.get('paradigm', ''):
      return False
    if self.sd_tense is not None and self.sd_tense != sd_conj.get('tense', ''):
      return False
    return True


RULES = [
      # Anki field |
      ConjugationRule(note_field="IndicativePresentYo",       sd_pronoun="yo",               sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentTu",       sd_pronoun="tú",               sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentEl",       sd_pronoun="él/ella/Ud.",      sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentNosotros", sd_pronoun="nosotros",         sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentVosotros", sd_pronoun="vosotros",         sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentEllos",    sd_pronoun="ellos/ellas/Uds.", sd_paradigm="presentIndicative"),

      ConjugationRule(note_field="ImperativeAffirmativeTu",    sd_pronoun="tú",  sd_paradigm="imperative", sd_tense="affirmative"),
      ConjugationRule(note_field="ImperativeAffirmativeUsted", sd_pronoun="Ud.", sd_paradigm="imperative", sd_tense="affirmative"),

      ConjugationRule(note_field="PreteriteYo",       sd_pronoun="yo",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteTu",       sd_pronoun="tú",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteEl",       sd_pronoun="él/ella/Ud.",      sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteNosotros", sd_pronoun="nosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteVosotros", sd_pronoun="vosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteEllos",    sd_pronoun="ellos/ellas/Uds.", sd_paradigm="preteritIndicative"),
]

if __name__ == '__main__':
  verb_infinitive = sys.argv[1]
  note_tags = set(json.loads(sys.argv[2]))

  exec_result = subprocess.run(['sd-conjugate', verb_infinitive], capture_output=True, encoding='utf-8')
  if exec_result.returncode != 0:
    raise ValueError(f"unexpected return code of sd-conjugate: {exec_result.returncode}")
  sd_conjugation = json.loads(exec_result.stdout)

  commands = []
  for rule in RULES:
    command = rule.produce_note_modification(sd_conjugation, note_tags)
    if command is not None:
      commands.append(command)

  json.dump(commands, sys.stdout)
