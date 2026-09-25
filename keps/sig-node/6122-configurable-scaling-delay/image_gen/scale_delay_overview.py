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
"""Generate scale_delay_overview.svg for KEP-6122.

Run `make` in this directory to regenerate the SVG and the PNG.

Layout is computed, not hand-placed: column positions are derived from the
width of the notes and frames that sit on each lifeline, and the result is
checked afterwards, so no note and no loop/alt frame can cross a lifeline it
does not belong to. Edit the content below, not the coordinates.
"""

import os

FONT = "Helvetica, Arial, sans-serif"
FS_PART = 12.5   # participant labels
FS_TEXT = 11.5   # notes and messages
FS_FRAME = 11.0  # loop/alt labels
LH = 15          # line height

ACCENT = "#14489c"   # code identifiers
INK = "#222222"
LINE = "#555555"
LIFE = "#9a9a9a"
NOTE_BG = "#fdfbe4"
NOTE_BD = "#b9b48a"
PART_BG = "#f4f4f8"
PART_BD = "#333333"
GROUP_BD = "#333333"
OTHER = "#1a6fb5"    # everything that belongs to KEP-6369, not to this KEP

NOTE_PAD_X = 12
NOTE_PAD_Y = 9
FRAME_PAD = 12
TAB_H = 17
MSG_PAD = 8
BAR_W = 11.0
BAR_HALF = BAR_W / 2


def tw(s, size, bold=False):
    """Estimate rendered text width, ignoring ** markers."""
    s = s.replace("**", "")
    w = 0.0
    for ch in s:
        if ch in "iljItf.,:;'|!()[]{}":
            w += 0.31
        elif ch in "mwMW@":
            w += 0.90
        elif ch.isupper() or ch.isdigit():
            w += 0.66
        elif ch == " ":
            w += 0.30
        else:
            w += 0.55
    return w * size * (1.06 if bold else 1.0)


def lines_w(lines, size):
    return max(tw(l, size) for l in lines)


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def rich(line, size, anchor_x, y, ink=INK, bold=ACCENT):
    """Render a line with **bold** segments as flowing tspans."""
    parts = line.split("**")
    out = [f'<text xml:space="preserve" x="{anchor_x:.1f}" y="{y:.1f}" '
           f'font-family="{FONT}" font-size="{size}" fill="{ink}">']
    for i, p in enumerate(parts):
        if not p:
            continue
        if i % 2:
            out.append(f'<tspan font-weight="bold" fill="{bold}">{esc(p)}</tspan>')
        else:
            out.append(f'<tspan>{esc(p)}</tspan>')
    out.append("</text>")
    return "".join(out)


# ---------------------------------------------------------------- content ----

NOTE_ALLOC = [
    "1. Allocate the new CPUSet",
    "    into **preAssignments**",
    "2. Arm **scale_delay_timer** for",
    "    **scaleDownGracePeriodSeconds**",
    "3. Record preAssignments, the",
    "    deadline and the boot ID",
    "    in the **checkpoint**",
]

NOTE_REC = [
    "1. Release removed_CPUs",
    "    to **defaultCPUSet**",
    "2. Set **assignments** from",
    "    preAssignments",
    "3. Update the **checkpoint**",
    "4. Clear **scale_delay_timer**",
    "    and the pending entry",
]

NOTE_SYNC = [
    "If **assignments** equals lastCPUSet",
    "and no **scale_delay_timer** is active",
    "1. Clear the Pod Resize",
    "    InProgress status",
    "2. Update container",
    "    actual resources",
]

# key, label, group, kind
LANES = [
    ("actor", "Actor", None, "actor"),
    ("api", "API Server", None, "part"),
    ("alloc", "CPU Allocate", "cpumgr", "part"),
    ("rec", "Reconcile Loop", "cpumgr", "part"),
    # Volume Manager sits next to the CPU manager so that the KEP-6369 frame
    # around the two of them does not have to swallow PLEG and SyncPod
    ("vol", "Volume Manager", "kubelet", "part"),
    ("pleg", "PLEG", "kubelet", "part"),
    ("sync", "SyncPod", "kubelet", "part"),
    ("rt", "Runtime", None, "part"),
    ("ctr", "Container", None, "block"),
]

