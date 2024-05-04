package boltdb_go

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// 定义DB相关的选项常量。
const (
	// NoSync 表示禁用数据库同步操作。
	NoSync = iota
	// NoMetaSync 表示仅同步数据库数据，而不同步元数据。
	NoMetaSync
	// DupSort 表示开启键值对的重复排序功能。
	DupSort
	// IntegerKey 表示键为整数类型。
	IntegerKey
	// IntegerDupKey 表示重复键为整数类型。
	IntegerDupKey
)

var DatabaseAlreadyOpenError = &Error{"database already open", nil}

// DB 结构体实现了DB接口，是Boltdb数据库的具体实现。
type DB struct {
	sync.Mutex
	opened   bool
	file     *os.File
	metafile *os.File
	data     []byte
	buf      []byte
	m0       *meta
	m1       *meta
	pageSize int
	readers  []*reader
	buckets  []*Bucket
	//xbuckets       []*bucketx /**< array of static DB info */
	bucketFlags     []int /**< array of flags from MDB_db.md_flags */
	path            string
	mmapSize        int /**< size of the data memory map */
	size            int /**< current file size */
	pbuf            []byte
	transaction     *transaction /**< current write transaction */
	maxPageNumber   int          /**< me_mapsize / me_psize */
	pageState       pageState    /**< state of old pages from freeDB */
	dpages          []*page      /**< list of malloc'd blocks for re-use */
	freePages       []int        /** IDL of pages that became unused in a write txn */
	dirtyPages      []int        /** ID2L of pages written during a write txn. Length MDB_IDL_UM_SIZE. */
	maxFreeOnePage  int          /** Max number of freelist items that can fit in a single overflow page */
	maxPageDataSize int
	maxNodeSize     int /** Max size of a node on a page */
	maxKeySize      int /**< max size of a key */
}

// NewDB 创建并返回一个新的Boltdb数据库实例。
func NewDB() *DB {
	return &DB{}
}

func (db *DB) Open(path string, mode os.FileMode) error {
	var err error
	db.Lock()
	defer db.Unlock()
	// 检查数据库是否已经打开
	// 如果数据库已经打开，则返回一个DatabaseAlreadyOpenError错误。
	if db.opened {
		return DatabaseAlreadyOpenError
	}
	db.path = path
	if db.file, err = os.OpenFile(db.path, os.O_RDWR|os.O_CREATE, mode); err != nil {
		db.Close()
		return err
	}
	if db.metafile, err = os.OpenFile(db.path, os.O_RDWR|os.O_SYNC, mode); err != nil {
		db.Close()
		return err
	}

	var m, m0, m1 *meta
	var buf [pageHeaderSize + int(unsafe.Sizeof(meta{}))]byte
	if _, err = db.file.ReadAt(buf[:], 0); err == nil {
		if m0, _ = db.page(buf[:], 0).meta(); m0 != nil {
			db.pageSize = int(m0.free.pad)
		}
	}
	if _, err = db.file.ReadAt(buf[:], int64(db.pageSize)); err == nil {
		m1, _ = db.page(buf[:], 0).meta()
	}

	if m0 != nil && m1 != nil {
		if m0.txnid > m1.txnid {
			m = m0
		} else {
			m = m1
		}
	}
	// Initialize the page size for new environments.
	if m == nil {
		if err = db.init(); err != nil {
			db.Close()
		}
	}
	// Initialize db fields.
	db.buf = make([]byte, db.pageSize)
	db.maxPageDataSize = ((db.pageSize - pageHeaderSize) / int(unsafe.Sizeof(pgno(0)))) - 1
	db.maxNodeSize = (((db.pageSize - pageHeaderSize) / minKeyCount) & -2) - int(unsafe.Sizeof(indx(0)))
	if err = db.mmap(); err != nil {
		db.Close()
		return err
	}
	db.buf = make([]byte, db.pageSize)
	db.opened = true
	return nil
}

