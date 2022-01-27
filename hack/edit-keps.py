#!/usr/bin/env python3

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Edit KEPs en-masse by round-tripping them through ruamel.yaml

This is not intended for general usage, because:
- many keps have different formatting, and we're not at a point where
  we can enforce formatting standards, so this is almost guaranteed
  to introduce formatting change noise
- the idea is to manually edit this file with the specific edit to be
  done, rather that developing a general purpose language to do this
"""

import argparse
import glob

from os import path

import ruamel.yaml

# Files that will be ignored
EXCLUDED_FILES = []
# A hilariously large line length to ensure we never line-wrap
MAX_WIDTH = 2000000000

def setup_yaml():
    # Setup the ruamel.yaml parser
    yaml = ruamel.yaml.YAML(typ='rt')
    yaml.preserve_quotes = True
    # This is what's used in the template, currently ~36 KEPs have drifted
    yaml.indent(mapping=2, sequence=4, offset=2)
    yaml.width = MAX_WIDTH
    return yaml

def edit_kep(yaml, file_name, force_rewrite=False):
    with open(file_name, "r") as fp:
        kep = yaml.load(fp)

    rewrite = force_rewrite

    stage = kep.get("stage", "unknown")
    status = kep.get("status", "unknown")
    latest_milestone = kep.get("latest-milestone", "unknown")
    last_updated = kep.get("last-updated", "unknown")
    milestone = kep.get("milestone", {})

    if status == "implemented":
        if latest_milestone == "unknown":
            print(f'status: {status} stage: {stage} last-updated: {last_updated} file: {file_name}')
            kep["latest-milestone"] = "0.0"
            rewrite = True
        if stage == "unknown":
            if latest_milestone == "unknown":
                kep["stage"] = "stable"
            else:
                kep["stage"] = [s for s,v in milestone.items() if v == latest_milestone][0]
            rewrite = True

    # Dump KEP to file_name
    if rewrite:
        print(f'  writing {file_name}')
        with open(file_name, "w") as fp:
            yaml.dump(kep, fp)
            fp.truncate()

def main(keps_dir, force_rewrite):
    yaml = setup_yaml()
    for f in glob.glob(f'{keps_dir}/**/kep.yaml', recursive=True):
        if path.basename(f) not in EXCLUDED_FILES:
            try:
                print(f'processing file: {f}')
                edit_kep(yaml, f, force_rewrite)
            except Exception as e:  # pylint: disable=broad-except
                print(f'ERROR: could not edit {f}: {e}')

if __name__ == '__main__':
    PARSER = argparse.ArgumentParser(
        description='Does things to KEPs')
    PARSER.add_argument(
        '--keps-dir',
        default='../keps',
        help='Path to KEPs directoryProw Job Directory')
    PARSER.add_argument(
        '--force',
        default=False,
        help='Force rewrite of all KEPs')
    ARGS = PARSER.parse_args()

    main(ARGS.keps_dir, ARGS.force)