# lanes that stay busy for the whole diagram: one bar from top to bottom
FULL_BARS = {"rec", "pleg", "sync", "vol", "rt"}

# drawn inside a block rectangle, at the height of the arrow from that lane
BLOCK_INNER = {
    "ctr": ("vol", ["downward API file", "/etc/podinfo/assigned_cpuset"], OTHER),
}

BODY = [
    ("act", "api", 1),
    ("msg", "actor", "api", ["Scale down", "patch command"]),
    ("act", "alloc", 1),
    ("msg", "api", "alloc", ["Scale down"]),
    ("act", "api", 0),
    ("note", "alloc", NOTE_ALLOC),
    # not this KEP's work, but the reason the delay is useful at all
    ("frame", "opt", "scope of KEP-6369: expose the new cpuset to the container",
     ("alloc", "vol"), [
         ("msg", "alloc", "vol", ["Get cpuset", "(preAssignments)"], OTHER),
         ("msg", "vol", "ctr", ["write cpuset"], OTHER),
     ], OTHER),
    ("frame", "loop", "for each container with a scale_delay_timer", "rec", [
        ("frame", "alt", "scale_delay_timer exceeds the grace period", "rec", [
            ("note", "rec", NOTE_REC),
        ]),
    ]),
    ("act", "alloc", 0),
    ("frame", "loop", "for each container", "rec", [
        ("frame", "alt", "assignments differ from the last applied ones", "rec", [
            ("act", "rt", 1),
            ("msg", "rec", "rt", ["updateContainerCPUSet"]),
            ("msg", "rt", "ctr", ["apply cpuset"]),
            ("act", "rt", 0),
            # the arrow leaves the box itself, not the lifeline below it
            ("boxmsg", "rec", "Trigger PLEG event", "pleg",
             ["SetPodWatchCondition"]),
        ]),
    ]),
    ("act", "rec", 0),
    ("act", "sync", 1),
    ("msg", "pleg", "sync", ["SyncLoop (PLEG)"]),
    ("act", "pleg", 0),
    ("note", "sync", NOTE_SYNC),
    ("act", "sync", 0),
]

keys = [k for k, _, _, _ in LANES]
kind = {k: t for k, _, _, t in LANES}
label_of = {k: l for k, l, _, _ in LANES}

# ------------------------------------------------------------- horizontal ----

box_w = {}
for k, label, _, t in LANES:
    if t == "actor":
        box_w[k] = 34
    elif t == "block":
        box_w[k] = max(118, tw(label, FS_PART) + 44)
        if k in BLOCK_INNER:
            box_w[k] = max(box_w[k],
                           lines_w(BLOCK_INNER[k][1], FS_TEXT) + 24 + 32)
    else:
        box_w[k] = max(72, tw(label, FS_PART) + 26)

need_l = {k: 0.0 for k in keys}
need_r = {k: 0.0 for k in keys}


def claim(key, half):
    need_l[key] = max(need_l[key], half)
    need_r[key] = max(need_r[key], half)


claim("alloc", lines_w(NOTE_ALLOC, FS_TEXT) / 2 + NOTE_PAD_X)
claim("sync", lines_w(NOTE_SYNC, FS_TEXT) / 2 + NOTE_PAD_X)
# the note on rec is wrapped in an alt inside a loop
claim("rec", lines_w(NOTE_REC, FS_TEXT) / 2 + NOTE_PAD_X + 2 * FRAME_PAD)

# a block is drawn as a rectangle, so half of it must fit beside its neighbour
for k in keys:
    if kind[k] == "block":
        need_l[k] = max(need_l[k], box_w[k] / 2)
        need_r[k] = max(need_r[k], box_w[k] / 2)


def walk(els):
    for e in els:
        yield e
        if e[0] == "frame":
            yield from walk(e[4])


# pairs of neighbours whose gap must hold two things side by side at one height
hard = {}


