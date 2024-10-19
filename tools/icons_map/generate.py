"""Generate go file that maps Eve icons IDs to fyne resource variables.

This is needed to find the embedded icon resource to an icon ID.
"""
import json
from pathlib import Path
import sys

# Icon IDs to ignore, e.g. because we have not PNG file for it
BLACKLISTED_ICON_IDS = {21934}

data_file = Path(__file__).parent / "data.json"

with data_file.open("r") as f:
    d = json.load(f)

out = """
// auto-generated
package eveicon

import "fyne.io/fyne/v2"

var id2fileMap = map[int32]*fyne.StaticResource{
"""

for row in d:
    id = int(row["id"])
    if id in BLACKLISTED_ICON_IDS:
        continue
    file: str = row["file"]
    name = file.replace("_", "").replace(".", "").replace("png", "Png")
    out += f"\t{id}: resource{name},\n"  # resource1006412Png

out += "}\n"
sys.stdout.write(out)
