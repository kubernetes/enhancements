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
"""Generate flow_as_is.svg and flow_to_be.svg for KEP-6122.

Run `make` in this directory to regenerate the SVGs and the PNGs.

These two diagrams reproduce the hand-drawn originals: the lane order, the
box and arrow positions, the colours and the strikethroughs are taken from
them, so the coordinates below are deliberately explicit rather than derived.

Two things differ from the originals:
  * the Downward API lane is gone, together with the "Desired CPUSet" arrow
    and the "Expose desired CPUSet" box - that flow now belongs to KEP-6369.
    Nothing else moves: only the kubelet frame is shortened, so that it does
    not enclose the column the lane used to occupy;
  * both diagrams use one common set of row positions for the part they
    share, which makes them directly comparable side by side.
"""

import os

FONT = "Helvetica, Arial, sans-serif"
FS = 10.5          # all labels
BLACK = "#000000"
RED = "#ff0000"
BAR_W = 10         # width of an activation bar
HEAD = 8           # arrowhead length

W, H = 1180, 530
LIFE_BOT = 529
BAR_BOT = 513

# key: label, header box, bar top
LANES = [
    ("api", "API Server", 83, 171, "outer", 103, 157),
    ("hpu", "HandlePodUpdates", 225, 323, "inner", 103, BAR_BOT),
    ("sync", "SyncPod", 389, 493, "inner", 85, BAR_BOT),
    ("am", "Allocation Manager", 572, 680, "inner", 85, BAR_BOT),
    ("cpu", "CPU Manager", 753, 842, "inner", 103, BAR_BOT),
    ("rt", "Runtime", 1089, 1178, "outer", 103, BAR_BOT),
]
CX = {k: (a + b) / 2 for k, _, a, b, _, _, _ in LANES}

KUBELET = (207, 878)       # the frame around the kubelet lane headers
Y_OUTER = (5, 67)          # API Server and Runtime header boxes
Y_INNER = (32, 67)         # headers inside the kubelet frame

out = []


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def tw(s):
    """Estimate the rendered width of a label, for the strikethrough lines.

    Calibrated against the originals: the two struck lines measure 150 and
    172 pixels there.
    """
    w = 0.0
    for ch in s:
        if ch in "iljItf.,:;'|!()[]{}":
            w += 0.31
        elif ch in "mwMW@":
            w += 0.90
        elif ch == " ":
            w += 0.30
        elif ch.isupper() or ch.isdigit():
            w += 0.66
        else:
            w += 0.55
    return w * FS * 0.96


def text(s, cx, baseline, colour=BLACK, anchor="middle"):
    out.append(f'<text x="{cx:g}" y="{baseline:g}" text-anchor="{anchor}" '
               f'font-family="{FONT}" font-size="{FS}" fill="{colour}">'
               f'{esc(s)}</text>')


def rect(x0, y0, x1, y1, fill="#ffffff", stroke=BLACK, width=1):
    out.append(f'<rect x="{x0:g}" y="{y0:g}" width="{x1 - x0:g}" '
               f'height="{y1 - y0:g}" fill="{fill}" stroke="{stroke}" '
               f'stroke-width="{width}"/>')


def line(x0, y0, x1, y1, colour=BLACK, dash=None):
    d = f' stroke-dasharray="{dash}"' if dash else ""
    out.append(f'<line x1="{x0:g}" y1="{y0:g}" x2="{x1:g}" y2="{y1:g}" '
               f'stroke="{colour}" stroke-width="1"{d}/>')


def arrow(y, x0, x1, label=None, label_cx=None, label_baseline=None,
          colour=BLACK, label_colour=None):
    """Horizontal arrow from x0 to x1, filled head at x1."""
    d = 1 if x1 > x0 else -1
    line(x0, y, x1, y, colour)
    out.append(f'<path d="M {x1:g} {y:g} l {-d * HEAD:g} -3.5 l 0 7 z" '
               f'fill="{colour}"/>')
    if label:
        text(label, label_cx, label_baseline, label_colour or colour)


def box(x0, y0, x1, lines, text_colour=BLACK, strike=False, height=None):
    """White box with centred lines of text; optional strikethrough."""
    h = height if height is not None else (35 if len(lines) > 1 else 26)
    y1 = y0 + h
    rect(x0, y0, x1, y1)
    cx = (x0 + x1) / 2
    if len(lines) == 1:
        bases = [y0 + h / 2 + 3.5]
    else:
        bases = [y0 + 16, y0 + 29]
    for s, b in zip(lines, bases):
        text(s, cx, b, text_colour)
        if strike:
            # the originals strike the text through in black, not in red
            w = tw(s)
            line(cx - w / 2, b - 4, cx + w / 2, b - 4, BLACK)
    return y1