def depth_walk(els, d=0):
    for e in els:
        yield e, d
        if e[0] == "frame":
            yield from depth_walk(e[4], d + 1)


for e, d in depth_walk(BODY):
    if e[0] == "msg":
        a, b, lab = e[1], e[2], e[3]
        w = lines_w(lab, FS_TEXT) + 2 * MSG_PAD + d * FRAME_PAD
        # the label is drawn on the source side of the arrow
        side = need_r if keys.index(b) > keys.index(a) else need_l
        side[a] = max(side[a], w)
        if kind[b] == "block" and abs(keys.index(a) - keys.index(b)) == 1:
            pair = (a, b) if keys.index(b) > keys.index(a) else (b, a)
            hard[pair] = max(hard.get(pair, 0), w + box_w[b] / 2)
    elif e[0] == "box":
        claim(e[1], tw(e[2], FS_TEXT) / 2 + 14 + d * FRAME_PAD)
    elif e[0] == "boxmsg":
        _, a, text, b, lab = e
        half = tw(text, FS_TEXT) / 2 + 14
        claim(a, half + d * FRAME_PAD)
        side = need_r if keys.index(b) > keys.index(a) else need_l
        side[a] = max(side[a], half + MSG_PAD + lines_w(lab, FS_TEXT)
                      + MSG_PAD + d * FRAME_PAD)
    elif e[0] == "frame":
        # a frame is never narrower than its tab plus its condition text
        minw = tw(e[1], FS_FRAME, True) + 18 + 10 + tw(f"[{e[2]}]", FS_FRAME) + 14
        if isinstance(e[3], str):
            claim(e[3], minw / 2 + (d + 1) * FRAME_PAD)

x = {}


def place():
    x.clear()
    x[keys[0]] = 30.0
    for i in range(1, len(keys)):
        prev, cur = keys[i - 1], keys[i]
        gap = max(box_w[prev] / 2 + box_w[cur] / 2 + 26,
                  need_r[prev] + 16, need_l[cur] + 16,
                  hard.get((prev, cur), 0) + 16)
        x[cur] = x[prev] + gap

# --------------------------------------------------------------- vertical ----

TOP = 10
Y_KUB = TOP
Y_CPU = TOP + 23
Y_BOX = TOP + 52
BOX_H = 30
Y_GROUP_BOT = Y_BOX + BOX_H + 7
Y_LIFE = Y_BOX + BOX_H


FRAME_GEOM = []


