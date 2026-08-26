package sqlite3

import (
	"path/filepath"
	"testing"
)

func TestOpenExecQueryRoundtrip(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, score REAL, note TEXT, UNIQUE(name))`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	ins, err := db.Prepare(`INSERT INTO t (name, score, note) VALUES (?, ?, ?)`)
	if err != nil {
		t.Fatalf("prepare insert: %v", err)
	}
	if err := ins.BindText(1, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := ins.BindDouble(2, 3.5); err != nil {
		t.Fatal(err)
	}
	if err := ins.BindNull(3); err != nil {
		t.Fatal(err)
	}
	if _, err := ins.Step(); err != nil {
		t.Fatalf("step insert: %v", err)
	}
	ins.Finalize()

	id := db.LastInsertRowID()
	if id != 1 {
		t.Fatalf("expected id=1 got %d", id)
	}

	sel, err := db.Prepare(`SELECT id, name, score, note FROM t WHERE id = ?`)
	if err != nil {
		t.Fatalf("prepare select: %v", err)
	}
	defer sel.Finalize()
	if err := sel.BindInt64(1, id); err != nil {
		t.Fatal(err)
	}
	hasRow, err := sel.Step()
	if err != nil {
		t.Fatalf("step select: %v", err)
	}
	if !hasRow {
		t.Fatal("expected a row")
	}
	if got := sel.ColumnInt64(0); got != 1 {
		t.Fatalf("id: got %d", got)
	}
	if got := sel.ColumnText(1); got != "alice" {
		t.Fatalf("name: got %q", got)
	}
	if got := sel.ColumnDouble(2); got != 3.5 {
		t.Fatalf("score: got %v", got)
	}
	if !sel.ColumnIsNull(3) {
		t.Fatal("expected note to be NULL")
	}

	hasRow, err = sel.Step()
	if err != nil {
		t.Fatalf("step select 2: %v", err)
	}
	if hasRow {
		t.Fatal("expected no more rows")
	}

	// 唯一约束冲突应该映射为 ErrConstraint。
	ins2, err := db.Prepare(`INSERT INTO t (name) VALUES (?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer ins2.Finalize()
	ins2.BindText(1, "alice")
	if _, err := ins2.Step(); err == nil {
		t.Fatal("expected unique constraint error")
	}
}

func TestReopenPersistsData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatal(err)
	}
	stmt, err := db.Prepare(`INSERT INTO t (id, name) VALUES (?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	stmt.BindInt64(1, 1)
	stmt.BindText(2, "hello")
	if _, err := stmt.Step(); err != nil {
		t.Fatal(err)
	}
	stmt.Finalize()
	db.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	sel, err := db2.Prepare(`SELECT name FROM t WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer sel.Finalize()
	hasRow, err := sel.Step()
	if err != nil || !hasRow {
		t.Fatalf("expected row after reopen, hasRow=%v err=%v", hasRow, err)
	}
	if got := sel.ColumnText(0); got != "hello" {
		t.Fatalf("got %q", got)
	}
}
