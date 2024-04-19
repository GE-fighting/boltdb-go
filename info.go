package boltdb_go

type info struct {
	MapSize           int
	LastPageID        int
	LastTransactionID int
	MaxReaders        int
	ReaderCount       int
}
