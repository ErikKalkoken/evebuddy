"""Generate go file that maps Eve icons IDs to resource variables."""

from pathlib import Path
import json

# IDs to ignore, e.g. because we have not PNG file for it
blacklist = {21934}

data_file = Path(__file__).parent / "data.json"

with data_file.open("r") as f:
    d = json.load(f)

out_file = Path(__file__).parent / "icons_map.go"

out = """
// auto-generated
package icons

import "fyne.io/fyne/v2"

var id2fileMap = map[int]*fyne.StaticResource{
"""

for row in d:
    id = int(row['id'])
    if id in blacklist:
        continue
    file: str = row['file']
    name = file.replace("_", "").replace(".", "").replace("png", "Png")
    out += f"\t{id}: resource{name},\n"  # resource1006412Png

out += "}\n"

with out_file.open("w") as f:
    f.write(out)
