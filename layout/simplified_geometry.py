import yaml
import numpy as np
import argparse
import json
from collections import defaultdict

_default_geometry_yaml = 'multi_tile_layout-3.0.40.yaml'
_default_geometry_yaml_mod2 = 'multi_tile_layout-2.5.16.yaml'
_default_outfile_path = 'geometry.json'


def _default_pxy():
    return (0., 0.)


def _rotate_pixel(pixel_pos, tile_orientation):
    return pixel_pos[0]*tile_orientation[2], pixel_pos[1]*tile_orientation[1]


def main(geometry_yaml=_default_geometry_yaml, outfile_path=_default_outfile_path):

    with open(geometry_yaml) as fi:
        geo = yaml.full_load(fi)

    if 'multitile_layout_version' in geo.keys():
        # Adapted from: https://github.com/larpix/larpix-v2-testing-scripts/blob/master/event-display/evd_lib.py

        pixel_pitch = geo['pixel_pitch']

        chip_channel_to_position = geo['chip_channel_to_position']
        tile_orientations = geo['tile_orientations']
        tile_positions = geo['tile_positions']
        tile_indeces = geo['tile_indeces']
        xs = np.array(list(chip_channel_to_position.values()))[
            :, 0] * pixel_pitch
        ys = np.array(list(chip_channel_to_position.values()))[
            :, 1] * pixel_pitch
        x_size = max(xs)-min(xs)+pixel_pitch
        y_size = max(ys)-min(ys)+pixel_pitch

        tile_geometry = defaultdict(int)
        io_group_io_channel_to_tile = {}
        geometry = defaultdict(_default_pxy)

        for tile in geo['tile_chip_to_io']:
            tile_orientation = tile_orientations[tile]
            tile_geometry[tile] = tile_positions[tile], tile_orientations[tile]
            for chip in geo['tile_chip_to_io'][tile]:
                io_group_io_channel = geo['tile_chip_to_io'][tile][chip]
                io_group = io_group_io_channel//1000
                io_channel = io_group_io_channel % 1000
                io_group_io_channel_to_tile[(
                    io_group, io_channel)] = tile

            for chip_channel in geo['chip_channel_to_position']:
                chip = chip_channel // 1000
                channel = chip_channel % 1000
                try:
                    io_group_io_channel = geo['tile_chip_to_io'][tile][chip]
                except KeyError:
                    print("Chip %i on tile %i not present in network" %
                          (chip, tile))
                    continue

                io_group = io_group_io_channel // 1000
                io_channel = io_group_io_channel % 1000
                x = chip_channel_to_position[chip_channel][0] * \
                    pixel_pitch + pixel_pitch / 2 - x_size / 2
                y = chip_channel_to_position[chip_channel][1] * \
                    pixel_pitch + pixel_pitch / 2 - y_size / 2

                x, y = _rotate_pixel((x, y), tile_orientation)
                x += tile_positions[tile][2] 
            
                y += tile_positions[tile][1] 
                    

                geometry['{}-{}-{}-{}'.format(io_group, io_group_io_channel_to_tile[(
                    io_group, io_channel)], chip, channel)] = x, y

        # Save dictionaries
        d = dict()
        d['geometry'] = geometry
        d['pixel_pitch'] = pixel_pitch

        with open(outfile_path, "w") as outfile:
            json.dump(d, outfile)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--geometry_yaml', default=_default_geometry_yaml, type=str,
                        help='''Multitile geometry yaml file''')
    parser.add_argument('--outfile_path', default=_default_outfile_path, type=str,
                        help='''Output JSON file''')
    args = parser.parse_args()

    main(**vars(args))
