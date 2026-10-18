package main

// #cgo pkg-config: hdf5
// #include <stdlib.h>
// #include <hdf5.h>
//
// static hid_t vlenU8Type(void) { return H5Tvlen_create(H5T_NATIVE_UINT8); }
// static hid_t fileAccRdonly(void) { return H5F_ACC_RDONLY; }
// static hid_t defaultPlist(void) { return H5P_DEFAULT; }
// static hid_t selectSet(void) { return H5S_SELECT_SET; }
import "C"

import (
	"fmt"
	"unsafe"
)

// MsgReader reads the variable-length uint8 "msgs" dataset of a PACMAN
// binary HDF5 file, in blocks to bound memory usage.
type MsgReader struct {
	file, dset, space, memType C.hid_t
	Len                        int
}

func OpenMsgs(fname string) (*MsgReader, error) {
	cname := C.CString(fname)
	defer C.free(unsafe.Pointer(cname))
	cdset := C.CString("msgs")
	defer C.free(unsafe.Pointer(cdset))

	r := &MsgReader{file: -1, dset: -1, space: -1, memType: -1}

	r.file = C.H5Fopen(cname, C.uint(C.fileAccRdonly()), C.defaultPlist())
	if r.file < 0 {
		return nil, fmt.Errorf("failed to open %s", fname)
	}
	r.dset = C.H5Dopen2(r.file, cdset, C.defaultPlist())
	if r.dset < 0 {
		r.Close()
		return nil, fmt.Errorf("failed to open msgs dataset in %s", fname)
	}
	r.space = C.H5Dget_space(r.dset)
	if r.space < 0 || C.H5Sget_simple_extent_ndims(r.space) != 1 {
		r.Close()
		return nil, fmt.Errorf("msgs dataset in %s is not 1-dimensional", fname)
	}
	var dim C.hsize_t
	C.H5Sget_simple_extent_dims(r.space, &dim, nil)
	r.Len = int(dim)
	r.memType = C.vlenU8Type()

	return r, nil
}

// ReadBlock calls fn on each message in [start, start+count). The slice
// passed to fn is only valid for the duration of the call.
func (r *MsgReader) ReadBlock(start, count int, fn func([]byte) error) error {
	cstart := C.hsize_t(start)
	ccount := C.hsize_t(count)
	if C.H5Sselect_hyperslab(r.space, C.H5S_seloper_t(C.selectSet()),
		&cstart, nil, &ccount, nil) < 0 {
		return fmt.Errorf("failed to select hyperslab")
	}
	memSpace := C.H5Screate_simple(1, &ccount, nil)
	defer C.H5Sclose(memSpace)

	buf := make([]C.hvl_t, count)
	if C.H5Dread(r.dset, r.memType, memSpace, r.space, C.defaultPlist(),
		unsafe.Pointer(&buf[0])) < 0 {
		return fmt.Errorf("failed to read msgs [%d, %d)", start, start+count)
	}
	defer C.H5Treclaim(r.memType, memSpace, C.defaultPlist(),
		unsafe.Pointer(&buf[0]))

	for _, vl := range buf {
		msg := unsafe.Slice((*byte)(vl.p), int(vl.len))
		if err := fn(msg); err != nil {
			return err
		}
	}
	return nil
}

func (r *MsgReader) Close() {
	if r.memType >= 0 {
		C.H5Tclose(r.memType)
	}
	if r.space >= 0 {
		C.H5Sclose(r.space)
	}
	if r.dset >= 0 {
		C.H5Dclose(r.dset)
	}
	if r.file >= 0 {
		C.H5Fclose(r.file)
	}
}