def brace(x, y0, y1, lines, cx):
    """A right curly brace, spine at x, opening towards the lane on its left."""
    mid = (y0 + y1) / 2
    out.append(
        f'<path d="M {x - 10:g} {y0:g} q 10 0 10 10 L {x:g} {mid - 10:g} '
        f'q 0 10 5 10 q -5 0 -5 10 L {x:g} {y1 - 10:g} '
        f'q 0 10 -10 10" fill="none" stroke="{BLACK}" stroke-width="1"/>')
    if len(lines) == 1:
        text(lines[0], cx, mid + 8, BLACK)
    else:
        text(lines[0], cx, mid - 1, BLACK)
        text(lines[1], cx, mid + 13, RED)


def header():
    rect(*KUBELET[:1], Y_OUTER[0], KUBELET[1], Y_OUTER[1], fill="none")
    text("kubelet", (KUBELET[0] + KUBELET[1]) / 2, 21)
    for k, label, x0, x1, where, _, _ in LANES:
        y0, y1 = Y_OUTER if where == "outer" else Y_INNER
        rect(x0, y0, x1, y1)
        text(label, CX[k], (y0 + y1) / 2 + 3.5)
    # lifelines first, then the activation bars on top of them
    for k, _, _, _, where, bar0, bar1 in LANES:
        y0, y1 = Y_OUTER if where == "outer" else Y_INNER
        line(CX[k], y1, CX[k], LIFE_BOT, BLACK, dash="8 3")
    for k, _, _, _, _, bar0, bar1 in LANES:
        rect(CX[k] - BAR_W / 2, bar0, CX[k] + BAR_W / 2, bar1)


def actor():
    out.append(f'<circle cx="17" cy="85" r="4" fill="none" stroke="{BLACK}" '
               f'stroke-width="1"/>')
    out.append(f'<path d="M 17 89 v 10 M 10 93 h 14 M 17 99 l -6 9 '
               f'M 17 99 l 6 9" fill="none" stroke="{BLACK}" '
               f'stroke-width="1"/>')
    text("Actor", 16, 125)
    out.append(f'<circle cx="25" cy="94" r="3.5" fill="{BLACK}"/>')
    text("Scale down", 73, 77)
    text("patch command", 73, 90)
    arrow(94, 25, CX["api"] - BAR_W / 2)


PENDING = ["1. Pod Resize Completed event",
           "2. update container actual resources"]


def shared(struck_first_box, delay_lines, delay_cx):
    """Everything both diagrams have in common."""
    header()
    actor()
    arrow(116, CX["api"] + BAR_W / 2, CX["hpu"] - BAR_W / 2,
          "Scale down", 198, 111)
    arrow(121, CX["hpu"] + BAR_W / 2, CX["am"] - BAR_W / 2,
          "Scale down", 536, 116)
    arrow(127, CX["am"] + BAR_W / 2, CX["cpu"] - BAR_W / 2,
          "Scale down", 752, 123)
    box(717, 139, 878, ["Allocate new CPUSet"])
    arrow(192, CX["cpu"] - BAR_W / 2, CX["sync"] + BAR_W / 2)
    box(341, 214, 537, ["CPU Request actuation"])
    arrow(228, 537, CX["rt"] - BAR_W / 2, "CPU Request apply", 1028, 216)
    brace(833, 174, 325, delay_lines, delay_cx)
    arrow(272, CX["sync"] - BAR_W / 2, CX["hpu"] + BAR_W / 2,
          "SyncLoop (PLEG)", 359, 268)
    arrow(288, CX["hpu"] + BAR_W / 2, CX["sync"] - BAR_W / 2)
    box(341, 304, 537, PENDING,
        RED if struck_first_box else BLACK, strike=struck_first_box)
    box(717, 335, 878, ["New CPUSet actuation"])
    arrow(349, 878, CX["rt"] - BAR_W / 2, "CPUSet apply", 1004, 338)


def write(name):
    svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" '
           f'viewBox="0 0 {W} {H}">',
           '<rect width="100%" height="100%" fill="#ffffff"/>'] + out + ['</svg>']
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), name)
    with open(path, "w") as f:
        f.write("\n".join(svg) + "\n")
    print(f"wrote {name} ({W}x{H})")


# --------------------------------------------------------------- as is -------

out = []
shared(struck_first_box=False, delay_lines=["Delay"], delay_cx=888)
write("flow_as_is.svg")

# --------------------------------------------------------------- to be -------

out = []
# shifted a little further right than in the original, where the opening
# bracket of the second line ran into the brace
shared(struck_first_box=True,
       delay_lines=["Delay", "( > scaleDownGracePeriodSeconds)"],
       delay_cx=925)
# the resize only completes after the delayed cpuset has been applied
arrow(387, CX["cpu"] - BAR_W / 2, CX["hpu"] + BAR_W / 2,
      "SyncLoop (PLEG)", 536, 382, label_colour=RED)
arrow(409, CX["hpu"] + BAR_W / 2, CX["sync"] - BAR_W / 2)
box(341, 426, 537, PENDING, RED)
write("flow_to_be.svg")