def run_layout(forced):
    """Lay out the body. Returns (draw ops, bars, blocks, end y, measured spans)."""
    draw, bars, measured = [], {}, {}
    blocks, checks = {}, []
    FRAME_GEOM.clear()
    state = {"active": set(FULL_BARS), "pending": [], "open": {},
             "last_bottom": Y_LIFE}

    def start_pending(y):
        for k in state["pending"]:
            state["open"][k] = y
            state["active"].add(k)
        state["pending"] = []

    def note_at(key, lines, y):
        start_pending(y)
        w = lines_w(lines, FS_TEXT) + 2 * NOTE_PAD_X
        h = len(lines) * LH + 2 * NOTE_PAD_Y
        x0 = x[key] - w / 2
        draw.append(f'<rect x="{x0:.1f}" y="{y:.1f}" width="{w:.1f}" '
                    f'height="{h:.1f}" fill="{NOTE_BG}" stroke="{NOTE_BD}" '
                    f'stroke-width="1"/>')
        for i, l in enumerate(lines):
            draw.append(rich(l, FS_TEXT, x0 + NOTE_PAD_X,
                             y + NOTE_PAD_Y + (i + 1) * LH - 4))
        checks.append((x0, x0 + w, key, f"note on {key}"))
        return x0, y, x0 + w, y + h

    def box_at(key, text, y):
        start_pending(y)
        w = tw(text, FS_TEXT) + 28
        h = 26
        x0 = x[key] - w / 2
        draw.append(f'<rect x="{x0:.1f}" y="{y:.1f}" width="{w:.1f}" '
                    f'height="{h}" fill="#ffffff" stroke="{PART_BD}" '
                    f'stroke-width="1.2"/>')
        draw.append(f'<text x="{x[key]:.1f}" y="{y + h / 2 + 4:.1f}" '
                    f'text-anchor="middle" font-family="{FONT}" '
                    f'font-size="{FS_TEXT}" fill="{INK}">{esc(text)}</text>')
        checks.append((x0, x0 + w, key, f"box {text!r} on {key}"))
        return x0, y, x0 + w, y + h

    def boxmsg_at(src, text, dst, lab, y):
        bx0, by0, bx1, by1 = box_at(src, text, y)
        ay = (by0 + by1) / 2
        d = 1 if x[dst] > x[src] else -1
        x0 = bx1 if d > 0 else bx0
        x1 = x[dst] - d * (BAR_HALF if dst in state["active"] else 0)
        for i, l in enumerate(lab):
            draw.append(rich(l, FS_TEXT, x0 + MSG_PAD,
                             ay - 6 - (len(lab) - 1 - i) * LH))
        draw.append(f'<line x1="{x0:.1f}" y1="{ay:.1f}" x2="{x1:.1f}" '
                    f'y2="{ay:.1f}" stroke="{INK}" stroke-width="1.2"/>')
        draw.append(f'<path d="M {x1:.1f} {ay:.1f} l {-d * 9} -4 l 0 8 z" '
                    f'fill="{INK}"/>')
        lw = lines_w(lab, FS_TEXT)
        if d > 0:
            return bx0, by0, bx1 + MSG_PAD + lw + MSG_PAD, by1
        return bx0 - MSG_PAD - lw - MSG_PAD, by0, bx1, by1

    def msg_at(src, dst, lab, y, colour=INK):
        ay = y + len(lab) * LH + 3
        start_pending(ay - 8)
        x0, x1 = x[src], x[dst]
        right = x1 > x0
        d = 1 if right else -1
        if src in state["active"]:
            x0 += d * BAR_HALF
        if kind[dst] == "block":
            rect = blocks.setdefault(dst, {"top": ay, "bottom": ay, "at": {}})
            rect["top"] = min(rect["top"], ay)
            rect["bottom"] = max(rect["bottom"], ay)
            rect["at"][src] = ay
            x1 = x[dst] - d * box_w[dst] / 2
        elif dst in state["active"]:
            x1 -= d * BAR_HALF
        tx = min(x[src], x[dst]) + MSG_PAD
        for i, l in enumerate(lab):
            draw.append(rich(l, FS_TEXT, tx, y + (i + 1) * LH - 3, colour,
                             colour))
        draw.append(f'<line x1="{x0:.1f}" y1="{ay:.1f}" x2="{x1:.1f}" '
                    f'y2="{ay:.1f}" stroke="{colour}" stroke-width="1.2"/>')
        draw.append(f'<path d="M {x1:.1f} {ay:.1f} l {-d * 9} -4 l 0 8 z" '
                    f'fill="{colour}"/>')
        # the reported extent covers the source side and the label only: an
        # arrow is allowed to leave the frame it starts in
        lw = lines_w(lab, FS_TEXT)
        if right:
            return x[src] - MSG_PAD, y, x[src] + lw + 2 * MSG_PAD, ay + 4
        return x[src] - lw - 2 * MSG_PAD, y, x[src] + MSG_PAD, ay + 4

    def layout(children, y, owner=None):
        bx0, bx1, by1 = 1e9, -1e9, y
        for e in children:
            if e[0] == "act":
                _, k, on = e
                if k in FULL_BARS:
                    continue
                if on:
                    state["pending"].append(k)
                else:
                    if k in state["open"]:
                        bars.setdefault(k, []).append(
                            (state["open"].pop(k), state["last_bottom"] + 4))
                    state["active"].discard(k)
                continue
            if e[0] == "note":
                r = note_at(e[1], e[2], y + 6)
                y = r[3] + 14
                own = True
            elif e[0] == "box":
                r = box_at(e[1], e[2], y + 8)
                y = r[3] + 12
                own = True
            elif e[0] == "boxmsg":
                r = boxmsg_at(e[1], e[2], e[3], e[4], y + 8)
                y = r[3] + 12
                own = True
            elif e[0] == "msg":
                r = msg_at(e[1], e[2], e[3], y + 4, e[4] if len(e) > 4 else INK)
                y = r[3] + 14
                own = owner is None or e[1] in owner
            else:
                fkind, cond, fowner, kids = e[1], e[2], e[3], e[4]
                colour = e[5] if len(e) > 5 else LINE
                owners = (fowner,) if isinstance(fowner, str) else fowner
                ftop = y + 8
                at = len(draw)
                y2, kb = layout(kids, ftop + TAB_H + 6, owners)
                tabw = tw(fkind, FS_FRAME, True) + 18
                minw = tabw + 10 + tw(f"[{cond}]", FS_FRAME) + 14
                fx0 = min([kb[0]] + [x[k] for k in owners]) - FRAME_PAD
                fx1 = max([kb[1]] + [x[k] for k in owners]) + FRAME_PAD
                if fkind in forced:
                    fx0, fx1 = forced[fkind]
                elif fx1 - fx0 < minw:
                    grow = (minw - (fx1 - fx0)) / 2
                    fx0, fx1 = fx0 - grow, fx1 + grow
                measured[fkind] = (min(measured.get(fkind, (fx0, fx1))[0], fx0),
                                   max(measured.get(fkind, (fx0, fx1))[1], fx1))
                fbot = y2 - 6
                tint = "none" if colour == LINE else colour
                draw[at:at] = [
                    f'<rect x="{fx0:.1f}" y="{ftop:.1f}" width="{fx1 - fx0:.1f}" '
                    f'height="{fbot - ftop:.1f}" fill="{tint}" '
                    f'fill-opacity="0.05" stroke="{colour}" '
                    f'stroke-width="1" stroke-dasharray="4 3"/>',
                    f'<path d="M {fx0:.1f} {ftop:.1f} h {tabw:.1f} '
                    f'l 0 {TAB_H - 6} l -6 6 h {-(tabw - 6):.1f} z" '
                    f'fill="#ffffff" stroke="{colour}" stroke-width="1"/>',
                    f'<text x="{fx0 + 9:.1f}" y="{ftop + TAB_H - 5:.1f}" '
                    f'font-family="{FONT}" font-size="{FS_FRAME}" '
                    f'font-weight="bold" fill="{colour}">{esc(fkind)}</text>',
                    f'<text x="{fx0 + tabw + 10:.1f}" y="{ftop + TAB_H - 5:.1f}" '
                    f'font-family="{FONT}" font-size="{FS_FRAME}" '
                    f'fill="{colour}">[{esc(cond)}]</text>',
                ]
                checks.append((fx0, fx1, owners, f"{fkind} on {fowner}"))
                FRAME_GEOM.append((fkind, ftop, fbot))
                r = (fx0, ftop, fx1, fbot)
                y = fbot + 16
                own = True
            state["last_bottom"] = r[3]
            if own:
                bx0, bx1 = min(bx0, r[0]), max(bx1, r[2])
            by1 = max(by1, r[3])
        return y, (bx0, bx1, by1)

    end, _ = layout(BODY, Y_LIFE + 18)
    for k, y0 in state["open"].items():
        bars.setdefault(k, []).append((y0, state["last_bottom"] + 4))
    return draw, bars, blocks, end, measured, checks


