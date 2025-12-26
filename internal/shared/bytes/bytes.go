package bytes

import (
	"bytes"
	"fmt"
	"github.com/zeebo/xxh3"
)

func IsBytesAreEquals(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) < 32 {
		return bytes.Equal(a, b)
	}

	ha := xxh3.Hash(a[:8]) ^ xxh3.Hash(a[len(a)/2:len(a)/2+8]) ^ xxh3.Hash(a[len(a)-8:])
	hb := xxh3.Hash(b[:8]) ^ xxh3.Hash(b[len(b)/2:len(b)/2+8]) ^ xxh3.Hash(b[len(b)-8:])
	return ha == hb
}

func FmtMem(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		t := bytes / TB
		rem := bytes % TB
		return fmt.Sprintf("%dTB %dGB", t, rem/GB)
	case bytes >= GB:
		g := bytes / GB
		rem := bytes % GB
		return fmt.Sprintf("%dGB %dMB", g, rem/MB)
	case bytes >= MB:
		m := bytes / MB
		rem := bytes % MB
		return fmt.Sprintf("%dMB %dKB", m, rem/KB)
	case bytes >= KB:
		k := bytes / KB
		return fmt.Sprintf("%dKB %dB", k, bytes%KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
