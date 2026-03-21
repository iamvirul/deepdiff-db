package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	_ "modernc.org/sqlite"
)

func openMemDBB(b *testing.B) *sql.DB {
	b.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	b.Cleanup(func() { db.Close() })
	return db
}

func seedUsers(b *testing.B, db *sql.DB, n int) schema.Table {
	b.Helper()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id    INTEGER PRIMARY KEY,
		name  TEXT    NOT NULL,
		email TEXT
	)`); err != nil {
		b.Fatalf("create table: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare("INSERT INTO users (id, name, email) VALUES (?, ?, ?)")
	if err != nil {
		b.Fatalf("prepare: %v", err)
	}
	for i := 1; i <= n; i++ {
		if _, err := stmt.Exec(i, fmt.Sprintf("user-%d", i), fmt.Sprintf("u%d@example.com", i)); err != nil {
			b.Fatalf("insert row %d: %v", i, err)
		}
	}
	if err := stmt.Close(); err != nil {
		b.Fatalf("close stmt: %v", err)
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
	return schema.Table{
		Name: "users",
		Columns: map[string]schema.Column{
			"id":    {Name: "id", DataType: "integer", IsNullable: false},
			"name":  {Name: "name", DataType: "text", IsNullable: false},
			"email": {Name: "email", DataType: "text", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
	}
}

// BenchmarkHashTable measures HashTable throughput at different scales and
// batch sizes so we can document rows/sec in the performance guide.
//
//   go test ./tests/content/... -bench=BenchmarkHashTable -benchmem -benchtime=3s
func BenchmarkHashTable_Unbatched_1k(b *testing.B)  { benchHash(b, 1_000, 0) }
func BenchmarkHashTable_Unbatched_10k(b *testing.B) { benchHash(b, 10_000, 0) }
func BenchmarkHashTable_Batched_1k(b *testing.B)    { benchHash(b, 1_000, 500) }
func BenchmarkHashTable_Batched_10k(b *testing.B)   { benchHash(b, 10_000, 1_000) }

func benchHash(b *testing.B, rows, batchSize int) {
	b.Helper()
	db := openMemDBB(b)
	tbl := seedUsers(b, db, rows)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(rows))

	for i := 0; i < b.N; i++ {
		hashes, err := content.HashTable(ctx, db, "sqlite", tbl, nil, batchSize)
		if err != nil {
			b.Fatalf("HashTable: %v", err)
		}
		if len(hashes) != rows {
			b.Fatalf("expected %d hashes, got %d", rows, len(hashes))
		}
	}
}