def crossings(checks):
    """Everything drawn on a lifeline must stay between its neighbours."""
    out = []
    for x0, x1, owner, what in checks:
        owners = (owner,) if isinstance(owner, str) else owner
        lo = min(keys.index(k) for k in owners)
        hi = max(keys.index(k) for k in owners)
        for k in keys:
            # a frame that spans several lanes may of course contain them,
            # and the ones in between
            if kind[k] == "block" or lo <= keys.index(k) <= hi:
                continue
            if x0 + 1 < x[k] < x1 - 1:
                out.append((x0, x1, owner, what, k))
    return out


# Widen the columns until nothing crosses a foreign lifeline. A frame grows
# around its content, not around its lifeline, so how wide it ends up cannot
# be predicted before laying it out; measuring and retrying is exact.
for attempt in range(20):
    place()
    # pass 1 measures the loop/alt spans, pass 2 aligns every loop and alt
    *_, spans, _ = run_layout({})
    draw, bars, blocks, body_end, _, checks = run_layout(spans)
    bad = crossings(checks)
    if not bad:
        print(f"no lifeline crossings (after {attempt} widening passes)")
        break
    for x0, x1, owner, what, k in bad:
        owners = (owner,) if isinstance(owner, str) else owner
        # widen on the side the crossed lifeline sits on, at the nearest owner
        if x[k] > max(x[o] for o in owners):
            near = max(owners, key=lambda o: x[o])
            need_r[near] = max(need_r[near], x1 - x[near] + 10)
        else:
            near = min(owners, key=lambda o: x[o])
            need_l[near] = max(need_l[near], x[near] - x0 + 10)
