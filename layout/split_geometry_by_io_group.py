#!/usr/bin/env python3

import argparse
import json
import os
from collections import defaultdict


def main(geometry_json, outdir):
    with open(geometry_json) as fi:
        geo = json.load(fi)

    # Keys of `geometry` are 'iog-tile-chip-channel'
    geometry = defaultdict(dict)
    for key, xy in geo['geometry'].items():
        io_group = int(key.split('-')[0])
        geometry[io_group][key] = xy

    # Keys of `io_group_io_channel_to_tile` are 'iog-iochan'
    iog_iochan_to_tile = defaultdict(dict)
    tile_to_iochans = defaultdict(lambda: defaultdict(set))
    for key, tile in geo['io_group_io_channel_to_tile'].items():
        io_group, io_channel = map(int, key.split('-'))
        iog_iochan_to_tile[io_group][key] = tile
        tile_to_iochans[io_group][str(tile)].add(io_channel)

    tile_to_io_channels = defaultdict(dict)
    for io_group, tiles in tile_to_iochans.items():
        for tile, iochans in sorted(tiles.items(), key=lambda t: int(t[0])):
            tile_to_io_channels[io_group][tile] = [
                ch for ch in geo['tile_to_io_channels'][tile] if ch in iochans]

    os.makedirs(outdir, exist_ok=True)
    base = os.path.splitext(os.path.basename(geometry_json))[0]

    for io_group in sorted(set(geometry) | set(iog_iochan_to_tile)):
        d = dict()
        d['geometry'] = geometry[io_group]
        d['pixel_pitch'] = geo['pixel_pitch']
        d['tile_to_io_channels'] = tile_to_io_channels[io_group]
        d['io_group_io_channel_to_tile'] = iog_iochan_to_tile[io_group]

        outfile_path = os.path.join(outdir, f'{base}_iog{io_group}.json')
        with open(outfile_path, 'w') as outfile:
            json.dump(d, outfile)
        print(f'Wrote {outfile_path}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='Split the output of simplified_geometry.py by IO group')
    parser.add_argument('geometry_json', type=str,
                        help='''JSON file produced by simplified_geometry.py''')
    parser.add_argument('outdir', type=str,
                        help='''Directory in which to write per-IO-group JSON files''')
    args = parser.parse_args()

    main(**vars(args))
