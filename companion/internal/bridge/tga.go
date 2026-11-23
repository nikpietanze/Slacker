package bridge

import "encoding/binary"

// GrayTGA returns an uncompressed 8-bit grayscale TGA of the given size with
// every pixel black. Content is irrelevant to the bridge; only the size is read.
func GrayTGA(w, h int) []byte {
	buf := make([]byte, 18+w*h)
	buf[2] = 3 // uncompressed grayscale
	binary.LittleEndian.PutUint16(buf[12:], uint16(w))
	binary.LittleEndian.PutUint16(buf[14:], uint16(h))
	buf[16] = 8 // bits per pixel
	return buf
}

// RGBATGA returns an uncompressed 32-bit TGA from top-down RGBA pixels.
func RGBATGA(w, h int, rgba []byte) []byte {
	buf := make([]byte, 18, 18+w*h*4)
	buf[2] = 2 // uncompressed true-color
	binary.LittleEndian.PutUint16(buf[12:], uint16(w))
	binary.LittleEndian.PutUint16(buf[14:], uint16(h))
	buf[16] = 32
	buf[17] = 0x28 // 8 alpha bits, top-left origin
	for i := 0; i < w*h; i++ {
		r, g, b, a := rgba[i*4], rgba[i*4+1], rgba[i*4+2], rgba[i*4+3]
		buf = append(buf, b, g, r, a) // TGA stores BGRA
	}
	return buf
}
