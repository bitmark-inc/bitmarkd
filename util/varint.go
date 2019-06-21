// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

// Varint64MaximumBytes - maximum possible number of bytes in Varint64
const Varint64MaximumBytes = 9

// ToVarint64 - convert a 64 bit unsigned integer to Varint64
//
// Structure of the result
// byte 1:  ext | B06 | B05 | B04 | B03 | B02 | B01 | B00
// byte 2:  ext | B13 | B12 | B11 | B10 | B09 | B08 | B07
// byte 3:  ext | B20 | B19 | B18 | B17 | B16 | B15 | B14
// byte 4:  ext | B27 | B26 | B25 | B24 | B23 | B22 | B21
// byte 5:  ext | B34 | B33 | B32 | B31 | B30 | B29 | B28
// byte 6:  ext | B41 | B40 | B39 | B38 | B37 | B36 | B35
// byte 7:  ext | B48 | B47 | B46 | B45 | B44 | B43 | B42
// byte 8:  ext | B55 | B54 | B53 | B52 | B51 | B50 | B49
// byte 9:  B63 | B62 | B61 | B60 | B59 | B58 | B57 | B56
func ToVarint64(value uint64) []byte {
	result := make([]byte, 0, Varint64MaximumBytes)
	if value < 0x80 {
		result = append(result, byte(value))
		return result
	}

	for i := 0; i < Varint64MaximumBytes && value != 0; i += 1 {
		ext := uint64(0x80)
		if value < 0x80 {
			ext = 0x00
		}
		result = append(result, byte(value|ext))
		value >>= 7
	}
	return result
}

// FromVarint64 - convert an array of up to Varint64MaximumBytes to a uint64
//
// also return the number of bytes used as second value
// returns 0, 0 if varint64 buffer is truncated
func FromVarint64(buffer []byte) (uint64, int) {
	result := uint64(0)

	shift := uint(0)
	count := 0

	for count < len(buffer) {
		currByte := uint64(buffer[count])
		count += 1
		if count < Varint64MaximumBytes {
			result |= currByte & 0x7f << shift
			if 0 == currByte&0x80 {
				return result, count
			}
		} else {
			result |= currByte << shift
			return result, count
		}
		shift += 7
	}
	return 0, 0
}

// CopyVarint64 - make a copy of a Varint64 from the beginning of a buffer
func CopyVarint64(buffer []byte) []byte {
	result := make([]byte, 0)

loop:
	for count := 0; count < Varint64MaximumBytes; count += 1 {
		currentByte := buffer[count]
		result = append(result, currentByte)
		if 0 == currentByte&0x80 {
			break loop
		}
	}
	return result
}

// ClippedVarint64 - return a positive clipped value as an int
// any value outside the range minimum..maximum is an error
func ClippedVarint64(buffer []byte, minimum int, maximum int) (int, int) {
	if minimum < 0 || maximum < 0 || minimum >= maximum {
		return 0, 0
	}

	value, count := FromVarint64(buffer)
	if 0 == count {
		return 0, 0
	}
	iValue := int(value)
	if iValue < minimum || iValue > maximum {
		return 0, 0
	}
	return iValue, count
}
