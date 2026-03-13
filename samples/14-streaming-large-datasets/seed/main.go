// seed generates two large SQLite databases (prod.db and dev.db) for use with
// the streaming large-datasets sample. It creates three tables:
//
//   - orders     — 500 000 rows; 10 rows modified in dev (amount changed)
//   - products   — 100 000 rows;  5 rows modified in dev (price changed)
//   - audit_logs — 200 000 rows; identical in both (ignored via config)
//
// Run from the sample root:
//
//	go run ./seed            # full size (~800k rows, takes ~60s)
//	go run ./seed --small    # 10k / 2k / 5k rows — completes in seconds
package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

const (
	ordersRows    = 500_000
	productsRows  = 100_000
	auditLogsRows = 200_000
)

func main() {
	small := len(os.Args) > 1 && os.Args[1] == "--small"
	oRows, pRows, aRows := ordersRows, productsRows, auditLogsRows
	if small {
		oRows, pRows, aRows = 10_000, 2_000, 5_000
		fmt.Printf("Running in --small mode (%d / %d / %d rows per table)\n", oRows, pRows, aRows)
	}

	// Seed prod.db
	log.Println("Creating prod.db …")
	prodDB := mustOpen("prod.db")
	setupSchema(prodDB)
	rng := rand.New(rand.NewSource(42))
	seedOrders(prodDB, rng, oRows)
	seedProducts(prodDB, rng, pRows)
	seedAuditLogs(prodDB, rng, aRows)
	prodDB.Close()
	log.Println("prod.db ready")

	// Seed dev.db with identical data (same RNG seed), then mutate.
	log.Println("Creating dev.db …")
	devDB := mustOpen("dev.db")
	setupSchema(devDB)
	rng2 := rand.New(rand.NewSource(42)) // same seed → identical rows
	seedOrders(devDB, rng2, oRows)
	seedProducts(devDB, rng2, pRows)
	seedAuditLogs(devDB, rng2, aRows)

	mutateDev(devDB, rand.New(rand.NewSource(99)), oRows, pRows)
	devDB.Close()
	log.Println("dev.db ready")

	fmt.Println()
	fmt.Println("Databases created. Run 'make diff' (or see README.md) to compare them.")
}

// ---- schema ---------------------------------------------------------------

func setupSchema(db *sql.DB) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			id          INTEGER PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			product_id  INTEGER NOT NULL,
			quantity    INTEGER NOT NULL,
			amount      REAL    NOT NULL,
			status      TEXT    NOT NULL,
			created_at  TEXT    NOT NULL,
			updated_at  TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id         INTEGER PRIMARY KEY,
			name       TEXT    NOT NULL,
			category   TEXT    NOT NULL,
			price      REAL    NOT NULL,
			stock      INTEGER NOT NULL,
			created_at TEXT    NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id         INTEGER PRIMARY KEY,
			entity     TEXT    NOT NULL,
			entity_id  INTEGER NOT NULL,
			action     TEXT    NOT NULL,
			created_at TEXT    NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			log.Fatalf("setup schema: %v", err)
		}
	}
}

// ---- seeders --------------------------------------------------------------

var statuses = []string{"pending", "processing", "shipped", "delivered", "cancelled"}
var categories = []string{"electronics", "clothing", "food", "books", "furniture"}

func seedOrders(db *sql.DB, rng *rand.Rand, n int) {
	base := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	bulkInsert(db, n, 10_000, "orders",
		"id,customer_id,product_id,quantity,amount,status,created_at,updated_at",
		func(i int) []any {
			ts := base.Add(time.Duration(rng.Int63n(int64(365*24*time.Hour)))).Format(time.RFC3339)
			return []any{
				i,
				rng.Intn(50_000) + 1,
				rng.Intn(10_000) + 1,
				rng.Intn(10) + 1,
				float64(rng.Intn(100000)+100) / 100.0,
				statuses[rng.Intn(len(statuses))],
				ts, ts,
			}
		},
	)
}

