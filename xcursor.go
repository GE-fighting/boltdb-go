package boltdb_go

// xcursor 结构体代表了一个扩展游标，用于遍历Boltdb数据库中的特定范围数据。
type xcursor struct {
	// cursor 是基础游标对象，提供基本的游标功能。
	cursor cursor
	// bucket 指向当前游标所在的原始桶。
	bucket *bucket
	// bucketx 指向与当前游标关联的扩展桶。
	bucketx *bucketx
	// bucketFlag 存储与游标关联的桶标志位信息。
	bucketFlag int
}
