#!/usr/bin/env python3

import argparse
import time

import h5py
import numpy as np
import zmq


def run(args: argparse.Namespace, ctx: zmq.Context, f: h5py.File):
    msgs = f['msgs']

    t_offset = 0
    if args.start_seconds_ago:
        t0 = int(time.time()) - args.start_seconds_ago
        t_first_true = int.from_bytes(msgs[0][1:5], 'little')
        t_offset = t0 - t_first_true
        assert t_offset > 0

    socket = ctx.socket(zmq.PUB)
    socket.bind(f'tcp://*:{args.port}')

    for msg in msgs:
        if t_offset:
            t_orig = int.from_bytes(msg[1:5], 'little')
            t_new = t_orig + t_offset
            msg[1:5] = np.frombuffer(t_new.to_bytes(4, 'little'),
                                     dtype=np.uint8)

        socket.send(msg)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('infile')
    ap.add_argument('--port', '-p', default=6555)
    ap.add_argument('--start-seconds-ago', '-t', type=int,
                    help='If set, add a constant offset to all UNIX timestamps such that the first msg has a timestamp of start-seconds-ago before now')
    args = ap.parse_args()

    with zmq.Context() as ctx:
        with h5py.File(args.infile, 'r') as f:
            run(args, ctx, f)


if __name__ == '__main__':
    main()