else:
    print("GAVE UP, still crossing:")
    for b in bad:
        print("  !", b[3], "crosses", b[4])

HEIGHT = max([body_end] + [b["bottom"] + 44 for b in blocks.values()]) + 12

# CPU Allocate stays busy into the delay, but not for the whole of it: end its
# bar a third of the way down the first loop so it does not look aligned with it
_loops = [g for g in FRAME_GEOM if g[0] == "loop"]
if _loops and "alloc" in bars:
    _, ftop, fbot = _loops[0]
    y0, _ = bars["alloc"][-1]
    bars["alloc"][-1] = (y0, ftop + (fbot - ftop) / 3)
WIDTH = x[keys[-1]] + box_w[keys[-1]] / 2 + 24

# ------------------------------------------------------------ assemble svg ---

head = []

# group frames enclose only the header rectangles
for gkey, glabel, gy in (("kubelet", "kubelet", Y_KUB),
                         ("cpumgr", "CPU manager", Y_CPU)):
    members = [k for k, _, g, _ in LANES
               if g == gkey or (gkey == "kubelet" and g == "cpumgr")]
    gx0 = min(x[k] - box_w[k] / 2 for k in members) - 12
    gx1 = max(x[k] + box_w[k] / 2 for k in members) + 12
    head.append(f'<rect x="{gx0:.1f}" y="{gy:.1f}" width="{gx1 - gx0:.1f}" '
                f'height="{Y_GROUP_BOT - gy:.1f}" fill="none" '
                f'stroke="{GROUP_BD}" stroke-width="1.2"/>')
    head.append(f'<text x="{(gx0 + gx1) / 2:.1f}" y="{gy + 15:.1f}" '
                f'text-anchor="middle" font-family="{FONT}" '
                f'font-size="{FS_PART}" font-weight="bold" fill="{INK}">'
                f'{esc(glabel)}</text>')

for k in keys:
    if kind[k] == "block":
        continue
    head.append(f'<line x1="{x[k]:.1f}" y1="{Y_LIFE:.1f}" x2="{x[k]:.1f}" '
                f'y2="{HEIGHT - 8:.1f}" stroke="{LIFE}" stroke-width="1" '
                f'stroke-dasharray="5 4"/>')

