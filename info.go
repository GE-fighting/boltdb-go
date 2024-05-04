package boltdb_go

type Info struct {
	MapSize           int
	LastPageID        int
	LastTransactionID int
	MaxReaders        int
	ReaderCount       int
}
