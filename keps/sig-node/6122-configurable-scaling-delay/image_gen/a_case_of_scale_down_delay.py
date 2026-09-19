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

#!/usr/bin/env python3
"""Generate a_case_of_scale_down_delay.svg for KEP-6122.

Run `make` in this directory to regenerate the SVG and the PNG.

This reproduces the hand-drawn original: the lane order, the container
panels, the colours and the positions are taken from it, so the coordinates
below are deliberately explicit rather than derived.

Two things differ from the original:
  * the parts that belong to KEP-6369 - the two cpuset arrows, the
    /etc/podinfo/assigned_cpuset paths and the files holding the new cpuset -
    are drawn dashed in OPT, and the legend says they are optional. The
    cpuset value inside the file stays red, because red means "the new
    cpuset" here, the same as in the Allocate box;
  * the legend calls the green circles baseline CPUs, not original CPUs.
"""

import os

FONT = "Helvetica, Arial, sans-serif"
FS = 9.5           # labels
FS_HEAD = 10       # lane and panel headers
FS_SMALL = 8.5     # inside the Allocate box and the CPU circles

BLACK = "#000000"
RED = "#ff0000"     # a cpuset value
BLUE = "#0000ff"    # the word "cpuset" in a label
OPT = "#1a6fb5"     # optional, scope of KEP-6369
GREEN_F, GREEN_S = "#d5e8d4", "#82b366"
ORANGE_F, ORANGE_S = "#ffe6cc", "#d79b00"

W, H = 847, 542
BAR_W = 10
LIFE_BOT = 535
DASH = "4 3"

# key: header box, bar bottom
LANES = [("cpu", "CPU Manager", 162, 259, 465),
         ("vol", "Volume Manager", 286, 374, 457),
         ("cd", "Containerd", 436, 523, 457)]
CX = {k: (a + b) / 2 for k, _, a, b, _ in LANES}
BAR_TOP = 68
KUBELET = (145, 391, 15, 51)
Y_LANE = (33, 50)          # the two boxes inside the kubelet frame
Y_CD = (15, 51)            # Containerd, drawn like an outer participant

PANEL_X = (559, 735)
PANELS = [2, 120, 232, 343]
PANEL_H = [62, 105, 105, 106]
PATH = "/etc/podinfo/assigned_cpuset"
CPUSET = "1,2,11"

out = []


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def text(s, x, baseline, colour=BLACK, size=FS, anchor="middle", weight=None):
    w = f' font-weight="{weight}"' if weight else ""
    out.append(f'<text x="{x:g}" y="{baseline:g}" text-anchor="{anchor}" '
               f'font-family="{FONT}" font-size="{size}" fill="{colour}"{w}>'
               f'{esc(s)}</text>')


def parts(segments, x, baseline, size=FS):
    """A label made of differently coloured runs, left-aligned at x."""
    body = "".join(f'<tspan fill="{c}">{esc(s)}</tspan>' for s, c in segments)
    out.append(f'<text xml:space="preserve" x="{x:g}" y="{baseline:g}" '
               f'font-family="{FONT}" font-size="{size}">{body}</text>')


