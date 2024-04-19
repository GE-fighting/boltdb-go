// PackageName: boltdb_go
package boltdb_go

// 错误类型定义及预设错误变量

// Error 是一个自定义错误类型，用于封装具体的错误消息字符串。
// 它实现了 Go 的 error 接口，可以通过调用其 Error() 方法返回错误消息。
type Error struct {
	message string
}

// Error 实现 error 接口，返回封装的错误消息字符串。
func (e Error) Error() string {
	return e.message
}

// 下列预设错误变量是 Error 类型的实例，分别代表不同的错误情况，
// 并在初始化时指定了对应的错误消息。这些变量可以在程序中直接引用，
// 以便快速、一致地报告特定的错误状态。

var (
	// KeyExistError 表示键值对已存在，通常在尝试插入重复键时触发。
	KeyExistError = &Error{"Key/Value pairs has existed"}

	// NotFoundError 表示未找到匹配的键值对，常见于查询不存在的键时。
	NotFoundError = &Error{"No matching key/value pair found"}

	// PageNotFoundError 表示请求的页面未找到，可能指示数据结构内部错误或请求资源不存在。
	PageNotFoundError = &Error{"Request page not found"}

	// CorruptedError 表示定位到的页面类型不正确，暗示数据结构损坏或文件格式异常。
	CorruptedError = &Error{"Located page was wrong type"}

	// PanicError 表示更新元数据页面失败，可能导致数据库状态不稳定或不可恢复。
	PanicError = &Error{"Update meta page failed"}

	// VersionMismatchError 表示数据库环境版本不匹配，可能需要升级或降级以兼容。
	VersionMismatchError = &Error{"Database environment Version mismatch"}

	// InvalidError 表示文件不是有效的 MDB 文件格式，无法被正确解析或操作。
	InvalidError = &Error{"File is not an MDB file"}

	// MapFullError 表示环境的最大映射尺寸（mapsize）已达到限制，无法继续增长。
	MapFullError = &Error{"Environment mapsize limit reached"}

	// BucketFullError 表示环境支持的最大数据库数量（maxdbs）已达上限。
	BucketFullError = &Error{"Environment maxdbs limit reached"}

	// ReadersFullError 表示环境支持的最大读取器数量已达上限。
	ReadersFullError = &Error{"Environment max-readers limit"}

	// TransactionFullError 表示事务包含过多脏页，可能超出允许的大小限制。
	TransactionFullError = &Error{"Transaction has too many dirty pages - transaction too big"}

	// CursorFullError 表示内部错误，游标堆栈已达到极限。
	CursorFullError = &Error{"Internal error - cursor stack limit reached"}

	// PageFullError 表示内部错误，当前页面已无可用空间。
	PageFullError = &Error{"Internal error - page has no more space"}

	// MapResizedError 表示数据库内容增长超出了当前环境的映射尺寸限制。
	MapResizedError = &Error{"Database contents grew beyond environment mapsize"}

	// InCompatibleError 表示操作与数据库不兼容，或者数据库标志（flags）发生了变化。
	InCompatibleError = &Error{"Operation and DB incompatible, or DB flags changed"}

	// BadReaderSlotError 表示非法重用了读取器锁表槽位，可能导致并发控制问题。
	BadReaderSlotError = &Error{"Invalid reuse of reader locktable slot"}

	// BadTransactionError 表示事务无法恢复，必须中止。
	BadTransactionError = &Error{"Transaction cannot recover, it must be aborted"}

	// BadValueSizeError 表示键值对过大、键为空或固定大小重复项（DUPFIXED）尺寸错误。
	BadValueSizeError = &Error{"Too big key/value, key is empty, or wrong DUPFIXED size"}
)