func seedProducts(db *sql.DB, rng *rand.Rand, n int) {
	base := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	bulkInsert(db, n, 10_000, "products",
		"id,name,category,price,stock,created_at",
		func(i int) []any {
			ts := base.Add(time.Duration(rng.Int63n(int64(730*24*time.Hour)))).Format(time.RFC3339)
			return []any{
				i,
				fmt.Sprintf("Product-%07d", i),
				categories[rng.Intn(len(categories))],
				float64(rng.Intn(50000)+100) / 100.0,
				rng.Intn(1000),
				ts,
			}
		},
	)
}

func seedAuditLogs(db *sql.DB, rng *rand.Rand, n int) {
	actions := []string{"create", "update", "delete", "view"}
	entities := []string{"order", "product", "user"}
	base := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	bulkInsert(db, n, 10_000, "audit_logs",
		"id,entity,entity_id,action,created_at",
		func(i int) []any {
			ts := base.Add(time.Duration(rng.Int63n(int64(365*24*time.Hour)))).Format(time.RFC3339)
			return []any{
				i,
				entities[rng.Intn(len(entities))],
				rng.Intn(100_000) + 1,
				actions[rng.Intn(len(actions))],
				ts,
			}
		},
	)
}

// bulkInsert inserts n rows into table, committing every txSize rows.
func bulkInsert(db *sql.DB, n, txSize int, table, cols string, row func(i int) []any) {
	// Enable WAL + relaxed sync for fast inserts.
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")

	// Count placeholders from cols.
	count := 1
	for _, c := range cols {
		if c == ',' {
			count++
		}
	}
	ph := "?"
	for i := 1; i < count; i++ {
		ph += ",?"
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, cols, ph)

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(sql)
	for i := 1; i <= n; i++ {
		stmt.Exec(row(i)...)
		if i%txSize == 0 {
			tx.Commit()
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare(sql)
			log.Printf("  %s: %d / %d", table, i, n)
		}
	}
	tx.Commit()
	log.Printf("  %s: %d / %d (done)", table, n, n)
}

// ---- mutations ------------------------------------------------------------

// mutateDev applies targeted updates to dev so the diff produces meaningful output.
func mutateDev(dev *sql.DB, rng *rand.Rand, oRows, pRows int) {
	log.Println("Applying mutations to dev.db …")

	// 10 order amounts updated (simulating price adjustments after checkout).
	for i := 0; i < 10; i++ {
		id := rng.Intn(oRows) + 1
		newAmt := float64(rng.Intn(100000)+100) / 100.0
		ts := time.Now().UTC().Format(time.RFC3339)
		if _, err := dev.Exec(`UPDATE orders SET amount=?, updated_at=? WHERE id=?`, newAmt, ts, id); err != nil {
			log.Printf("warn: mutate order %d: %v", id, err)
		}
	}

	// 5 product prices updated.
	for i := 0; i < 5; i++ {
		id := rng.Intn(pRows) + 1
		newPrice := float64(rng.Intn(50000)+100) / 100.0
		if _, err := dev.Exec(`UPDATE products SET price=? WHERE id=?`, newPrice, id); err != nil {
			log.Printf("warn: mutate product %d: %v", id, err)
		}
	}

	// 3 new orders that exist only in dev.
	ts := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 3; i++ {
		id := oRows + 1 + i
		dev.Exec(`INSERT INTO orders (id,customer_id,product_id,quantity,amount,status,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			id, rng.Intn(50_000)+1, rng.Intn(10_000)+1, 1, 99.99, "pending", ts, ts,
		)
	}

	log.Println("  10 order amounts changed, 5 product prices changed, 3 new orders in dev")
}

// ---- helpers --------------------------------------------------------------

func mustOpen(path string) *sql.DB {
	// Remove stale database so the seed is always deterministic.
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	return db
}
