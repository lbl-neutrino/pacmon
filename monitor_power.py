import logging
import json
from base.utility_base import now
from base import pacman_base
import time
import argparse
from RUNENV import *
import larpix
import larpix.io

import datetime as dt

import matplotlib.pyplot as plt
import matplotlib.animation as animation


import influxdb_client
from influxdb_client.client.write_api import SYNCHRONOUS

# urllib3 has functions to help with url timeout problems
import urllib3


import warnings
warnings.filterwarnings("ignore")

_default_verbose = False
skip_readback = False


def main(verbose, pacman_config):

    pacman_configs = {}
    with open(pacman_config, 'r') as f:
        pacman_configs = json.load(f)

    c = larpix.Controller()
    c.io = larpix.io.PACMAN_IO(relaxed=True, config_filepath=pacman_config)

    print(pacman_configs)
    # for each io_group, perform networking

    while True:
        for io_group_ip_pair in pacman_configs['io_group']:
            io_group = io_group_ip_pair[0]
            print('Configuring IO Group {}'.format(io_group))

            # read pacman information and metadata from RUNENV
            pacman_version = iog_pacman_version_[io_group]

            pacman_version = iog_pacman_version_[io_group]

            VDDA_REG = None
            VDDD_REG = None

            readback = pacman_base.power_readback(
                c.io, io_group, pacman_version, io_group_pacman_tile_[io_group])

        time.sleep(10)

    return


def power_readback(io, io_group, pacman_version, tile):
    readback = {}
    for i in tile:
        readback[i] = []
        if pacman_version == 'v1rev4':
            vdda = io.get_reg(0x24030+(i-1), io_group=io_group)
            vddd = io.get_reg(0x24040+(i-1), io_group=io_group)
            idda = io.get_reg(0x24050+(i-1), io_group=io_group)
            iddd = io.get_reg(0x24060+(i-1), io_group=io_group)
            print('Tile ', i, '  VDDA: ', vdda, ' mV  IDDA: ', idda*0.1, ' mA  ',
                  'VDDD: ', vddd, ' mV  IDDD: ', abs(iddd >> 12), ' mA')
            readback[i] = [vdda, idda*0.1, vddd, iddd >> 12]
        elif pacman_version == 'v1rev3' or 'v1revS1':
            vdda = io.get_reg(0x00024001+(i-1)*32+1, io_group=io_group)
            idda = io.get_reg(0x00024001+(i-1)*32, io_group=io_group)
            vddd = io.get_reg(0x00024001+(i-1)*32+17, io_group=io_group)
            iddd = io.get_reg(0x00024001+(i-1)*32+16, io_group=io_group)
            print('Tile ', i, '  VDDA: ', (((vdda >> 16) >> 3)*4), ' mV  IDDA: ',
                  (((idda >> 16)-(idda >> 31)*65535)*500*0.001), ' mV  VDDD: ',
                  (((vddd >> 16) >> 3)*4), ' mV  IDDD: ',
                  (((iddd >> 16)-(iddd >> 31)*65535)*500*0.001), ' mA')
            readback[i] = [(((vdda >> 16) >> 3)*4),
                           (((idda >> 16)-(idda >> 31)*65535)*500*0.001),
                           (((vddd >> 16) >> 3)*4),
                           (((iddd >> 16)-(iddd >> 31)*65535)*500*0.001)]
        else:
            print('WARNING: PACMAN version ', pacman_version, ' unknown')
            return readback
    return readback


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--pacman_config', default="io/pacman.json",
                        type=str, help='''Config specifying PACMANs''')
    parser.add_argument('--verbose', '-v', action='store_true',
                        default=_default_verbose)
    args = parser.parse_args()
    c = main(**vars(args))
# Define LArPix amd PACMAN controllers
ctl = larpix.Controller()
ctl.io = larpix.io.PACMAN_IO(relaxed=True)

tile = 1
io_group = 1

# Get VDDD, IDDD, VDDA, IDDA data
Vddd = []
Vdda = []
Iddd = []
Idda = []
time = []

fig = plt.figure()
ax = fig.add_subplot(1, 1, 1)


def animate(i, time, Vddd, Vdda, Iddd, Idda):

    now = dt.datetime.now().strftime('%H:%M:%S.%f')

    vdda = ctl.io.get_reg(0x00024001 + (tile - 1) * 32 + 1, io_group=io_group)
    idda = ctl.io.get_reg(0x00024001 + (tile - 1) * 32, io_group=io_group)
    vddd = ctl.io.get_reg(0x00024001 + (tile - 1) * 32 + 17, io_group=io_group)
    iddd = ctl.io.get_reg(0x00024001 + (tile - 1) * 32 + 16, io_group=io_group)

    Vddd.append(((vddd >> 16) >> 3)*4)
    Vdda.append(((vdda >> 16) >> 3)*4)
    Iddd.append(((iddd >> 16)-(iddd >> 31)*65535)*500*0.001)
    Idda.append(((idda >> 16)-(idda >> 31)*65535)*500*0.001)
    time.append(now)

    if (len(Vddd) > 20):
        Vddd = Vddd[-20:]
        Vdda = Vdda[-20:]
        Iddd = Iddd[-20:]
        Idda = Idda[-20:]
        time = time[-20:]

    ax.clear()
    ax.plot(time, Vddd)
    ax.plot(time, Vdda)
    plt.xticks(rotation=45, ha='right')
    plt.subplots_adjust(bottom=0.30)


ani = animation.FuncAnimation(fig, animate, fargs=(
    time, Vddd, Vdda, Iddd, Idda), interval=1000)
plt.show()
