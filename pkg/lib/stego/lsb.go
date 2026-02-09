package stego

import (
	"fmt"
	"image"
	"image/draw"
)

// LSBEmbed embeds a short message in an image using simple RGB least-significant bits.
// It returns a new RGBA image containing the embedded payload.
func LSBEmbed(src image.Image, message string) (image.Image, error) {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	// Store payload as [4-byte big-endian length][message bytes].
	payload := make([]byte, 4+len(message))
	n := len(message)
	payload[0] = byte(n >> 24)
	payload[1] = byte(n >> 16)
	payload[2] = byte(n >> 8)
	payload[3] = byte(n)
	copy(payload[4:], []byte(message))

	requiredBits := len(payload) * 8
	capacityBits := bounds.Dx() * bounds.Dy() * 3 // R, G, B channels only
	if requiredBits > capacityBits {
		return nil, fmt.Errorf("message too large for image capacity: need %d bits, have %d bits", requiredBits, capacityBits)
	}

	bitIndex := 0
	for y := bounds.Min.Y; y < bounds.Max.Y && bitIndex < requiredBits; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && bitIndex < requiredBits; x++ {
			off := dst.PixOffset(x, y)

			for channel := 0; channel < 3 && bitIndex < requiredBits; channel++ {
				byteIndex := bitIndex / 8
				shift := 7 - (bitIndex % 8)
				bit := (payload[byteIndex] >> shift) & 1

				dst.Pix[off+channel] = (dst.Pix[off+channel] & 0xFE) | bit
				bitIndex++
			}
		}
	}

	return dst, nil
}
