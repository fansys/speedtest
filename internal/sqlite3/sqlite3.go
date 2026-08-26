// Package sqlite3 是一个极小的 SQLite CGO 绑定，只暴露 internal/store 需要的那部分能力。
//
// 之所以手写而不是依赖 modernc.org/sqlite 或 mattn/go-sqlite3：构建环境无法访问外网下载
// Go module，也没有 root 权限安装 libsqlite3-dev（缺 sqlite3.h）。系统已经安装了运行时库
// libsqlite3.so.0，SQLite 的 C ABI 数十年保持稳定，因此这里直接手写必要函数的 extern 声明
// 并动态链接到系统库，不需要头文件也不需要下载任何依赖。
//
// 只封装了 prepare/bind/step/column 这条主线（预处理语句 + 参数绑定），刻意不提供任何拼接
// SQL 字符串的入口，调用方必须使用 ? 占位符，从根源上避免 SQL 注入。
package sqlite3

/*
#cgo linux LDFLAGS: -l:libsqlite3.so.0
#cgo darwin LDFLAGS: -lsqlite3
#include <stdlib.h>

typedef struct sqlite3 sqlite3;
typedef struct sqlite3_stmt sqlite3_stmt;

extern int sqlite3_open_v2(const char *filename, sqlite3 **ppDb, int flags, const char *zVfs);
extern int sqlite3_close_v2(sqlite3 *db);
extern int sqlite3_exec(sqlite3 *db, const char *sql, void *cb, void *arg, char **errmsg);
extern void sqlite3_free(void *p);
extern const char *sqlite3_errmsg(sqlite3 *db);
extern int sqlite3_extended_errcode(sqlite3 *db);
extern int sqlite3_busy_timeout(sqlite3 *db, int ms);

extern int sqlite3_prepare_v2(sqlite3 *db, const char *zSql, int nByte, sqlite3_stmt **ppStmt, const char **pzTail);
extern int sqlite3_step(sqlite3_stmt *stmt);
extern int sqlite3_finalize(sqlite3_stmt *stmt);
extern int sqlite3_reset(sqlite3_stmt *stmt);
extern int sqlite3_clear_bindings(sqlite3_stmt *stmt);

extern int sqlite3_bind_text(sqlite3_stmt *stmt, int idx, const char *val, int n, void (*destructor)(void*));
extern int sqlite3_bind_int64(sqlite3_stmt *stmt, int idx, long long val);
extern int sqlite3_bind_double(sqlite3_stmt *stmt, int idx, double val);
extern int sqlite3_bind_null(sqlite3_stmt *stmt, int idx);

extern int sqlite3_column_count(sqlite3_stmt *stmt);
extern int sqlite3_column_type(sqlite3_stmt *stmt, int col);
extern long long sqlite3_column_int64(sqlite3_stmt *stmt, int col);
extern double sqlite3_column_double(sqlite3_stmt *stmt, int col);
extern const unsigned char *sqlite3_column_text(sqlite3_stmt *stmt, int col);
extern int sqlite3_column_bytes(sqlite3_stmt *stmt, int col);

extern long long sqlite3_last_insert_rowid(sqlite3 *db);
extern int sqlite3_changes(sqlite3 *db);

#define GO_SQLITE_OPEN_READWRITE 0x00000002
#define GO_SQLITE_OPEN_CREATE    0x00000004
#define GO_SQLITE_OPEN_FULLMUTEX 0x00010000

static int go_sqlite3_open(const char *path, sqlite3 **db) {
    return sqlite3_open_v2(path, db, GO_SQLITE_OPEN_READWRITE | GO_SQLITE_OPEN_CREATE | GO_SQLITE_OPEN_FULLMUTEX, NULL);
}

static int go_sqlite3_exec_simple(sqlite3 *db, const char *sql, char **errmsg) {
    return sqlite3_exec(db, sql, NULL, NULL, errmsg);
}

static int go_sqlite3_bind_text(sqlite3_stmt *stmt, int idx, const char *val, int n) {
    // SQLITE_TRANSIENT == (void(*)(void*))-1，让 sqlite 自己复制一份，
    // 调用方绑定后可以立即释放传入的 C 字符串。
    return sqlite3_bind_text(stmt, idx, val, n, (void (*)(void*))-1);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

const (
	sqliteOK         = 0
	sqliteRow        = 100
	sqliteDone       = 101
	sqliteConstraint = 19

	typeInteger = 1
	typeFloat   = 2
	typeText    = 3
	typeBlob    = 4
	typeNull    = 5
)

// ErrConstraint 在唯一约束等冲突时返回，调用方可以用 errors.Is 判断。
var ErrConstraint = errors.New("sqlite: constraint violation")

// DB 是对一个 sqlite3* 连接的包装。所有方法都通过内部互斥锁串行化，
// 因为我们只使用单个连接，不依赖 SQLite 的多线程模式做并发控制。
type DB struct {
	mu     sync.Mutex
	handle *C.sqlite3
}

// Open 打开（或创建）指定路径的 SQLite 数据库文件。
func Open(path string) (*DB, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var handle *C.sqlite3
	rc := C.go_sqlite3_open(cpath, &handle)
	if rc != sqliteOK {
		msg := "无法打开数据库"
		if handle != nil {
			msg = C.GoString(C.sqlite3_errmsg(handle))
			C.sqlite3_close_v2(handle)
		}
		return nil, fmt.Errorf("sqlite3_open 失败(rc=%d): %s", int(rc), msg)
	}
	C.sqlite3_busy_timeout(handle, 5000)
	return &DB{handle: handle}, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.handle == nil {
		return nil
	}
	rc := C.sqlite3_close_v2(db.handle)
	db.handle = nil
	if rc != sqliteOK {
		return fmt.Errorf("sqlite3_close_v2 失败(rc=%d)", int(rc))
	}
	return nil
}

func (db *DB) errmsg() string {
	return C.GoString(C.sqlite3_errmsg(db.handle))
}

// Exec 执行不需要参数、不返回结果集的语句（主要用于建表 DDL 与 PRAGMA）。
// 出于避免 SQL 注入的考虑，这个方法只应该用于调用方硬编码的语句，不能拼接外部输入。
func (db *DB) Exec(sql string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	csql := C.CString(sql)
	defer C.free(unsafe.Pointer(csql))

	var errmsg *C.char
	rc := C.go_sqlite3_exec_simple(db.handle, csql, &errmsg)
	if rc != sqliteOK {
		msg := db.errmsg()
		if errmsg != nil {
			msg = C.GoString(errmsg)
			C.sqlite3_free(unsafe.Pointer(errmsg))
		}
		return fmt.Errorf("sqlite exec 失败: %s", msg)
	}
	return nil
}

// Stmt 是一个预处理语句，绑定的参数用 ? 占位符（1-based index）。
type Stmt struct {
	db     *DB
	handle *C.sqlite3_stmt
}

// Prepare 编译一条带 ? 占位符的 SQL 语句。调用方必须始终使用参数绑定传入外部数据，
// 不允许把外部输入拼进 SQL 字符串本身。
func (db *DB) Prepare(sql string) (*Stmt, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	csql := C.CString(sql)
	defer C.free(unsafe.Pointer(csql))

	var handle *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(db.handle, csql, -1, &handle, nil)
	if rc != sqliteOK {
		return nil, fmt.Errorf("sqlite prepare 失败: %s", db.errmsg())
	}
	return &Stmt{db: db, handle: handle}, nil
}

// Finalize 释放预处理语句。
func (s *Stmt) Finalize() error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rc := C.sqlite3_finalize(s.handle)
	s.handle = nil
	if rc != sqliteOK {
		return fmt.Errorf("sqlite finalize 失败(rc=%d)", int(rc))
	}
	return nil
}

func (s *Stmt) BindText(idx int, val string) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	cval := C.CString(val)
	defer C.free(unsafe.Pointer(cval))
	rc := C.go_sqlite3_bind_text(s.handle, C.int(idx), cval, C.int(len(val)))
	if rc != sqliteOK {
		return fmt.Errorf("bind_text 失败(rc=%d)", int(rc))
	}
	return nil
}

func (s *Stmt) BindInt64(idx int, val int64) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rc := C.sqlite3_bind_int64(s.handle, C.int(idx), C.longlong(val))
	if rc != sqliteOK {
		return fmt.Errorf("bind_int64 失败(rc=%d)", int(rc))
	}
	return nil
}

func (s *Stmt) BindDouble(idx int, val float64) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rc := C.sqlite3_bind_double(s.handle, C.int(idx), C.double(val))
	if rc != sqliteOK {
		return fmt.Errorf("bind_double 失败(rc=%d)", int(rc))
	}
	return nil
}

func (s *Stmt) BindNull(idx int) error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rc := C.sqlite3_bind_null(s.handle, C.int(idx))
	if rc != sqliteOK {
		return fmt.Errorf("bind_null 失败(rc=%d)", int(rc))
	}
	return nil
}

// BindNullableText：字符串指针为 nil 时绑定 NULL，否则绑定其值。
func (s *Stmt) BindNullableText(idx int, val *string) error {
	if val == nil {
		return s.BindNull(idx)
	}
	return s.BindText(idx, *val)
}

// BindNullableDouble：浮点指针为 nil 时绑定 NULL，否则绑定其值。
func (s *Stmt) BindNullableDouble(idx int, val *float64) error {
	if val == nil {
		return s.BindNull(idx)
	}
	return s.BindDouble(idx, *val)
}

// Step 推进一行结果。返回 hasRow=true 表示当前行有数据可读；hasRow=false 且 err=nil
// 表示语句执行完毕（SQLITE_DONE）。
func (s *Stmt) Step() (hasRow bool, err error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rc := C.sqlite3_step(s.handle)
	switch rc {
	case sqliteRow:
		return true, nil
	case sqliteDone:
		return false, nil
	case sqliteConstraint:
		return false, fmt.Errorf("%w: %s", ErrConstraint, s.db.errmsg())
	default:
		return false, fmt.Errorf("sqlite step 失败(rc=%d): %s", int(rc), s.db.errmsg())
	}
}

// Reset 允许重新绑定参数并复用同一条预处理语句。
func (s *Stmt) Reset() error {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	C.sqlite3_clear_bindings(s.handle)
	rc := C.sqlite3_reset(s.handle)
	if rc != sqliteOK {
		return fmt.Errorf("sqlite reset 失败(rc=%d)", int(rc))
	}
	return nil
}

func (s *Stmt) ColumnText(col int) string {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	ptr := C.sqlite3_column_text(s.handle, C.int(col))
	if ptr == nil {
		return ""
	}
	n := C.sqlite3_column_bytes(s.handle, C.int(col))
	return C.GoStringN((*C.char)(unsafe.Pointer(ptr)), n)
}

func (s *Stmt) ColumnIsNull(col int) bool {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	return C.sqlite3_column_type(s.handle, C.int(col)) == typeNull
}

func (s *Stmt) ColumnInt64(col int) int64 {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	return int64(C.sqlite3_column_int64(s.handle, C.int(col)))
}

func (s *Stmt) ColumnDouble(col int) float64 {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	return float64(C.sqlite3_column_double(s.handle, C.int(col)))
}

// ColumnNullableText 在该列为 NULL 时返回 nil。
func (s *Stmt) ColumnNullableText(col int) *string {
	if s.ColumnIsNull(col) {
		return nil
	}
	v := s.ColumnText(col)
	return &v
}

// ColumnNullableDouble 在该列为 NULL 时返回 nil。
func (s *Stmt) ColumnNullableDouble(col int) *float64 {
	if s.ColumnIsNull(col) {
		return nil
	}
	v := s.ColumnDouble(col)
	return &v
}

// LastInsertRowID 返回最近一次 INSERT 生成的 rowid。
func (db *DB) LastInsertRowID() int64 {
	db.mu.Lock()
	defer db.mu.Unlock()
	return int64(C.sqlite3_last_insert_rowid(db.handle))
}

// Changes 返回最近一条语句影响的行数。
func (db *DB) Changes() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return int(C.sqlite3_changes(db.handle))
}
