package config

// CompressionCfg
//   - Supported levels:
//     CompressNoCompression      = 0
//     CompressBestSpeed          = 1
//     CompressBestCompression    = 9
//     CompressDefaultCompression = 6  // flate.DefaultCompression
//     CompressHuffmanOnly        = -2 // flate.HuffmanOnly
type CompressionCfg struct {
	Level int `yaml:"level"`
}

func (cfg *CompressionCfg) Enabled() bool {
	return cfg != nil
}
