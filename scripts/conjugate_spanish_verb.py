#!/usr/bin/env python3

"""
  Spanish verb conjugation script.

  This script accepts spanish verb and a list of Anki note tags, and produces note modification commands
  that fill out the note with verb conjugation. Field X is not populated if the note has a tag "conjugation_done:X".

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
import random

@dataclass
class ConjugationRule:
  note_field: str
  prob: float # number in range [0; 1]; probability of the conjugation card generation for regular verbs
  sd_pronoun: str
  sd_paradigm: str
  sd_tense: str = None

  def produce_note_modifications(self, sd_conjugation, note_tags):
    if self._done_tag() in note_tags:
      print(f"conjugation is skipped for field {self.note_field} via tag", file=sys.stderr)
      return []

    for conj in sd_conjugation:
      if self._matches(conj):
        modifications = [{'add_tag': self._done_tag()}]

        prob = self.prob
        if conj.get('isIrregular', 'False'):
          # we always generate conjugation card if the conjugation form is irregular
          prob = 1
        if random.random() < prob:
          modifications.append({
            'set_field_if_not_empty': {self.note_field: conj.get('word', '')}
          })
        else:
          print(f"do not generate regular conjugation card for field {self.note_field} (prob={prob})", file=sys.stderr)

        return modifications
    return []

  def _done_tag(self):
    return f"conjugation_done:{self.note_field}"

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
      ConjugationRule(note_field="IndicativePresentYo",        prob=0.05, sd_pronoun="yo",               sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentTu",        prob=0.30, sd_pronoun="tú",               sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentEl",        prob=0.30, sd_pronoun="él/ella/Ud.",      sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentNosotros",  prob=0.10, sd_pronoun="nosotros",         sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentVosotros",  prob=0.10, sd_pronoun="vosotros",         sd_paradigm="presentIndicative"),
      ConjugationRule(note_field="IndicativePresentEllos",     prob=0.15, sd_pronoun="ellos/ellas/Uds.", sd_paradigm="presentIndicative"),

      ConjugationRule(note_field="ImperativeAffirmativeTu",    prob=0.10, sd_pronoun="tú",               sd_paradigm="imperative", sd_tense="affirmative"),
      ConjugationRule(note_field="ImperativeAffirmativeUsted", prob=0.10, sd_pronoun="Ud.",              sd_paradigm="imperative", sd_tense="affirmative"),

      ConjugationRule(note_field="PreteriteYo",                prob=0.30, sd_pronoun="yo",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteTu",                prob=0.10, sd_pronoun="tú",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteEl",                prob=0.30, sd_pronoun="él/ella/Ud.",      sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteNosotros",          prob=0.05, sd_pronoun="nosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteVosotros",          prob=0.05, sd_pronoun="vosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="PreteriteEllos",             prob=0.20, sd_pronoun="ellos/ellas/Uds.", sd_paradigm="preteritIndicative"),

      ConjugationRule(note_field="ImperfectYo",                prob=0.15, sd_pronoun="yo",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="ImperfectTu",                prob=0.20, sd_pronoun="tú",               sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="ImperfectEl",                prob=0.15, sd_pronoun="él/ella/Ud.",      sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="ImperfectNosotros",          prob=0.15, sd_pronoun="nosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="ImperfectVosotros",          prob=0.15, sd_pronoun="vosotros",         sd_paradigm="preteritIndicative"),
      ConjugationRule(note_field="ImperfectEllos",             prob=0.20, sd_pronoun="ellos/ellas/Uds.", sd_paradigm="preteritIndicative"),
]

if __name__ == '__main__':
  random.seed()

  verb_infinitive = sys.argv[1]
  note_tags = set(json.loads(sys.argv[2]))

  exec_result = subprocess.run(['sd-conjugate', verb_infinitive], capture_output=True, encoding='utf-8')
  if exec_result.returncode != 0:
    raise ValueError(f"unexpected return code of sd-conjugate: {exec_result.returncode}")
  sd_conjugation = json.loads(exec_result.stdout)

  commands = []
  for rule in RULES:
    rule_commands = rule.produce_note_modifications(sd_conjugation, note_tags)
    if len(rule_commands) > 0:
      commands.extend(rule_commands)

  json.dump(commands, sys.stdout, indent=2)
