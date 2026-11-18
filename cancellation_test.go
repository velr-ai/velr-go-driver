package velr

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func requireQueryControl(t *testing.T) {
	t.Helper()
	rt, err := runtimeAPI()
	if err != nil {
		t.Fatal(err)
	}
	if !rt.supportsQueryControl() {
		if os.Getenv("VELR_REQUIRE_QUERY_CONTROL") == "1" ||
			os.Getenv("VELR_RUNTIME_PATH") != "" || os.Getenv("VELR_LIB") != "" {
			t.Fatal("selected runtime does not support query cancellation")
		}
		t.Skip("bundled runtime predates query cancellation")
	}
}

func TestQueryControlDuringExecution(t *testing.T) {
	requireQueryControl(t)
	db, err := OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const query = "UNWIND range(1, 10000) AS a UNWIND range(1, 10000) AS b RETURN sum(a * b)"
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	err = db.RunContext(ctx, query)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("executing query returned %v, want context.DeadlineExceeded", err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	timer := time.AfterFunc(70*time.Millisecond, cancel)
	err = db.RunContext(ctx, query)
	timer.Stop()
	cancel()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("executing query returned %v, want context.Canceled", err)
	}
	if err := db.Run("RETURN 1"); err != nil {
		t.Fatalf("interruption affected the next query: %v", err)
	}
}

func TestQueryControlWhileWaitingForWriter(t *testing.T) {
	requireQueryControl(t)
	path := filepath.Join(t.TempDir(), "cancellation.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	writer, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	tx, err := writer.BeginTx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()

	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(60*time.Millisecond, cancel)
	err = db.RunContext(ctx, "CREATE (:CancelledWrite)")
	timer.Stop()
	cancel()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked write returned %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed >= 3*time.Second {
		t.Fatalf("cancellation took %v, want less than the normal busy timeout", elapsed)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if err := db.RunContext(ctx, "CREATE (:CancelledWrite)"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked write returned %v, want context.DeadlineExceeded", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query("MATCH (n:CancelledWrite) RETURN count(n) AS n")
	if err != nil || len(rows) != 1 || rows[0]["n"] != int64(0) {
		t.Fatalf("write interrupted before lock left unexpected result %v, %v", rows, err)
	}
}

func TestQueryControlThroughDeferredFetch(t *testing.T) {
	requireQueryControl(t)
	db, err := OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("deadline while consuming a stream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		stream, err := db.ExecContext(ctx, "UNWIND [1, 2, 3] AS n RETURN n")
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		table, err := stream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		defer table.Close()
		rows, err := table.Rows()
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		if row, err := rows.Next(); err != nil || len(row) != 1 {
			t.Fatalf("first row = %v, %v", row, err)
		}
		<-ctx.Done()
		// The runtime deadline is relative to creation of the native handle.
		time.Sleep(5 * time.Millisecond)
		if _, err := rows.Next(); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("row fetch returned %v, want context.DeadlineExceeded", err)
		}
	})

	t.Run("cancel while consuming a transaction table", func(t *testing.T) {
		tx, err := db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		table, err := tx.ExecOneContext(ctx, "UNWIND [1, 2, 3] AS n RETURN n")
		if err != nil {
			t.Fatal(err)
		}
		defer table.Close()
		rows, err := table.Rows()
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		if row, err := rows.Next(); err != nil || len(row) != 1 {
			t.Fatalf("first row = %v, %v", row, err)
		}
		cancel()
		if _, err := rows.Next(); !errors.Is(err, context.Canceled) {
			t.Fatalf("transaction row fetch returned %v, want context.Canceled", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	rows, err := db.QueryContext(ctx, "RETURN $answer AS answer", WithParams(Params{"answer": int64(42)}))
	cancel()
	if err != nil || len(rows) != 1 || rows[0]["answer"] != int64(42) {
		t.Fatalf("parameterized query = %v, %v", rows, err)
	}
	if err := db.Run("RETURN 42"); err != nil {
		t.Fatalf("late cancellation affected the next query: %v", err)
	}
}
