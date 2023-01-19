#!/usr/bin/env python3

"""
  Anki Image Resizer.

  This script accepts two arguments:
    1. Name of an anki field that contains a single image.
    2. Value of that field
    3. Height attribute to set for the image.
    4. Tag to add to the note.

  It extracts the image specified in the field, and prints a field modification command to stdout
  that sets the field to the same field but with the specified height.

  NOTE: any contents except the image will be lost, any image attributes will be lost.
"""

from html.parser import HTMLParser
from typing import List, Tuple
import argparse
import json
import sys


class ImgExtractor(HTMLParser):
  """
    ImgExtractor extracts from given HTML image tags like <img src="oido.jpg">
    and stores sources of images into self.image_sources.
  """

  def __init__(self):
    HTMLParser.__init__(self)
    self.image_sources = []

  def handle_starttag(self, tag: str, attrs: List[Tuple[str, str]]):
    if tag != "img":
      return
    srcs = [val for key, val in attrs if key == 'src']
    if len(srcs) != 1:
      print(f"Unexpected number of 'src' attributes in 'img' tag: {len(srcs)}", f=sys.stderr)
      return
    self.image_sources.append(srcs[0])


def extract_image_sources(html_text: str):
  extractor = ImgExtractor()
  extractor.feed(img_field)
  return extractor.image_sources


def escape_xml_attr(value: str):
  for char in value:
    if char in {'"', "'", "\\"}:
      # I'm not planning to use special characters in file names, so there is no need to escape them.
      raise ValueError(f"Unsupported character in XML attribute: {char}")
  return value

field_name = sys.argv[1]
img_field = sys.argv[2]
height = escape_xml_attr(sys.argv[3])
tag = sys.argv[4]

image_sources = extract_image_sources(img_field)
if len(image_sources) != 1:
  raise ValueError(f"unexecpeted number of images found in the field: {len(image_sources)}")
img_src = escape_xml_attr(image_sources[0])
new_img = f'<img src="{img_src}" height="{height}">'

commands = [
  {"set_field": {field_name: new_img}},
  {"add_tag": tag},
]
json.dump(commands, fp=sys.stdout)
