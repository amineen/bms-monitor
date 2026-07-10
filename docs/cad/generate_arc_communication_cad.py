#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.10"
# dependencies = ["ezdxf>=1.4,<2"]
# ///
"""Convert the ARC communication SVG into layered, editable A3 CAD.

The SVG remains the visual source of truth. This converter preserves its exact
layout while mapping colors and line styles onto named DXF layers.
"""

from __future__ import annotations

import json
import math
import re
from pathlib import Path
from xml.etree import ElementTree as ET

import ezdxf
from ezdxf.colors import rgb2int
from ezdxf.enums import TextEntityAlignment
from ezdxf.lldxf.const import DXF2018


HERE = Path(__file__).resolve().parent
DOCS = HERE.parent
SOURCE = DOCS / "arc-communication-diagram.svg"
DXF_OUT = DOCS / "arc-communication-diagram.dxf"
PRIMITIVES_OUT = HERE / "arc-communication-diagram.primitives.json"

SCALE = 0.25  # SVG px -> mm; 1680 px becomes A3 landscape width (420 mm)
SVG_NS = "{http://www.w3.org/2000/svg}"


def css_rules(root: ET.Element) -> dict[str, dict[str, str]]:
    rules: dict[str, dict[str, str]] = {}
    style = root.find(f".//{SVG_NS}style")
    if style is None or not style.text:
        return rules
    for selector, body in re.findall(r"\.([\w-]+)\s*\{([^}]*)\}", style.text):
        declarations: dict[str, str] = {}
        for item in body.split(";"):
            if ":" in item:
                key, value = item.split(":", 1)
                declarations[key.strip()] = value.strip()
        rules[selector] = declarations
    return rules


def merged_style(element: ET.Element, rules: dict[str, dict[str, str]]) -> dict[str, str]:
    style: dict[str, str] = {}
    for name in element.get("class", "").split():
        style.update(rules.get(name, {}))
    for key in (
        "fill", "stroke", "stroke-width", "stroke-dasharray", "font-size",
        "font-weight", "font-family", "text-anchor", "letter-spacing",
    ):
        if key in element.attrib:
            style[key] = element.attrib[key]
    if element.get("style"):
        for item in element.get("style", "").split(";"):
            if ":" in item:
                key, value = item.split(":", 1)
                style[key.strip()] = value.strip()
    return style


def rgb(value: str | None, background=(255, 255, 255)) -> tuple[int, int, int] | None:
    if not value or value in {"none", "transparent"}:
        return None
    if not value.startswith("#"):
        return None
    raw = value[1:]
    if len(raw) in (3, 4):
        raw = "".join(ch * 2 for ch in raw)
    if len(raw) not in (6, 8):
        return None
    base = tuple(int(raw[i:i + 2], 16) for i in (0, 2, 4))
    if len(raw) == 6:
        return base
    alpha = int(raw[6:8], 16) / 255.0
    return tuple(round(base[i] * alpha + background[i] * (1 - alpha)) for i in range(3))


def number(value: str | None, default: float = 0.0) -> float:
    if value is None:
        return default
    match = re.search(r"[-+]?(?:\d*\.\d+|\d+)", value)
    return float(match.group(0)) if match else default


def rounded_rect_points(x: float, y: float, w: float, h: float, radius: float) -> list[tuple[float, float]]:
    radius = min(max(radius, 0.0), w / 2, h / 2)
    if radius == 0:
        return [(x, y), (x + w, y), (x + w, y + h), (x, y + h)]
    points: list[tuple[float, float]] = []
    corners = (
        (x + w - radius, y + radius, -90, 0),
        (x + w - radius, y + h - radius, 0, 90),
        (x + radius, y + h - radius, 90, 180),
        (x + radius, y + radius, 180, 270),
    )
    for cx, cy, a0, a1 in corners:
        for step in range(5):
            angle = math.radians(a0 + (a1 - a0) * step / 4)
            points.append((cx + radius * math.cos(angle), cy + radius * math.sin(angle)))
    return points


PATH_TOKEN = re.compile(r"[A-Za-z]|[-+]?(?:\d*\.\d+|\d+)(?:[eE][-+]?\d+)?")


def path_polylines(data: str) -> list[list[tuple[float, float]]]:
    tokens = PATH_TOKEN.findall(data.replace(",", " "))
    paths: list[list[tuple[float, float]]] = []
    points: list[tuple[float, float]] = []
    x = y = 0.0
    command = ""
    i = 0

    def flush() -> None:
        nonlocal points
        if len(points) >= 2:
            paths.append(points)
        points = []

    while i < len(tokens):
        token = tokens[i]
        if token.isalpha():
            command = token
            i += 1
            if command in "Zz":
                if points and points[-1] != points[0]:
                    points.append(points[0])
                flush()
                continue
        if command in "Mm":
            nx, ny = float(tokens[i]), float(tokens[i + 1]); i += 2
            if command == "m":
                nx += x; ny += y
            flush(); x, y = nx, ny; points = [(x, y)]
            command = "l" if command == "m" else "L"
        elif command in "Ll":
            nx, ny = float(tokens[i]), float(tokens[i + 1]); i += 2
            if command == "l":
                nx += x; ny += y
            x, y = nx, ny; points.append((x, y))
        elif command in "Hh":
            nx = float(tokens[i]); i += 1
            x = x + nx if command == "h" else nx
            points.append((x, y))
        elif command in "Vv":
            ny = float(tokens[i]); i += 1
            y = y + ny if command == "v" else ny
            points.append((x, y))
        else:
            raise ValueError(f"Unsupported SVG path command {command!r} in {data!r}")
    flush()
    return paths


