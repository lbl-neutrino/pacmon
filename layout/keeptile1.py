import json

# Load JSON
with open("../../clean/pacmon/layout/geometry_fsd.json", "r") as f:
    data = json.load(f)

# Filter geometry keys
filtered_geometry = {
    k: v for k, v in data["geometry"].items()
    if k.startswith("1-1-")
}

# Replace geometry with filtered version
data["geometry"] = filtered_geometry

# Save back (optional)
with open("geometry_fsdcube.json", "w") as f:
    json.dump(data, f)
