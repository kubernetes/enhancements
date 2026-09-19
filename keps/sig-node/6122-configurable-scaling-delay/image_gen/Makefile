# Copyright The Kubernetes Authors.
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


IMAGES += scale_delay_overview.png
IMAGES += flow_as_is.png flow_to_be.png
IMAGES += a_case_of_scale_down_delay.png
SOURCES = $(IMAGES:.png=.svg)

# The sources live here, but the rendered images belong next to the README
# that embeds them.
OUT = ..
PNGS = $(addprefix $(OUT)/,$(IMAGES))

# Rendered at twice the SVG size so the text stays sharp in the rendered KEP.
SCALE = 2

.PHONY: all clean
all: $(SOURCES) $(PNGS)
clean:
	rm -f $(PNGS) $(SOURCES)

# the generated SVG is checked in, so make must not treat it as intermediate
.PRECIOUS: $(SOURCES)

# The SVG is generated, so that the layout stays consistent and no element
# can end up drawn across a lifeline it does not belong to. Needs python3.
%.svg: %.py
	python3 $<

# one script draws both flow diagrams, because they share most of their layout
flow_as_is.svg flow_to_be.svg: flow_diagrams.py
	python3 $<

# Needs one of rsvg-convert, inkscape or chromium.
$(OUT)/%.png: %.svg render-svg.sh
	./render-svg.sh $< $@ $(SCALE)