def semantic_layer(element: ET.Element, style: dict[str, str]) -> str:
    classes = set(element.get("class", "").split())
    if "eth" in classes:
        return "ARC_ETHERNET"
    if "ser" in classes:
        return "ARC_RS485"
    if "net" in classes:
        return "ARC_VPN"
    if "mech" in classes:
        return "ARC_HARDWIRED_IO"
    if "boxgrey" in classes:
        return "ARC_NOT_IN_SERVICE"
    if "zone" in classes:
        return "ARC_ZONES"
    if "box" in classes:
        return "ARC_DEVICE_BOXES"
    if "ip" in classes:
        return "ARC_IP_ADDRESSES"
    if classes.intersection({"t1", "t2", "t3", "zlabel", "lbl"}) or element.tag.endswith("text"):
        fill = (style.get("fill") or "").upper()
        if fill.startswith("#059669"):
            return "ARC_CONTROL_TEXT"
        if fill.startswith("#0284C7") or fill.startswith("#0369A1"):
            return "ARC_MONITOR_TEXT"
        return "ARC_TEXT"
    fill = (style.get("fill") or "").upper()
    stroke = (style.get("stroke") or "").upper()
    if fill.startswith("#059669") or stroke.startswith("#059669"):
        return "ARC_CONTROL"
    if fill.startswith("#0284C7") or stroke.startswith("#0284C7"):
        return "ARC_MONITOR"
    return "ARC_MISC"


def dxf_point(point: tuple[float, float], height: float) -> tuple[float, float]:
    return point[0] * SCALE, (height - point[1]) * SCALE


def add_colored_hatch(msp, points, layer: str, color: tuple[int, int, int]) -> None:
    hatch = msp.add_hatch(dxfattribs={"layer": layer})
    hatch.rgb = color
    hatch.paths.add_polyline_path(points, is_closed=True)


