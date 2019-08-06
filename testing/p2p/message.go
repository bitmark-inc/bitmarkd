package p2p

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

func NewTextMessage(m string) []byte {
	return []byte(m)
}

func NewU64Message(m uint64) []byte {
	v := make([]byte, 8)
	binary.BigEndian.PutUint64(v, m)
	return v
}

func NewHexMessage(m string) []byte {
	v, _ := hex.DecodeString(m)
	return v
}

func parseResponse(response [][]byte) string {
	var builder strings.Builder

	for i, r := range response {
		lenR := len(r)

		// segment no.
		builder.WriteString(fmt.Sprintf("%d. ", i+1))

		switch {
		case lenR == 8:
			u64 := binary.BigEndian.Uint64(r)
			builder.WriteString(fmt.Sprintf("%-8s| %08d", "u64", u64))
		case lenR <= 16 && isASCII(r):
			builder.WriteString(fmt.Sprintf("%-8s| %q", "text", r))
		case lenR == 32: // digest as big endian hex value
			reversed := make([]byte, lenR)
			for i, b := range r {
				reversed[lenR-i-1] = b
			}
			builder.WriteString(fmt.Sprintf("%-8s| %q", "digest", hex.EncodeToString(reversed)))
		default:
			builder.WriteString(fmt.Sprintf("%-8s| %q\n%s", "data", hex.EncodeToString(r), hex.Dump(r)))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

func isASCII(s []byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
