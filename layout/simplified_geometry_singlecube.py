import yaml
import json


pitch = 4.4

geometrypath = 'layout-2.4.0.yaml'
with open(geometrypath) as fi:
    geo = yaml.full_load(fi)

chip_pix = dict([(chip_id, pix) for chip_id, pix in geo['chips']])

N_CHIPS = 100
N_CHANNELS = 64


def chip_channel_to_index(chip, channel):
    return (chip-11)*N_CHANNELS + channel


simple_geo = dict()
for chip in range(11, 111):
    for channel in range(64):
        pix = chip_pix[chip][channel]
        counter = chip_channel_to_index(chip, channel)
        if not pix is None:
            simple_geo['{}-{}-{}-{}'.format(1, 1, chip, channel)
                       ] = geo['pixels'][pix][1], geo['pixels'][pix][2]


d = dict()
d['geometry'] = simple_geo
d['pixel_pitch'] = pitch

with open("geometry_singlecube.json", "w") as outfile:
    json.dump(d, outfile)
