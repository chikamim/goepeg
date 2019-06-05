package goepeg

import (
	"fmt"
	"unsafe"
)

/*
#cgo linux LDFLAGS: ${SRCDIR}/libepeg_linux_amd64.a ${SRCDIR}/libjpeg_linux_amd64.a
#cgo darwin LDFLAGS: ${SRCDIR}/libepeg_darwin_amd64.a ${SRCDIR}/libjpeg_darwin_amd64.a
#include <stdlib.h>
#include "Epeg.h"
*/
import "C"

type TransformType int

const (
	TransformNone       TransformType = iota
	TransformFlipH                    = iota
	TransformFlipV                    = iota
	TransformTranspose                = iota
	TransformTransverse               = iota
	TransformRot90                    = iota
	TransformRot180                   = iota
	TransformRot270                   = iota
)

type ScaleType int

const (
	ScaleTypeFitMax ScaleType = iota
	ScaleTypeFitMin           = iota
)

type Result struct {
	p    *C.uchar
	size C.int
}

func (r Result) Release() {
	C.free(unsafe.Pointer(r.p))
}

func (r Result) Bytes() []byte {
	return C.GoBytes(unsafe.Pointer(r.p), r.size)
}

// Thumbnail returns resized jpeg bytes from input jpeg bytes and max width, max height and quality settings.
func Thumbnail(b []byte, width, height, quality int) ([]byte, error) {
	var img *C.Epeg_Image

	p := unsafe.Pointer(C.CString(string(b)))
	defer C.free(p)
	img = C.epeg_memory_open((*C.uchar)(p), C.int(len(b)))

	if img == nil {
		return nil, fmt.Errorf("Epeg could not decode input image")
	}
	defer C.epeg_close(img)

	var cw C.int
	var ch C.int
	C.epeg_size_get(img, &cw, &ch)
	w, h := imageSize(int(cw), int(ch), width, height)

	C.epeg_decode_size_set(img, C.int(w), C.int(h))
	C.epeg_quality_set(img, C.int(quality))

	res := Result{}
	defer res.Release()
	C.epeg_memory_output_set(img, &res.p, &res.size)

	if C.epeg_encode(img) != 0 {
		return nil, fmt.Errorf("Epeg encode error")
	}
	return res.Bytes(), nil
}

func imageSize(imageWidth, imageHeight, maxWidth, maxHeight int) (width, height int) {
	if imageWidth > imageHeight {
		width = maxWidth
		height = int(float64(maxHeight) * float64(imageHeight) / float64(imageWidth))
	} else {
		width = int(float64(maxWidth) * float64(imageWidth) / float64(imageHeight))
		height = maxHeight
	}
	return
}

func Transform(input string, output string, transform TransformType) error {
	var trans int

	switch transform {
	case TransformNone:
		trans = C.EPEG_TRANSFORM_NONE
	case TransformFlipH:
		trans = C.EPEG_TRANSFORM_FLIP_H
	case TransformFlipV:
		trans = C.EPEG_TRANSFORM_FLIP_V
	case TransformTranspose:
		trans = C.EPEG_TRANSFORM_TRANSPOSE
	case TransformTransverse:
		trans = C.EPEG_TRANSFORM_TRANSVERSE
	case TransformRot90:
		trans = C.EPEG_TRANSFORM_ROT_90
	case TransformRot180:
		trans = C.EPEG_TRANSFORM_ROT_180
	case TransformRot270:
		trans = C.EPEG_TRANSFORM_ROT_270
	default:
		return fmt.Errorf("Epeg invalid transformation")
	}

	inputCString := C.CString(input)
	defer C.free(unsafe.Pointer(inputCString))

	outputCString := C.CString(output)
	defer C.free(unsafe.Pointer(outputCString))

	var img *C.Epeg_Image

	img = C.epeg_file_open(inputCString)

	if img == nil {
		return fmt.Errorf("Epeg could not open image %s", input)
	}

	defer C.epeg_close(img)

	C.epeg_transform_set(img, C.Epeg_Transform(trans))

	C.epeg_file_output_set(img, outputCString)

	if code := int(C.epeg_transform(img)); code != 0 {
		buf := [1024]byte{}
		C.epeg_error(img, (*C.char)((unsafe.Pointer(&buf[0]))))
		return fmt.Errorf("Epeg transform error: error %d: %s", code, buf)
	}

	return nil
}
