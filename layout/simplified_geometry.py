import yaml
import numpy as np
import argparse
import json
from collections import defaultdict

_default_geometry_yaml = 'multi_tile_layout-2.3.16.yaml'
_default_geometry_yaml_mod2 = 'multi_tile_layout-2.5.16.yaml'


def _default_pxy():
    return (0., 0.)


def _rotate_pixel(pixel_pos, tile_orientation):
    return pixel_pos[0]*tile_orientation[2], pixel_pos[1]*tile_orientation[1]


def main(geometry_yaml=_default_geometry_yaml,
         geometry_yaml_mod2=_default_geometry_yaml_mod2):

    with open(geometry_yaml) as fi:
        geo = yaml.full_load(fi)

    with open(geometry_yaml_mod2) as fi2:
        geo_mod2 = yaml.full_load(fi2)

    if 'multitile_layout_version' in geo.keys():
        # Adapted from: https://github.com/larpix/larpix-v2-testing-scripts/blob/master/event-display/evd_lib.py

        # Module 1, 3, 4 layout
        pixel_pitch = geo['pixel_pitch']

        chip_channel_to_position = geo['chip_channel_to_position']
        tile_orientations = geo['tile_orientations']
        tile_positions = geo['tile_positions']
        tpc_centers = geo['tpc_centers']
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
                x += tile_positions[tile][2] + \
                    tpc_centers[tile_indeces[tile][0]][0]
                y += tile_positions[tile][1] + \
                    tpc_centers[tile_indeces[tile][0]][1]

                geometry['{}-{}-{}-{}'.format(io_group, io_group_io_channel_to_tile[(
                    io_group, io_channel)], chip, channel)] = x, y

        xmin = min(np.array(list(geometry.values()))[:, 0])-pixel_pitch/2
        xmax = max(np.array(list(geometry.values()))[:, 0])+pixel_pitch/2
        ymin = min(np.array(list(geometry.values()))[:, 1])-pixel_pitch/2
        ymax = max(np.array(list(geometry.values()))[:, 1])+pixel_pitch/2

        tile_vertical_lines = np.linspace(xmin, xmax, 3)
        tile_horizontal_lines = np.linspace(ymin, ymax, 5)
        chip_vertical_lines = np.linspace(xmin, xmax, 21)
        chip_horizontal_lines = np.linspace(ymin, ymax, 41)

        nonrouted_v2a_channels = [6, 7, 8, 9, 22,
                                  23, 24, 25, 38, 39, 40, 54, 55, 56, 57]
        routed_v2a_channels = [i for i in range(
            64) if i not in nonrouted_v2a_channels]

        # Module 2 layout
        pixel_pitch_mod2 = geo_mod2['pixel_pitch']

        chip_channel_to_position_mod2 = geo_mod2['chip_channel_to_position']
        tile_orientations_mod2 = geo_mod2['tile_orientations']
        tile_positions_mod2 = geo_mod2['tile_positions']
        tpc_centers_mod2 = geo['tpc_centers']
        tile_indeces_mod2 = geo_mod2['tile_indeces']
        xs_mod2 = np.array(list(chip_channel_to_position_mod2.values()))[
            :, 0] * pixel_pitch_mod2
        ys_mod2 = np.array(list(chip_channel_to_position_mod2.values()))[
            :, 1] * pixel_pitch_mod2
        x_size_mod2 = max(xs_mod2)-min(xs_mod2)+pixel_pitch_mod2
        y_size_mod2 = max(ys_mod2)-min(ys_mod2)+pixel_pitch_mod2

        tile_geometry_mod2 = defaultdict(int)
        io_group_io_channel_to_tile_mod2 = {}
        geometry_mod2 = defaultdict(_default_pxy)

        for tile in geo_mod2['tile_chip_to_io']:
            tile_orientation_mod2 = tile_orientations_mod2[tile]
            tile_geometry_mod2[tile] = tile_positions_mod2[tile], tile_orientations_mod2[tile]
            for chip in geo_mod2['tile_chip_to_io'][tile]:
                io_group_io_channel = geo_mod2['tile_chip_to_io'][tile][chip]
                io_group = io_group_io_channel//1000
                io_channel = io_group_io_channel % 1000
                io_group_io_channel_to_tile_mod2[(
                    io_group, io_channel)] = tile

            for chip_channel in geo_mod2['chip_channel_to_position']:
                chip = chip_channel // 1000
                channel = chip_channel % 1000
                try:
                    io_group_io_channel = geo_mod2['tile_chip_to_io'][tile][chip]
                except KeyError:
                    print("Chip %i on tile %i not present in Module 2 network" %
                          (chip, tile))
                    continue

                io_group = io_group_io_channel // 1000
                io_channel = io_group_io_channel % 1000
                x = chip_channel_to_position_mod2[chip_channel][0] * \
                    pixel_pitch_mod2 + pixel_pitch_mod2 / 2 - x_size_mod2 / 2
                y = chip_channel_to_position_mod2[chip_channel][1] * \
                    pixel_pitch_mod2 + pixel_pitch_mod2 / 2 - y_size_mod2 / 2

                x, y = _rotate_pixel((x, y), tile_orientation_mod2)
                x += tile_positions_mod2[tile][2] + \
                    tpc_centers_mod2[tile_indeces_mod2[tile][0]][0]
                y += tile_positions_mod2[tile][1] + \
                    tpc_centers_mod2[tile_indeces_mod2[tile][0]][1]

                geometry_mod2['{}-{}-{}-{}'.format(io_group, io_group_io_channel_to_tile_mod2[(
                    io_group, io_channel)], chip, channel)] = x, y

        # Save dictionaries
        geometry['pixel_pitch'] = pixel_pitch
        geometry_mod2['pixel_pitch'] = pixel_pitch_mod2

        with open("geometry_mod013.json", "w") as outfile:
            json.dump(geometry, outfile)
        with open("geometry_mod2.json", "w") as outfile_mod2:
            json.dump(geometry_mod2, outfile_mod2)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--geometry_yaml', default=_default_geometry_yaml, type=str,
                        help='''geometry yaml file (layout 2.4.0 for LArPix-v2a 10x10 tile)''')
    parser.add_argument('--geometry_yaml_mod2', default=_default_geometry_yaml_mod2, type=str,
                        help='''geometry yaml file for Module 2''')
    args = parser.parse_args()

    main(**vars(args))
