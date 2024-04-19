package boltdb_go

// meta 结构体表示Boltdb数据库的元数据。
type meta struct {
	// magic 用于验证文件是否为有效的Boltdb数据库文件。
	magic int32
	// version 表示数据库文件的版本信息。
	version int32
	// mapSize 指定数据库映射到内存的大小。
	mapSize int
	// free 存储空闲空间管理的相关信息。
	free *bucket
	// main 为数据库的主要桶，用于存储用户数据。
	main *bucket
	// lastPageNumber 记录数据库中最后分配的页面号。
	lastPageNumber int
	// transactionID 用于跟踪当前事务的状态。
	transactionID int
}