def main() -> None:
    tree = ET.parse(SOURCE)
    root = tree.getroot()
    width = number(root.get("width"))
    height = number(root.get("height"))
    rules = css_rules(root)

    doc = ezdxf.new(DXF2018, setup=True)
    doc.units = ezdxf.units.MM
    doc.header["$EXTMIN"] = (0.0, 0.0, 0.0)
    doc.header["$EXTMAX"] = (width * SCALE, height * SCALE, 0.0)

    if "ARC_DASHED" not in doc.linetypes:
        doc.linetypes.add("ARC_DASHED", pattern=[3.25, 2.0, -1.25], description="ARC dashed")
    if "ARC_DOTTED" not in doc.linetypes:
        doc.linetypes.add("ARC_DOTTED", pattern=[1.625, 0.625, -1.0], description="ARC dotted")

    layer_colors = {
        "ARC_BACKGROUND": (255, 255, 255), "ARC_ZONES": (203, 213, 225),
        "ARC_DEVICE_BOXES": (148, 163, 184), "ARC_NOT_IN_SERVICE": (203, 213, 225),
        "ARC_ETHERNET": (71, 85, 105), "ARC_RS485": (217, 119, 6),
        "ARC_VPN": (2, 132, 199), "ARC_HARDWIRED_IO": (148, 163, 184),
        "ARC_CONTROL": (5, 150, 105), "ARC_MONITOR": (2, 132, 199),
        "ARC_CONTROL_TEXT": (5, 150, 105), "ARC_MONITOR_TEXT": (2, 132, 199),
        "ARC_IP_ADDRESSES": (3, 105, 161), "ARC_TEXT": (15, 23, 42),
        "ARC_MISC": (71, 85, 105),
    }
    for name, color in layer_colors.items():
        if name not in doc.layers:
            doc.layers.add(name, true_color=rgb2int(color))

    # Helvetica's SVG metrics are narrower than the common DXF Arial fallback.
    # Keep text editable while matching the source line lengths and box fit.
    doc.styles.add("ARC_REGULAR", font="Arial.ttf")
    doc.styles.add("ARC_BOLD", font="Arialbd.ttf")
    doc.styles.add("ARC_MONO", font="Menlo.ttc")
    msp = doc.modelspace()
    primitives: list[dict] = []

    for element in root:
        tag = element.tag.removeprefix(SVG_NS)
        if tag == "defs":
            continue
        style = merged_style(element, rules)
        layer = "ARC_BACKGROUND" if tag == "rect" and element.get("width") == str(int(width)) else semantic_layer(element, style)
        stroke = rgb(style.get("stroke"))
        fill = rgb(style.get("fill"))
        stroke_width = number(style.get("stroke-width"), 1.0)
        dashed = bool(style.get("stroke-dasharray"))
        linetype = "ARC_DOTTED" if layer in {"ARC_VPN", "ARC_HARDWIRED_IO"} else ("ARC_DASHED" if dashed else "CONTINUOUS")
        attrs = {"layer": layer, "linetype": linetype, "lineweight": min(211, max(5, round(stroke_width * SCALE * 100)))}

        if tag == "rect":
            x, y = number(element.get("x")), number(element.get("y"))
            w, h = number(element.get("width")), number(element.get("height"))
            radius = number(element.get("rx"))
            points_svg = rounded_rect_points(x, y, w, h, radius)
            points = [dxf_point(p, height) for p in points_svg]
            if fill:
                add_colored_hatch(msp, points, layer, fill)
            if stroke:
                entity = msp.add_lwpolyline(points, close=True, dxfattribs=attrs)
                entity.rgb = stroke
            primitives.append({"type": "rect", "x": x, "y": y, "w": w, "h": h, "rx": radius, "style": style, "layer": layer})
        elif tag == "line":
            points_svg = [(number(element.get("x1")), number(element.get("y1"))), (number(element.get("x2")), number(element.get("y2")))]
            entity = msp.add_lwpolyline([dxf_point(p, height) for p in points_svg], dxfattribs=attrs)
            if stroke:
                entity.rgb = stroke
            primitives.append({"type": "polyline", "points": points_svg, "style": style, "layer": layer})
        elif tag == "path":
            for points_svg in path_polylines(element.get("d", "")):
                entity = msp.add_lwpolyline([dxf_point(p, height) for p in points_svg], dxfattribs=attrs)
                if stroke:
                    entity.rgb = stroke
                primitives.append({"type": "polyline", "points": points_svg, "style": style, "layer": layer})
        elif tag == "circle":
            cx, cy, radius = number(element.get("cx")), number(element.get("cy")), number(element.get("r"))
            center = dxf_point((cx, cy), height)
            if fill:
                polygon_svg = [(cx + radius * math.cos(i * math.tau / 24), cy + radius * math.sin(i * math.tau / 24)) for i in range(24)]
                add_colored_hatch(msp, [dxf_point(p, height) for p in polygon_svg], layer, fill)
            if stroke:
                entity = msp.add_circle(center, radius * SCALE, dxfattribs=attrs)
                entity.rgb = stroke
            primitives.append({"type": "circle", "cx": cx, "cy": cy, "r": radius, "style": style, "layer": layer})
        elif tag == "text":
            content = "".join(element.itertext()).strip()
            if not content:
                continue
            x, y = number(element.get("x")), number(element.get("y"))
            size = number(style.get("font-size"), 12.0)
            weight = style.get("font-weight", "400")
            family = style.get("font-family", "")
            font_style = "ARC_MONO" if "mono" in family.lower() or "menlo" in family.lower() else ("ARC_BOLD" if weight in {"600", "700", "bold"} else "ARC_REGULAR")
            anchor = style.get("text-anchor", "start")
            align = {"start": TextEntityAlignment.LEFT, "middle": TextEntityAlignment.CENTER, "end": TextEntityAlignment.RIGHT}.get(anchor, TextEntityAlignment.LEFT)
            factor = 1.0 if size >= 20 else 0.90
            if x == 40 and y == 80:
                factor = 0.72  # clear the two-column legend
            elif x == 1225 and y == 101:
                factor = 0.70  # clear the right-column status marker
            elif x == 898 and y == 772:
                factor = 0.78  # fit the long hardwired-genset note
            elif y == 592:
                factor = 0.85  # retain badge side padding with CAD font metrics
            cad_size = size * factor
            entity = msp.add_text(content, height=cad_size * SCALE, dxfattribs={"layer": layer, "style": font_style})
            entity.set_placement(dxf_point((x, y), height), align=align)
            if fill:
                entity.rgb = fill
            primitives.append({"type": "text", "x": x, "y": y, "text": content, "size": size, "anchor": anchor, "weight": weight, "family": family, "style": style, "layer": layer})

    doc.set_modelspace_vport(height=height * SCALE, center=(width * SCALE / 2, height * SCALE / 2))
    doc.saveas(DXF_OUT)
    PRIMITIVES_OUT.write_text(json.dumps({"width": width, "height": height, "scale_mm": SCALE, "source": str(SOURCE), "primitives": primitives}, indent=2), encoding="utf-8")
    print(f"written {DXF_OUT}")
    print(f"written {PRIMITIVES_OUT}")
    print(f"entities {len(msp)}; primitives {len(primitives)}; sheet {width * SCALE:g} x {height * SCALE:g} mm")


if __name__ == "__main__":
    main()
