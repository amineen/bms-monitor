"""Import the layered ARC communication DXF into a new Fusion design."""

import adsk.core
import adsk.fusion
import json


DXF_PATH = "/Users/aaronmineen/Developer/NRECA/TEC-Retrofit/Generator-Commissioning-TEC/BESS Installation And commissioning/bms-monitor/docs/arc-communication-diagram.dxf"
PRIMITIVES_PATH = "/Users/aaronmineen/Developer/NRECA/TEC-Retrofit/Generator-Commissioning-TEC/BESS Installation And commissioning/bms-monitor/docs/cad/arc-communication-diagram.primitives.json"


def run(_context: str):
    app = adsk.core.Application.get()
    design = adsk.fusion.Design.cast(app.activeProduct)
    has_arc = design is not None and any(
        design.rootComponent.sketches.item(i).name.startswith("ARC Communication CAD")
        for i in range(design.rootComponent.sketches.count)
    )
    if design is None or (design.rootComponent.sketches.count and not has_arc) or design.rootComponent.bRepBodies.count:
        app.documents.add(adsk.core.DocumentTypes.FusionDesignDocumentType)
        design = adsk.fusion.Design.cast(app.activeProduct)
    root = design.rootComponent

    if has_arc:
        for index in range(root.sketches.count - 1, -1, -1):
            sketch = root.sketches.item(index)
            if sketch.name.startswith("ARC Communication CAD") or sketch.name == "ARC Communication Text":
                sketch.deleteMe()

    manager = app.importManager
    options = manager.createDXF2DImportOptions(DXF_PATH, root.xYConstructionPlane)
    options.isViewFit = True
    manager.importToTarget(options, root)
    cad_index = 0
    for index in range(root.sketches.count):
        sketch = root.sketches.item(index)
        if sketch.name.startswith("Sketch"):
            cad_index += 1
            sketch.name = "ARC Communication CAD" if cad_index == 1 else "ARC Communication CAD %02d" % cad_index

    text_count = 0
    text_sketch = None
    if text_sketch is None:
        data = json.load(open(PRIMITIVES_PATH, encoding="utf-8"))
        height = data["height"]
        scale_cm = data["scale_mm"] / 10.0
        text_sketch = root.sketches.add(root.xYConstructionPlane)
        text_sketch.name = "ARC Communication Text"
        text_sketch.isComputeDeferred = True
        texts = text_sketch.sketchTexts
        horizontal = adsk.core.HorizontalAlignments
        vertical = adsk.core.VerticalAlignments
        valign = getattr(vertical, "MiddleVerticalAlignment", None) or getattr(vertical, "CenterVerticalAlignment")
        value = adsk.core.ValueInput.createByReal

        def point(x, y):
            return adsk.core.Point3D.create(x * scale_cm, (height - y) * scale_cm, 0)

        for primitive in data["primitives"]:
            if primitive["type"] != "text":
                continue
            x, y = primitive["x"], primitive["y"]
            label = primitive["text"].replace("'", "")
            size = primitive["size"]
            factor = 1.0 if size >= 20 else 0.90
            if x == 40 and y == 80:
                factor = 0.72
            elif x == 1225 and y == 101:
                factor = 0.70
            elif x == 898 and y == 772:
                factor = 0.78
            elif y == 592:
                factor = 0.85
            cad_size = size * factor
            width = max(len(label) * cad_size * 0.56, cad_size * 2)
            anchor = primitive["anchor"]
            if anchor == "middle":
                xa, xb, halign = x - width / 2, x + width / 2, horizontal.CenterHorizontalAlignment
            elif anchor == "end":
                xa, xb, halign = x - width, x, horizontal.RightHorizontalAlignment
            else:
                xa, xb, halign = x, x + width, horizontal.LeftHorizontalAlignment
            text_input = texts.createInput3("'" + label + "'", value(cad_size * scale_cm))
            text_input.setAsMultiLine(point(xa, y - cad_size * 0.85), point(xb, y + cad_size * 0.35), halign, valign, 0)
            texts.add(text_input)
            text_count += 1
        text_sketch.isComputeDeferred = False

    viewport = app.activeViewport
    camera = viewport.camera
    camera.cameraType = adsk.core.CameraTypes.OrthographicCameraType
    camera.viewOrientation = adsk.core.ViewOrientations.TopViewOrientation
    camera.isSmoothTransition = False
    viewport.camera = camera
    viewport.fit()
    app.userInterface.activeSelections.clear()
    adsk.doEvents()
    print("imported %s | sketches:%d | added texts:%d | document is unsaved" % (DXF_PATH, root.sketches.count, text_count))
