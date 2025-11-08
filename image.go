package main

import (
	"encoding/binary"
	"fmt"

	wz "github.com/diamondo25/go-wz"
)

// Color lookup tables from the C++ implementation
var (
	table4 = [16]uint8{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
	}

	table5 = [32]uint8{
		0x00, 0x08, 0x10, 0x19, 0x21, 0x29, 0x31, 0x3A,
		0x42, 0x4A, 0x52, 0x5A, 0x63, 0x6B, 0x73, 0x7B,
		0x84, 0x8C, 0x94, 0x9C, 0xA5, 0xAD, 0xB5, 0xBD,
		0xC5, 0xCE, 0xD6, 0xDE, 0xE6, 0xEF, 0xF7, 0xFF,
	}

	table6 = [64]uint8{
		0x00, 0x04, 0x08, 0x0C, 0x10, 0x14, 0x18, 0x1C,
		0x20, 0x24, 0x28, 0x2D, 0x31, 0x35, 0x39, 0x3D,
		0x41, 0x45, 0x49, 0x4D, 0x51, 0x55, 0x59, 0x5D,
		0x61, 0x65, 0x69, 0x6D, 0x71, 0x75, 0x79, 0x7D,
		0x82, 0x86, 0x8A, 0x8E, 0x92, 0x96, 0x9A, 0x9E,
		0xA2, 0xA6, 0xAA, 0xAE, 0xB2, 0xB6, 0xBA, 0xBE,
		0xC2, 0xC6, 0xCA, 0xCE, 0xD2, 0xD7, 0xDB, 0xDF,
		0xE3, 0xE7, 0xEB, 0xEF, 0xF3, 0xF7, 0xFB, 0xFF,
	}
)

// Pixel represents an RGBA pixel
type Pixel struct {
	R, G, B, A uint8
}

// RGB565 represents a 16-bit RGB565 pixel
type RGB565 struct {
	data uint16
}

func (p RGB565) R() uint8 { return uint8((p.data >> 11) & 0x1F) }
func (p RGB565) G() uint8 { return uint8((p.data >> 5) & 0x3F) }
func (p RGB565) B() uint8 { return uint8(p.data & 0x1F) }

// ARGB4444 represents a 16-bit ARGB4444 pixel
type ARGB4444 struct {
	data uint16
}

func (p ARGB4444) A() uint8 { return uint8((p.data >> 12) & 0xF) }
func (p ARGB4444) R() uint8 { return uint8((p.data >> 8) & 0xF) }
func (p ARGB4444) G() uint8 { return uint8((p.data >> 4) & 0xF) }
func (p ARGB4444) B() uint8 { return uint8(p.data & 0xF) }

// processCanvasData converts WZ canvas data to RGBA format
func processCanvasData(canvas *wz.WZCanvas, data []byte) ([]byte, error) {
	width := int(canvas.Width)
	height := int(canvas.Height)
	format := canvas.Format

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid canvas dimensions: %dx%d", width, height)
	}

	pixels := width * height
	output := make([]byte, pixels*4) // RGBA

	switch format {
	case 1: // ARGB4444
		return convertARGB4444(data, width, height)

	case 2: // ARGB8888
		return convertARGB8888(data, width, height)

	case 513: // RGB565
		return convertRGB565(data, width, height)

	case 1026: // DXT3
		// DXT3 decompression would go here
		// For now, return empty data or the raw data
		return make([]byte, pixels*4), nil

	case 2050: // DXT5
		// DXT5 decompression would go here
		// For now, return empty data or the raw data
		return make([]byte, pixels*4), nil

	default:
		// Unknown format, return empty RGBA
		return output, nil
	}
}

// convertARGB4444 converts ARGB4444 format to RGBA
func convertARGB4444(data []byte, width, height int) ([]byte, error) {
	pixels := width * height
	output := make([]byte, pixels*4)

	for i := 0; i < pixels && i*2+1 < len(data); i++ {
		pixel := ARGB4444{binary.LittleEndian.Uint16(data[i*2:])}
		output[i*4+0] = table4[pixel.R()] // R
		output[i*4+1] = table4[pixel.G()] // G
		output[i*4+2] = table4[pixel.B()] // B
		output[i*4+3] = table4[pixel.A()] // A
	}

	return output, nil
}

// convertARGB8888 converts ARGB8888 format to RGBA
func convertARGB8888(data []byte, width, height int) ([]byte, error) {
	pixels := width * height
	output := make([]byte, pixels*4)

	for i := 0; i < pixels && i*4+3 < len(data); i++ {
		// WZ stores as BGRA, we need RGBA
		output[i*4+0] = data[i*4+2] // R
		output[i*4+1] = data[i*4+1] // G
		output[i*4+2] = data[i*4+0] // B
		output[i*4+3] = data[i*4+3] // A
	}

	return output, nil
}

// convertRGB565 converts RGB565 format to RGBA
func convertRGB565(data []byte, width, height int) ([]byte, error) {
	pixels := width * height
	output := make([]byte, pixels*4)

	for i := 0; i < pixels && i*2+1 < len(data); i++ {
		pixel := RGB565{binary.LittleEndian.Uint16(data[i*2:])}
		output[i*4+0] = table5[pixel.R()] // R
		output[i*4+1] = table6[pixel.G()] // G
		output[i*4+2] = table5[pixel.B()] // B
		output[i*4+3] = 255               // A (fully opaque)
	}

	return output, nil
}

// decompressWZData decompresses the zlib-compressed WZ data
func decompressWZData(compressed []byte) ([]byte, error) {
	// WZ data is typically zlib compressed
	// The go-wz library should handle this, but if we need to do it manually:
	// Use compress/zlib or compress/flate
	// For now, assume data is already decompressed by go-wz
	return compressed, nil
}
