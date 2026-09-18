import json
import numpy as np

with open("geometry_fsdcube.json", "r") as f:
    data = json.load(f)

coords = np.array(list(data["geometry"].values()))  # shape (N, 2)

xmin, ymin = coords.min(axis=0)
xmax, ymax = coords.max(axis=0)

print("xmin:", xmin)
print("xmax:", xmax)
print("ymin:", ymin)
print("ymax:", ymax)
