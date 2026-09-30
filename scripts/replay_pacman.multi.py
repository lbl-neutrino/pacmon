#!/usr/bin/env python3

import argparse
import time
import glob

import h5py
import numpy as np
import zmq
from tqdm import tqdm


def run(args: argparse.Namespace, ctx: zmq.Context):
    t_offset = 0

    socket = ctx.socket(zmq.XPUB)
    socket.setsockopt(zmq.XPUB_NODROP, 1)
    socket.sndhwm = 100000
    socket.bind(f'tcp://*:{args.port}')

    files = sorted(glob.glob('binary-*.h5'))
    for fname in tqdm(files):
        print(fname)
        with h5py.File(fname) as f:
            msgs = f['msgs']
            if args.start_seconds_ago and t_offset == 0:
                t0 = int(time.time()) - args.start_seconds_ago
                t_first_true = int.from_bytes(msgs[0][1:5], 'little')
                t_offset = t0 - t_first_true
                assert t_offset > 0
            for msg in tqdm(msgs):
                if t_offset:
                    t_orig = int.from_bytes(msg[1:5], 'little')
                    t_new = t_orig + t_offset
                    msg[1:5] = np.frombuffer(t_new.to_bytes(4, 'little'),
                                             dtype=np.uint8)

                socket.send(msg)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--port', '-p', default=6555)
    ap.add_argument('--start-seconds-ago', '-t', type=int,
                    help='If set, add a constant offset to all UNIX timestamps such that the first msg has a timestamp of start-seconds-ago before now')
    args = ap.parse_args()

    with zmq.Context() as ctx:
        run(args, ctx)


if __name__ == '__main__':
    main()