// mmap函数用于将数据库文件映射到内存中。
// 它首先检查文件大小是否足够，然后尝试将文件映射到内存，并初始化相关的页面数据。
// 参数:
// - db *DB: 表示数据库的实例，包含文件句柄和页面大小等信息。
// 返回值:
// - error: 如果映射过程中遇到错误，则返回错误信息；否则返回nil。
func (db *DB) mmap() error {
	var err error
	var size int

	// 检查文件大小是否足够。
	if info, err := os.Stat(db.file.Name()); err != nil {
		return err // 无法获取文件状态时返回错误。
	} else if info.Size() < int64(db.pageSize*2) {
		return &Error{"file size is too small", nil} // 文件大小太小，不满足要求。
	} else {
		size = int(info.Size()) // 文件大小满足要求，记录大小。
	}

	// 尝试将文件映射到内存。
	if db.data, err = syscall.Mmap(int(db.file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED); err != nil {
		return err // 映射文件到内存失败。
	}

	// 初始化meta0和meta1页面。
	if db.m0, err = db.page(db.data, 0).meta(); err != nil {
		return &Error{"meta0 error", err} // 初始化meta0页面失败。
	}
	if db.m1, err = db.page(db.data, 1).meta(); err != nil {
		return &Error{"meta1 error", err} // 初始化meta1页面失败。
	}

	return nil // 映射和初始化成功，返回nil。
}

// init creates a new database file and initializes its meta pages.

func (db *DB) init() error {
	// 将页面大小设置为操作系统页面大小，但限制在最大允许值之内。
	db.pageSize = os.Getpagesize()
	if db.pageSize > maxPageSize {
		db.pageSize = maxPageSize
	}

	// 分配一个缓冲区以容纳两个页面的数据。
	buf := make([]byte, db.pageSize*2)

	// 初始化元数据页面，设置它们的ID和元数据信息。
	for i := 0; i < 2; i++ {
		p := db.page(buf[:], i) // 使用缓冲区和页面索引创建页面实例。
		p.id = pgno(i)          // 设置页面ID。
		p.initMeta(db.pageSize) // 使用配置的页面大小初始化页面上的元数据。
	}

	// 将初始化后的元数据页面写入元数据文件的起始位置。
	if _, err := db.metafile.WriteAt(buf, 0); err != nil {
		return err // 写入失败时返回错误。
	}

	return nil // 初始化成功则返回nil。
}

func (db *DB) close() {
	//TODO
}

// page 根据当前页面大小，从给定字节数组中检索页面引用。
func (db *DB) page(b []byte, id int) *page {
	// 计算id对应的页面在b中的起始位置
	offset := id * db.pageSize
	// 将计算得到的偏移量位置视为*page类型指针返回，这里假设b[offset:offset+db.pageSize]区域存放的是一个page结构体
	return (*page)(unsafe.Pointer(&b[offset]))
}

func (db *DB) freePage(p *page) {
	/*
		mp->mp_next = env->me_dpages;
		VGMEMP_FREE(env, mp);
		env->me_dpages = mp;
	*/
}

func (db *DB) freeDirtyPage(p *page) {
	/*
		if (!IS_OVERFLOW(dp) || dp->mp_pages == 1) {
			mdb_page_free(env, dp);
		} else {
			// large pages just get freed directly
			VGMEMP_FREE(env, dp);
			free(dp);
		}
	*/
}

func (db *DB) freeAllDirtyPages(p *page) {
	/*
		MDB_env *env = txn->mt_env;
		MDB_ID2L dl = txn->mt_u.dirty_list;
		unsigned i, n = dl[0].mid;

		for (i = 1; i <= n; i++) {
			mdb_dpage_free(env, dl[i].mptr);
		}
		dl[0].mid = 0;
	*/
}

func (db *DB) sync(force bool) error {
	/*
			int rc = 0;
			if (force || !F_ISSET(env->me_flags, MDB_NOSYNC)) {
				if (env->me_flags & MDB_WRITEMAP) {
					int flags = ((env->me_flags & MDB_MAPASYNC) && !force)
						? MS_ASYNC : MS_SYNC;
					if (MDB_MSYNC(env->me_map, env->me_mapsize, flags))
						rc = ErrCode();
		#ifdef _WIN32
					else if (flags == MS_SYNC && MDB_FDATASYNC(env->me_fd))
						rc = ErrCode();
		#endif
				} else {
					if (MDB_FDATASYNC(env->me_fd))
						rc = ErrCode();
				}
			}
			return rc;
	*/
	return nil
}

func (db *DB) Transaction(parent *transaction, flags int) (*transaction, error) {
	/*
		MDB_txn *txn;
		MDB_ntxn *ntxn;
		int rc, size, tsize = sizeof(MDB_txn);

		if (env->me_flags & MDB_FATAL_ERROR) {
			DPUTS("environment had fatal error, must shutdown!");
			return MDB_PANIC;
		}
		if ((env->me_flags & MDB_RDONLY) && !(flags & MDB_RDONLY))
			return EACCES;
		if (parent) {
			// Nested transactions: Max 1 child, write txns only, no writemap
			if (parent->mt_child ||
				(flags & MDB_RDONLY) ||
				(parent->mt_flags & (MDB_TXN_RDONLY|MDB_TXN_ERROR)) ||
				(env->me_flags & MDB_WRITEMAP))
			{
				return (parent->mt_flags & MDB_TXN_RDONLY) ? EINVAL : MDB_BAD_TXN;
			}
			tsize = sizeof(MDB_ntxn);
		}
		size = tsize + env->me_maxdbs * (sizeof(MDB_db)+1);
		if (!(flags & MDB_RDONLY))
			size += env->me_maxdbs * sizeof(MDB_cursor *);

		if ((txn = calloc(1, size)) == NULL) {
			DPRINTF(("calloc: %s", strerror(ErrCode())));
			return ENOMEM;
		}
		txn->mt_dbs = (MDB_db *) ((char *)txn + tsize);
		if (flags & MDB_RDONLY) {
			txn->mt_flags |= MDB_TXN_RDONLY;
			txn->mt_dbflags = (unsigned char *)(txn->mt_dbs + env->me_maxdbs);
		} else {
			txn->mt_cursors = (MDB_cursor **)(txn->mt_dbs + env->me_maxdbs);
			txn->mt_dbflags = (unsigned char *)(txn->mt_cursors + env->me_maxdbs);
		}
		txn->mt_env = env;

		if (parent) {
			unsigned int i;
			txn->mt_u.dirty_list = malloc(sizeof(MDB_ID2)*MDB_IDL_UM_SIZE);
			if (!txn->mt_u.dirty_list ||
				!(txn->mt_free_pgs = mdb_midl_alloc(MDB_IDL_UM_MAX)))
			{
				free(txn->mt_u.dirty_list);
				free(txn);
				return ENOMEM;
			}
			txn->mt_txnid = parent->mt_txnid;
			txn->mt_dirty_room = parent->mt_dirty_room;
			txn->mt_u.dirty_list[0].mid = 0;
			txn->mt_spill_pgs = NULL;
			txn->mt_next_pgno = parent->mt_next_pgno;
			parent->mt_child = txn;
			txn->mt_parent = parent;
			txn->mt_numdbs = parent->mt_numdbs;
			txn->mt_flags = parent->mt_flags;
			txn->mt_dbxs = parent->mt_dbxs;
			memcpy(txn->mt_dbs, parent->mt_dbs, txn->mt_numdbs * sizeof(MDB_db));
			// Copy parent's mt_dbflags, but clear DB_NEW
			for (i=0; i<txn->mt_numdbs; i++)
				txn->mt_dbflags[i] = parent->mt_dbflags[i] & ~DB_NEW;
			rc = 0;
			ntxn = (MDB_ntxn *)txn;
			ntxn->mnt_pgstate = env->me_pgstate; // save parent me_pghead & co
			if (env->me_pghead) {
				size = MDB_IDL_SIZEOF(env->me_pghead);
				env->me_pghead = mdb_midl_alloc(env->me_pghead[0]);
				if (env->me_pghead)
					memcpy(env->me_pghead, ntxn->mnt_pgstate.mf_pghead, size);
				else
					rc = ENOMEM;
			}
			if (!rc)
				rc = mdb_cursor_shadow(parent, txn);
			if (rc)
				mdb_txn_reset0(txn, "beginchild-fail");
		} else {
			rc = mdb_txn_renew0(txn);
		}
		if (rc)
			free(txn);
		else {
			*ret = txn;
			DPRINTF(("begin txn %"Z"u%c %p on mdbenv %p, root page %"Z"u",
				txn->mt_txnid, (txn->mt_flags & MDB_TXN_RDONLY) ? 'r' : 'w',
				(void *) txn, (void *) env, txn->mt_dbs[MAIN_DBI].md_root));
		}

		return rc;
	*/
	return nil, nil
}

func (db *DB) pickMeta() int {
	/*
		return (env->me_metas[0]->mm_txnid < env->me_metas[1]->mm_txnid);
	*/
	return 0
}

func (db *DB) Create() error {
	/*
			MDB_env *e;

			e = calloc(1, sizeof(MDB_env));
			if (!e)
				return ENOMEM;

			e->me_maxreaders = DEFAULT_READERS;
			e->me_maxdbs = e->me_numdbs = 2;
			e->me_fd = INVALID_HANDLE_VALUE;
			e->me_lfd = INVALID_HANDLE_VALUE;
			e->me_mfd = INVALID_HANDLE_VALUE;
		#ifdef MDB_USE_POSIX_SEM
			e->me_rmutex = SEM_FAILED;
			e->me_wmutex = SEM_FAILED;
		#endif
			e->me_pid = getpid();
			GET_PAGESIZE(e->me_os_psize);
			VGMEMP_CREATE(e,0,0);
			*env = e;
			return MDB_SUCCESS;
	*/
	return nil
}

func (db *DB) setMapSize(size int) error {
	/*
		// If env is already open, caller is responsible for making
		// sure there are no active txns.
		if (env->me_map) {
			int rc;
			void *old;
			if (env->me_txn)
				return EINVAL;
			if (!size)
				size = env->me_metas[mdb_env_pick_meta(env)]->mm_mapsize;
			else if (size < env->me_mapsize) {
				// If the configured size is smaller, make sure it's
				// still big enough. Silently round up to minimum if not.
				size_t minsize = (env->me_metas[mdb_env_pick_meta(env)]->mm_last_pg + 1) * env->me_psize;
				if (size < minsize)
					size = minsize;
			}
			munmap(env->me_map, env->me_mapsize);
			env->me_mapsize = size;
			old = (env->me_flags & MDB_FIXEDMAP) ? env->me_map : NULL;
			rc = mdb_env_map(env, old, 1);
			if (rc)
				return rc;
		}
		env->me_mapsize = size;
		if (env->me_psize)
			env->me_maxpg = env->me_mapsize / env->me_psize;
		return MDB_SUCCESS;
	*/
	return nil
}

func (db *DB) setMaxBucketCount(count int) error {
	/*
		if (env->me_map)
			return EINVAL;
		env->me_maxdbs = dbs + 2; // Named databases + main and free DB
		return MDB_SUCCESS;
	*/
	return nil
}

func (db *DB) setMaxReaderCount(count int) error {
	/*
		if (env->me_map || readers < 1)
			return EINVAL;
		env->me_maxreaders = readers;
		return MDB_SUCCESS;
	*/
	return nil
}

func (db *DB) getMaxReaderCount(count int) (int, error) {
	/*
		if (!env || !readers)
			return EINVAL;
		*readers = env->me_maxreaders;
		return MDB_SUCCESS;
	*/
	return 0, nil
}

func (db *DB) close0(excl int) {

}

func (db *DB) copyfd(handle int) error {
	/*
			MDB_txn *txn = NULL;
			int rc;
			size_t wsize;
			char *ptr;
		#ifdef _WIN32
			DWORD len, w2;
		#define DO_WRITE(rc, fd, ptr, w2, len)	rc = WriteFile(fd, ptr, w2, &len, NULL)
		#else
			ssize_t len;
			size_t w2;
		#define DO_WRITE(rc, fd, ptr, w2, len)	len = write(fd, ptr, w2); rc = (len >= 0)
		#endif

			// Do the lock/unlock of the reader mutex before starting the
			// write txn.  Otherwise other read txns could block writers.
			rc = mdb_txn_begin(env, NULL, MDB_RDONLY, &txn);
			if (rc)
				return rc;

			if (env->me_txns) {
				// We must start the actual read txn after blocking writers
				mdb_txn_reset0(txn, "reset-stage1");

				// Temporarily block writers until we snapshot the meta pages
				LOCK_MUTEX_W(env);

				rc = mdb_txn_renew0(txn);
				if (rc) {
					UNLOCK_MUTEX_W(env);
					goto leave;
				}
			}

			wsize = env->me_psize * 2;
			ptr = env->me_map;
			w2 = wsize;
			while (w2 > 0) {
				DO_WRITE(rc, fd, ptr, w2, len);
				if (!rc) {
					rc = ErrCode();
					break;
				} else if (len > 0) {
					rc = MDB_SUCCESS;
					ptr += len;
					w2 -= len;
					continue;
				} else {
					// Non-blocking or async handles are not supported
					rc = EIO;
					break;
				}
			}
			if (env->me_txns)
				UNLOCK_MUTEX_W(env);

			if (rc)
				goto leave;

			wsize = txn->mt_next_pgno * env->me_psize - wsize;
			while (wsize > 0) {
				if (wsize > MAX_WRITE)
					w2 = MAX_WRITE;
				else
					w2 = wsize;
				DO_WRITE(rc, fd, ptr, w2, len);
				if (!rc) {
					rc = ErrCode();
					break;
				} else if (len > 0) {
					rc = MDB_SUCCESS;
					ptr += len;
					wsize -= len;
					continue;
				} else {
					rc = EIO;
					break;
				}
			}

		leave:
			mdb_txn_abort(txn);
			return rc;
		}

		int
		mdb_env_copy(MDB_env *env, const char *path)
		{
			int rc, len;
			char *lpath;
			HANDLE newfd = INVALID_HANDLE_VALUE;

			if (env->me_flags & MDB_NOSUBDIR) {
				lpath = (char *)path;
			} else {
				len = strlen(path);
				len += sizeof(DATANAME);
				lpath = malloc(len);
				if (!lpath)
					return ENOMEM;
				sprintf(lpath, "%s" DATANAME, path);
			}

			// The destination path must exist, but the destination file must not.
			// We don't want the OS to cache the writes, since the source data is
			// already in the OS cache.
		#ifdef _WIN32
			newfd = CreateFile(lpath, GENERIC_WRITE, 0, NULL, CREATE_NEW,
						FILE_FLAG_NO_BUFFERING|FILE_FLAG_WRITE_THROUGH, NULL);
		#else
			newfd = open(lpath, O_WRONLY|O_CREAT|O_EXCL, 0666);
		#endif
			if (newfd == INVALID_HANDLE_VALUE) {
				rc = ErrCode();
				goto leave;
			}

		#ifdef O_DIRECT
			// Set O_DIRECT if the file system supports it
			if ((rc = fcntl(newfd, F_GETFL)) != -1)
				(void) fcntl(newfd, F_SETFL, rc | O_DIRECT);
		#endif
		#ifdef F_NOCACHE	// __APPLE__
			rc = fcntl(newfd, F_NOCACHE, 1);
			if (rc) {
				rc = ErrCode();
				goto leave;
			}
		#endif

			rc = mdb_env_copyfd(env, newfd);

		leave:
			if (!(env->me_flags & MDB_NOSUBDIR))
				free(lpath);
			if (newfd != INVALID_HANDLE_VALUE)
				if (close(newfd) < 0 && rc == MDB_SUCCESS)
					rc = ErrCode();

			return rc;
	*/
	return nil
}

func (db *DB) Close() {
	/*
		MDB_page *dp;

		if (env == NULL)
			return;

		VGMEMP_DESTROY(env);
		while ((dp = env->me_dpages) != NULL) {
			VGMEMP_DEFINED(&dp->mp_next, sizeof(dp->mp_next));
			env->me_dpages = dp->mp_next;
			free(dp);
		}

		mdb_env_close0(env, 0);
		free(env);
	*/
}

// Calculate the size of a leaf node.
// The size depends on the environment's page size; if a data item
// is too large it will be put onto an overflow page and the node
// size will only include the key and not the data. Sizes are always
// rounded up to an even number of bytes, to guarantee 2-byte alignment
// of the #MDB_node headers.
// @param[in] env The environment handle.
// @param[in] key The key for the node.
// @param[in] data The data for the node.
// @return The number of bytes needed to store the node.
func (db *DB) LeafSize(key []byte, data []byte) int {
	/*
		size_t		 sz;

		sz = LEAFSIZE(key, data);
		if (sz > env->me_nodemax) {
			// put on overflow page
			sz -= data->mv_size - sizeof(pgno_t);
		}

		return EVEN(sz + sizeof(indx_t));
	*/
	return 0
}

// Calculate the size of a branch node.
// The size should depend on the environment's page size but since
// we currently don't support spilling large keys onto overflow
// pages, it's simply the size of the #MDB_node header plus the
// size of the key. Sizes are always rounded up to an even number
// of bytes, to guarantee 2-byte alignment of the #MDB_node headers.
// @param[in] env The environment handle.
// @param[in] key The key for the node.
// @return The number of bytes needed to store the node.
func (db *DB) BranchSize(key []byte) int {
	/*
		size_t		 sz;

		sz = INDXSIZE(key);
		if (sz > env->me_nodemax) {
			// put on overflow page
			// not implemented
			// sz -= key->size - sizeof(pgno_t);
		}

		return sz + sizeof(indx_t);
	*/
	return 0
}

func (db *DB) SetFlags(flag int, onoff bool) error {
	/*
		if ((flag & CHANGEABLE) != flag)
			return EINVAL;
		if (onoff)
			env->me_flags |= flag;
		else
			env->me_flags &= ~flag;
		return MDB_SUCCESS;
	*/
	return nil
}

func (db *DB) Stat() *Stat {
	/*
		int toggle;

		if (env == NULL || arg == NULL)
			return EINVAL;

		toggle = mdb_env_pick_meta(env);
		stat := &Stat{}
		stat->ms_psize = env->me_psize;
		stat->ms_depth = db->md_depth;
		stat->ms_branch_pages = db->md_branch_pages;
		stat->ms_leaf_pages = db->md_leaf_pages;
		stat->ms_overflow_pages = db->md_overflow_pages;
		stat->ms_entries = db->md_entries;

		//return mdb_stat0(env, &env->me_metas[toggle]->mm_dbs[MAIN_DBI], stat);
		return stat
	*/
	return nil
}

func (db *DB) Info() *Info {
	/*
		int toggle;

		if (env == NULL || arg == NULL)
			return EINVAL;

		toggle = mdb_env_pick_meta(env);
		arg->me_mapaddr = (env->me_flags & MDB_FIXEDMAP) ? env->me_map : 0;
		arg->me_mapsize = env->me_mapsize;
		arg->me_maxreaders = env->me_maxreaders;

		// me_numreaders may be zero if this process never used any readers. Use
		// the shared numreader count if it exists.
		arg->me_numreaders = env->me_txns ? env->me_txns->mti_numreaders : env->me_numreaders;

		arg->me_last_pgno = env->me_metas[toggle]->mm_last_pg;
		arg->me_last_txnid = env->me_metas[toggle]->mm_txnid;
		return MDB_SUCCESS;
	*/
	return nil
}

// TODO: Move to bucket.go
func (db *DB) CloseBucket(b Bucket) {
	/*
		char *ptr;
		if (dbi <= MAIN_DBI || dbi >= env->me_maxdbs)
			return;
		ptr = env->me_dbxs[dbi].md_name.mv_data;
		env->me_dbxs[dbi].md_name.mv_data = NULL;
		env->me_dbxs[dbi].md_name.mv_size = 0;
		env->me_dbflags[dbi] = 0;
		free(ptr);
	*/
}

// int mdb_reader_list(MDB_env *env, MDB_msg_func *func, void *ctx)
func (db *DB) getReaderList() error {
	/*
		unsigned int i, rdrs;
		MDB_reader *mr;
		char buf[64];
		int rc = 0, first = 1;

		if (!env || !func)
			return -1;
		if (!env->me_txns) {
			return func("(no reader locks)\n", ctx);
		}
		rdrs = env->me_txns->mti_numreaders;
		mr = env->me_txns->mti_readers;
		for (i=0; i<rdrs; i++) {
			if (mr[i].mr_pid) {
				txnid_t	txnid = mr[i].mr_txnid;
				sprintf(buf, txnid == (txnid_t)-1 ?
					"%10d %"Z"x -\n" : "%10d %"Z"x %"Z"u\n",
					(int)mr[i].mr_pid, (size_t)mr[i].mr_tid, txnid);
				if (first) {
					first = 0;
					rc = func("    pid     thread     txnid\n", ctx);
					if (rc < 0)
						break;
				}
				rc = func(buf, ctx);
				if (rc < 0)
					break;
			}
		}
		if (first) {
			rc = func("(no active readers)\n", ctx);
		}
		return rc;
	*/
	return nil
}

// (bool return is whether reader is dead)
func (db *DB) checkReaders() (bool, error) {
	/*
		unsigned int i, j, rdrs;
		MDB_reader *mr;
		MDB_PID_T *pids, pid;
		int count = 0;

		if (!env)
			return EINVAL;
		if (dead)
			*dead = 0;
		if (!env->me_txns)
			return MDB_SUCCESS;
		rdrs = env->me_txns->mti_numreaders;
		pids = malloc((rdrs+1) * sizeof(MDB_PID_T));
		if (!pids)
			return ENOMEM;
		pids[0] = 0;
		mr = env->me_txns->mti_readers;
		for (i=0; i<rdrs; i++) {
			if (mr[i].mr_pid && mr[i].mr_pid != env->me_pid) {
				pid = mr[i].mr_pid;
				if (mdb_pid_insert(pids, pid) == 0) {
					if (!mdb_reader_pid(env, Pidcheck, pid)) {
						LOCK_MUTEX_R(env);
						// Recheck, a new process may have reused pid
						if (!mdb_reader_pid(env, Pidcheck, pid)) {
							for (j=i; j<rdrs; j++)
								if (mr[j].mr_pid == pid) {
									DPRINTF(("clear stale reader pid %u txn %"Z"d",
										(unsigned) pid, mr[j].mr_txnid));
									mr[j].mr_pid = 0;
									count++;
								}
						}
						UNLOCK_MUTEX_R(env);
					}
				}
			}
		}
		free(pids);
		if (dead)
			*dead = count;
		return MDB_SUCCESS;
	*/
	return false, nil
}
