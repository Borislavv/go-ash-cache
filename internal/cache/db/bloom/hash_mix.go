package bloom

// nextPow2 returns the smallest power-of-two >= x (for int).
func nextPow2(x int) int {
	if x <= 1 {
		return 1
	}
	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	// For 64-bit ints, add: x |= x >> 32
	return x + 1
}

// mix64 produces well-diffused pseudo-independent values from a single 64-bit seed.
// This is the SplitMix64 mixing function (public-domain; Steele et al.).
// Constants below are from the SplitMix64 reference; we name them for readability.
func mix64(x uint64) uint64 {
	const (
		// Golden ratio increment used by SplitMix64 to traverse states uniformly.
		splitmix64Increment = 0x9E3779B97F4A7C15

		// Multipliers from SplitMix64 reference implementation.
		splitmix64Mul1 = 0xBF58476D1CE4E5B9
		splitmix64Mul2 = 0x94D049BB133111EB
	)
	x += splitmix64Increment
	x = (x ^ (x >> 30)) * splitmix64Mul1
	x = (x ^ (x >> 27)) * splitmix64Mul2
	return x ^ (x >> 31)
}
