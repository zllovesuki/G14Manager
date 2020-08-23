package util

// Uint32ToLEBuffer will convert an uint32 into 4 bytes slice in little endian
func Uint32ToLEBuffer(input uint32) []byte {
	buf := make([]byte, 4)
	buf[0] = byte((input & 0x000000ff) >> 0)
	buf[1] = byte((input & 0x0000ff00) >> 8)
	buf[2] = byte((input & 0x00ff0000) >> 16)
	buf[3] = byte((input & 0xff000000) >> 24)
	return buf
}