for k, label, _, t in LANES:
    if t == "block":
        b = blocks[k]
        top, bot = b["top"] - 40, b["bottom"] + 40
        w = box_w[k]
        if k in BLOCK_INNER:
            # leave room above the inner box for the block's own label
            inner_h = len(BLOCK_INNER[k][1]) * LH + 12
            top = min(top, b["at"][BLOCK_INNER[k][0]] - inner_h / 2 - 32)
        head.append(f'<rect x="{x[k] - w / 2:.1f}" y="{top:.1f}" width="{w:.1f}" '
                    f'height="{bot - top:.1f}" rx="2" fill="#ffffff" '
                    f'stroke="{PART_BD}" stroke-width="1.2"/>')
        head.append(f'<text x="{x[k]:.1f}" y="{top + 20:.1f}" '
                    f'text-anchor="middle" font-family="{FONT}" '
                    f'font-size="{FS_PART}" fill="{INK}">{esc(label)}</text>')
        if k in BLOCK_INNER:
            src, lines, colour = BLOCK_INNER[k]
            iw = lines_w(lines, FS_TEXT) + 24
            ih = len(lines) * LH + 12
            iy = b["at"][src] - ih / 2
            head.append(f'<rect x="{x[k] - iw / 2:.1f}" y="{iy:.1f}" '
                        f'width="{iw:.1f}" height="{ih:.1f}" fill="#ffffff" '
                        f'stroke="{colour}" stroke-width="1.1"/>')
            for i, l in enumerate(lines):
                head.append(
                    f'<text x="{x[k]:.1f}" y="{iy + 6 + (i + 1) * LH - 4:.1f}" '
                    f'text-anchor="middle" font-family="{FONT}" '
                    f'font-size="{FS_TEXT}" fill="{colour}">{esc(l)}</text>')
        continue
    if t == "actor":
        cx, cy = x[k], Y_BOX + 4
        head.append(f'<circle cx="{cx:.1f}" cy="{cy:.1f}" r="6" fill="none" '
                    f'stroke="{PART_BD}" stroke-width="1.2"/>')
        head.append(f'<path d="M {cx:.1f} {cy + 6:.1f} v 12 M {cx - 8:.1f} '
                    f'{cy + 10:.1f} h 16 M {cx:.1f} {cy + 18:.1f} l -7 9 '
                    f'M {cx:.1f} {cy + 18:.1f} l 7 9" fill="none" '
                    f'stroke="{PART_BD}" stroke-width="1.2"/>')
        head.append(f'<text x="{cx:.1f}" y="{Y_LIFE + 10:.1f}" '
                    f'text-anchor="middle" font-family="{FONT}" '
                    f'font-size="{FS_PART}" fill="{INK}">{esc(label)}</text>')
        continue
    w = box_w[k]
    head.append(f'<rect x="{x[k] - w / 2:.1f}" y="{Y_BOX:.1f}" width="{w:.1f}" '
                f'height="{BOX_H}" rx="2" fill="{PART_BG}" stroke="{PART_BD}" '
                f'stroke-width="1.2"/>')
    head.append(f'<text x="{x[k]:.1f}" y="{Y_BOX + BOX_H / 2 + 4.5:.1f}" '
                f'text-anchor="middle" font-family="{FONT}" '
                f'font-size="{FS_PART}" fill="{INK}">{esc(label)}</text>')

bar_ops = []
# sorted, so that the generated SVG is byte for byte the same on every run
for k in sorted(FULL_BARS):
    bars.setdefault(k, []).append((Y_LIFE, HEIGHT - 8))
for k, spans_ in bars.items():
    for y0, y1 in spans_:
        bar_ops.append(f'<rect x="{x[k] - BAR_HALF:.1f}" y="{y0:.1f}" '
                       f'width="{BAR_W:.1f}" height="{max(y1 - y0, 14):.1f}" '
                       f'fill="#ffffff" stroke="{PART_BD}" stroke-width="1.1"/>')

svg = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{WIDTH:.0f}" '
       f'height="{HEIGHT:.0f}" viewBox="0 0 {WIDTH:.0f} {HEIGHT:.0f}">',
       '<rect width="100%" height="100%" fill="#ffffff"/>']
svg += head + bar_ops + draw + ["</svg>"]

out = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                   "scale_delay_overview.svg")
with open(out, "w") as f:
    f.write("\n".join(svg) + "\n")
print(f"DIM {WIDTH:.0f} {HEIGHT:.0f}")
for k in keys:
    print(f"  {k:6s} x={x[k]:7.1f}  L={need_l[k]:6.1f} R={need_r[k]:6.1f}")
print("  spans:", {k: (round(v[0]), round(v[1])) for k, v in spans.items()})
