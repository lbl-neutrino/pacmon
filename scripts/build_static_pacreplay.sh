#!/usr/bin/env bash
# Build a fully static pacreplay binary.
#
# Distro packages generally don't ship static libhdf5/libzmq (and the system
# builds drag in szip, gnutls, libsodium, pgm, ...), so we build minimal static
# versions of zlib, HDF5, and libzmq into a local prefix and link against them.
#
# Usage: scripts/build_static_pacreplay.sh [output]   (default: ./pacreplay)

set -euo pipefail

ZLIB_VERSION=1.3.1
HDF5_VERSION=1.14.6
ZMQ_VERSION=4.3.5

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT=$(realpath -m "${1:-$ROOT/pacreplay}")
BUILD=${STATIC_BUILD_DIR:-$ROOT/.static-build}
PREFIX=$BUILD/prefix
JOBS=$(nproc)

mkdir -p "$BUILD/src" "$PREFIX/lib/pkgconfig"
cd "$BUILD/src"

fetch() {
    local url=$1 tarball=${1##*/}
    [[ -f $tarball ]] || curl -fL -o "$tarball" "$url"
    tar xf "$tarball"
}

if [[ ! -f $PREFIX/lib/libz.a ]]; then
    fetch https://zlib.net/fossils/zlib-$ZLIB_VERSION.tar.gz
    (cd zlib-$ZLIB_VERSION &&
        CFLAGS="-O2 -fPIC" ./configure --static --prefix="$PREFIX" &&
        make -j"$JOBS" && make install)
fi

if [[ ! -f $PREFIX/lib/libhdf5.a ]]; then
    fetch https://github.com/HDFGroup/hdf5/releases/download/hdf5_$HDF5_VERSION/hdf5-$HDF5_VERSION.tar.gz
    cmake -S hdf5-$HDF5_VERSION -B hdf5-build \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX="$PREFIX" \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DBUILD_SHARED_LIBS=OFF \
        -DBUILD_STATIC_LIBS=ON \
        -DBUILD_TESTING=OFF \
        -DHDF5_BUILD_TOOLS=OFF \
        -DHDF5_BUILD_UTILS=OFF \
        -DHDF5_BUILD_EXAMPLES=OFF \
        -DHDF5_BUILD_CPP_LIB=OFF \
        -DHDF5_BUILD_FORTRAN=OFF \
        -DHDF5_BUILD_JAVA=OFF \
        -DHDF5_BUILD_HL_LIB=OFF \
        -DHDF5_ENABLE_SZIP_SUPPORT=OFF \
        -DHDF5_ENABLE_Z_LIB_SUPPORT=ON \
        -DZLIB_USE_EXTERNAL=OFF \
        -DZLIB_INCLUDE_DIR="$PREFIX/include" \
        -DZLIB_LIBRARY="$PREFIX/lib/libz.a"
    cmake --build hdf5-build -j"$JOBS"
    cmake --install hdf5-build
fi

if [[ ! -f $PREFIX/lib/libzmq.a ]]; then
    fetch https://github.com/zeromq/libzmq/releases/download/v$ZMQ_VERSION/zeromq-$ZMQ_VERSION.tar.gz
    cmake -S zeromq-$ZMQ_VERSION -B zmq-build \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX="$PREFIX" \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
        -DBUILD_SHARED=OFF \
        -DBUILD_STATIC=ON \
        -DBUILD_TESTS=OFF \
        -DWITH_DOCS=OFF \
        -DWITH_PERF_TOOL=OFF \
        -DWITH_LIBSODIUM=OFF \
        -DENABLE_CURVE=OFF \
        -DWITH_TLS=OFF \
        -DWITH_LIBBSD=OFF \
        -DENABLE_DRAFTS=OFF
    cmake --build zmq-build -j"$JOBS"
    cmake --install zmq-build
fi

# Write our own .pc files so the cgo `pkg-config:` directives resolve to the
# static archives with the right transitive deps.
cat > "$PREFIX/lib/pkgconfig/hdf5.pc" <<EOF
Name: hdf5
Description: HDF5 (static, local build)
Version: $HDF5_VERSION
Cflags: -I$PREFIX/include
Libs: $PREFIX/lib/libhdf5.a $PREFIX/lib/libz.a -lm -ldl -lpthread
EOF
cat > "$PREFIX/lib/pkgconfig/libzmq.pc" <<EOF
Name: libzmq
Description: ZeroMQ (static, local build)
Version: $ZMQ_VERSION
Cflags: -I$PREFIX/include -DZMQ_STATIC
Libs: $PREFIX/lib/libzmq.a -lstdc++ -lm -lpthread -lrt
EOF

cd "$ROOT"
# netgo/osusergo keep Go's own resolver/user lookup from calling into glibc NSS,
# which doesn't work properly from a static binary.
PKG_CONFIG_PATH=$PREFIX/lib/pkgconfig PKG_CONFIG_LIBDIR=$PREFIX/lib/pkgconfig \
CGO_ENABLED=1 \
    go build -trimpath -tags netgo,osusergo \
        -ldflags '-s -w -linkmode external -extldflags "-static"' \
        -o "$OUT" ./cmd/pacreplay

file "$OUT"