def rect(x0, y0, x1, y1, fill="#ffffff", stroke=BLACK, dash=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    out.append(f'<rect x="{x0:g}" y="{y0:g}" width="{x1 - x0:g}" '
               f'height="{y1 - y0:g}" fill="{fill}" stroke="{stroke}" '
               f'stroke-width="1"{d}/>')


def line(x0, y0, x1, y1, colour=BLACK, dash=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    out.append(f'<line x1="{x0:g}" y1="{y0:g}" x2="{x1:g}" y2="{y1:g}" '
               f'stroke="{colour}" stroke-width="1"{d}/>')


def arrow(y, x0, x1, colour=BLACK, dash=None):
    d = 1 if x1 > x0 else -1
    line(x0, y, x1, y, colour, dash)
    out.append(f'<path d="M {x1:g} {y:g} l {-d * 7:g} -3 l 0 6 z" '
               f'fill="{colour}"/>')


def up_arrow(cx, cy, colour=BLACK):
    """A task running on that CPU: a stem with a head pointing at the circle."""
    line(cx, cy + 23, cx, cy + 14, colour)
    out.append(f'<path d="M {cx:g} {cy + 8:g} l -3.5 6 l 7 0 z" '
               f'fill="{colour}"/>')


def cpu_circle(cx, cy, label, warm):
    f, s = (ORANGE_F, ORANGE_S) if warm else (GREEN_F, GREEN_S)
    out.append(f'<circle cx="{cx:g}" cy="{cy:g}" r="7" fill="{f}" '
               f'stroke="{s}" stroke-width="1"/>')
    text(label, cx, cy + 3, BLACK, FS_SMALL)


def file_shape(x0, y0, x1, y1, stroke, dash=None):
    """A document with its top right corner folded, as in the original."""
    d = f' stroke-dasharray="{dash}"' if dash else ""
    fold = 14
    out.append(f'<path d="M {x0:g} {y0:g} H {x1 - fold:g} L {x1:g} '
               f'{y0 + fold:g} V {y1:g} H {x0:g} Z" fill="#ffffff" '
               f'stroke="{stroke}" stroke-width="1"{d}/>')
    # the folded corner is filled, so that it still reads as a fold when the
    # outline around it is dashed
    out.append(f'<path d="M {x1 - fold:g} {y0:g} L {x1:g} {y0 + fold:g} '
               f'L {x1 - fold:g} {y0 + fold:g} Z" fill="#f2f2f2" '
               f'stroke="{stroke}" stroke-width="1"/>')


# ------------------------------------------------------------------ lanes ----

rect(KUBELET[0], KUBELET[2], KUBELET[1], KUBELET[3], fill="none")
text("kubelet", (KUBELET[0] + KUBELET[1]) / 2, 27, BLACK, FS_HEAD)
for k, label, x0, x1, bar_bot in LANES:
    y0, y1 = Y_CD if k == "cd" else Y_LANE
    rect(x0, y0, x1, y1)
    text(label, CX[k], (y0 + y1) / 2 + 3.5, BLACK, FS_HEAD)
    line(CX[k], y1, CX[k], BAR_TOP, BLACK, dash="3 2")
    rect(CX[k] - BAR_W / 2, BAR_TOP, CX[k] + BAR_W / 2, bar_bot)
    line(CX[k], bar_bot, CX[k], LIFE_BOT, BLACK, dash="3 2")

# ------------------------------------------------------------- the story -----

text("Pod CPU resize 4->3", 171, 92)
arrow(96, 118, CX["cpu"] - BAR_W / 2)

rect(175, 119, 246, 149)
text("Allocate", 210.5, 127, BLACK, FS_SMALL, weight="bold")
text("exclusive CPUs", 210.5, 136, BLACK, FS_SMALL, weight="bold")
text(CPUSET, 210.5, 147, RED, FS_SMALL, weight="bold")

# optional: the volume manager only needs the cpuset in order to expose it
text("Get cpuset status", 272, 163, OPT)
arrow(167, CX["cpu"] + BAR_W / 2, CX["vol"] - BAR_W / 2, OPT, DASH)
text("Write cpuset to the file", 458, 167, OPT)
arrow(172, CX["vol"] + BAR_W / 2, PANEL_X[0] + 42, OPT, DASH)

# the brace covering the grace period
out.append(f'<path d="M 175 130 q -13 0 -13 12 L 162 244 q 0 12 -12 12 '
           f'q 12 0 12 12 L 162 368 q 0 12 13 12" fill="none" '
           f'stroke="{BLACK}" stroke-width="1"/>')
text(">scaleDownGracePeriodSeconds", 80, 254)
text("(5s)", 80, 265)

rect(175, 370, 246, 399)
text("Reconcile Loop", 210.5, 387)
parts([("Call UpdateContainerResources() to update ", BLACK),
       ("cpuset", BLUE)], 254, 383)
arrow(388, 246, CX["cd"] - BAR_W / 2)
parts([("Apply ", BLACK), ("cpuset", BLUE)], 494, 388)
arrow(392, CX["cd"] + BAR_W / 2, PANEL_X[0])

# ------------------------------------------------------- container panels ----

CIRCLES = [(610.5, "1", False), (635.5, "11", False),
           (661, "2", True), (686.5, "12", True)]
px0, px1 = PANEL_X
pcx = (px0 + px1) / 2

for i, (top, h) in enumerate(zip(PANELS, PANEL_H)):
    rect(px0, top, px1, top + h)
    if i == 0:
        text("Container", pcx, top + 15, BLACK, FS_HEAD)
        circles, cy = CIRCLES, top + 36
    else:
        text("Container", pcx, top + 18, BLACK, FS_HEAD)
        text(PATH, pcx, top + 35, OPT)
        file_shape(px0 + 42, top + 39, px0 + 135, top + 65, OPT, "3 2")
        text(CPUSET, px0 + 44, top + 57, RED, anchor="start", weight="bold")
        circles = CIRCLES if i < 3 else CIRCLES[:3]
        cy = top + 79
    # during workload preparation the tasks have already moved off CPU 12,
    # so its circle keeps no arrow even though the CPU is still assigned
    running = 3 if i == 2 else len(circles)
    for n, (cx, label, warm) in enumerate(circles):
        cpu_circle(cx, cy, label, warm)
        if n < running:
            up_arrow(cx, cy)

# the resize request reaching the container, between the first two panels
out.append(f'<path d="M 611 82 L 617 82 L 617 92 L 623 92 L 614 101 '
           f'L 605 92 L 611 92 Z" fill="#ffffff" stroke="{BLACK}" '
           f'stroke-width="1"/>')
text("Pod CPU resize 4->3", 633, 95, BLACK, FS, anchor="start")

text("New cpuset file update", 744, 176, BLACK, FS, anchor="start")
text("Workload preparation", 744, 284, BLACK, FS, anchor="start")
text("New cpuset apply", 744, 401, BLACK, FS, anchor="start")

# ----------------------------------------------------------------- legend ----

out.append(f'<circle cx="580" cy="481" r="7" fill="{GREEN_F}" '
           f'stroke="{GREEN_S}" stroke-width="1"/>')
text("baseline CPUs", 600, 485, BLACK, FS, anchor="start")
arrow(503, 571, 590, OPT, DASH)
text("optional, scope of KEP-6369", 600, 507, OPT, FS, anchor="start")

# ------------------------------------------------------------------ write ----

svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" '
       f'viewBox="0 0 {W} {H}">',
       '<rect width="100%" height="100%" fill="#ffffff"/>'] + out + ['</svg>']
path = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                    "a_case_of_scale_down_delay.svg")
with open(path, "w") as f:
    f.write("\n".join(svg) + "\n")
print(f"wrote a_case_of_scale_down_delay.svg ({W}x{H})")
