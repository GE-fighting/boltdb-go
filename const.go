package boltdb_go

const Version = 1
const Magic = 0x83293

const (
	MaxKeySize  = 511
	MaxDataSize = 0xffffffff
)
const (
	DefaultMapSize     = 1 << 20
	DefaultReaderCount = 126
)
