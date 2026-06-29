package velr

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/cdata"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/arrow/memory/mallocator"
)

func TestSmokeFullAPI(t *testing.T) {
	clearRuntimeOverrides(t)

	t.Run("core query and result APIs", func(t *testing.T) {
		db := openSmokeDB(t)
		defer db.Close()

		assertMigrationAPI(t, db)

		mustRun(t, db.RunWithParams("CREATE (:Person {name: $name, age: $age, active: $active})", Params{
			"name":   "Alice",
			"age":    42,
			"active": true,
		}))
		mustRun(t, db.Run("CREATE (:Person {name:'Bob', age:30, active:false})"))
		mustRun(t, db.Run("MATCH (a:Person {name:'Alice'}), (b:Person {name:'Bob'}) CREATE (a)-[:KNOWS {since:2024}]->(b)"))

		rows, err := db.Query(
			"MATCH (p:Person) RETURN p.name AS name, p.age AS age ORDER BY age",
			MaxResultRows(1),
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["name"] != "Bob" || rows[0]["age"] != int64(30) {
			t.Fatalf("max-row query returned %#v", rows)
		}

		rows, err = db.QueryWithParams(
			"MATCH (p:Person {name:$name}) RETURN p.age AS age, p.active AS active",
			Params{"name": "Alice"},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["age"] != int64(42) || rows[0]["active"] != true {
			t.Fatalf("parameterized query returned %#v", rows)
		}

		mustRun(t, db.RunWithParams("RETURN $payload AS payload", Params{
			"payload": []any{int64(1), "two", map[string]any{"ok": true}},
		}))
		err = db.RunWithParams("RETURN $bad AS bad", Params{"bad": struct{ A int }{A: 1}})
		if err == nil || !strings.Contains(err.Error(), "unsupported query parameter $bad") {
			t.Fatalf("expected unsupported struct param error, got %v", err)
		}

		table := mustExecOne(t, db, "RETURN true AS ok, 42 AS age, 3.5 AS score, 'Alice' AS name, null AS missing")
		assertColumnNames(t, table, []string{"ok", "age", "score", "name", "missing"})
		assertColumnCount(t, table, 5)
		cursor := mustRows(t, table)
		var ok bool
		var age int64
		var score float64
		var name string
		var missing any
		has, err := cursor.NextInto(&ok, &age, &score, &name, &missing)
		if err != nil {
			t.Fatal(err)
		}
		if !has || !ok || age != 42 || score != 3.5 || name != "Alice" || missing != nil {
			t.Fatalf("NextInto decoded ok=%v age=%d score=%v name=%q missing=%#v", ok, age, score, name, missing)
		}
		has, err = cursor.NextInto(&ok, &age, &score, &name, &missing)
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatal("expected EOF from NextInto")
		}
		mustRun(t, cursor.Close())
		mustRun(t, table.Close())

		table = mustExecOne(t, db, "MATCH (p:Person {name:'Alice'}) RETURN p.name AS name, p.age AS age")
		cells := collectOneRow(t, table)
		var scannedName string
		var scannedAge int64
		if err := Scan(cells, &scannedName, &scannedAge); err != nil {
			t.Fatal(err)
		}
		if scannedName != "Alice" || scannedAge != 42 {
			t.Fatalf("Scan decoded name=%q age=%d", scannedName, scannedAge)
		}
		objects, err := table.ToObjects()
		if err != nil {
			t.Fatal(err)
		}
		if len(objects) != 1 || objects[0]["name"] != "Alice" || objects[0]["age"] != int64(42) {
			t.Fatalf("ToObjects returned %#v", objects)
		}
		mustRun(t, table.Close())

		stream, err := db.Exec(
			"MATCH (p:Person {name:'Alice'}) RETURN p.name AS name; " +
				"MATCH (p:Person {name:'Bob'}) RETURN p.age AS age",
		)
		if err != nil {
			t.Fatal(err)
		}
		first, err := stream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		assertColumnNames(t, first, []string{"name"})
		assertOneObject(t, first, "name", "Alice")
		mustRun(t, first.Close())
		second, err := stream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		assertColumnNames(t, second, []string{"age"})
		assertOneObject(t, second, "age", int64(30))
		mustRun(t, second.Close())
		none, err := stream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		if none != nil {
			t.Fatal("expected exhausted stream")
		}
		mustRun(t, stream.Close())

		assertPropertyDecoding(t, db)
		assertExplainAPI(t, db)
		assertArrowIPCAPI(t, db)
		assertArrowCDataAPI(t, db)
	})

	t.Run("transactions rollback savepoints and tx APIs", func(t *testing.T) {
		db := openSmokeDB(t)
		defer db.Close()

		tx, err := db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Run("CREATE (:Temp {k:'rolled_back_by_close'})"))
		mustRun(t, tx.Close())
		assertCount(t, db, "MATCH (n:Temp {k:'rolled_back_by_close'}) RETURN count(n) AS c", 0)

		mustRun(t, db.Transaction(func(tx *Tx) error {
			return tx.RunWithParams("CREATE (:Temp {k:$k})", Params{"k": "committed_by_helper"})
		}))
		assertCount(t, db, "MATCH (n:Temp {k:'committed_by_helper'}) RETURN count(n) AS c", 1)

		sentinel := errors.New("rollback helper")
		err = db.Transaction(func(tx *Tx) error {
			if err := tx.Run("CREATE (:Temp {k:'rolled_back_by_helper'})"); err != nil {
				return err
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected sentinel error, got %v", err)
		}
		assertCount(t, db, "MATCH (n:Temp {k:'rolled_back_by_helper'}) RETURN count(n) AS c", 0)

		tx, err = db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Run("CREATE (:Temp {k:'scoped_outer'})"))
		sp, err := tx.Savepoint()
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Run("CREATE (:Temp {k:'scoped_inner'})"))
		mustRun(t, sp.Rollback())
		mustRun(t, tx.Commit())
		assertValues(t, db, "MATCH (n:Temp) WHERE n.k STARTS WITH 'scoped_' RETURN n.k AS k ORDER BY k", []string{"scoped_outer"})

		tx, err = db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Run("CREATE (:Temp {k:'named_outer'})"))
		named, err := tx.SavepointNamed("sp_named")
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Run("CREATE (:Temp {k:'named_inner'})"))
		mustRun(t, tx.RollbackTo("sp_named"))
		mustRun(t, tx.Run("CREATE (:Temp {k:'named_after_rollback'})"))
		mustRun(t, tx.ReleaseSavepoint("sp_named"))
		if err := named.Close(); err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.Commit())
		assertValues(t, db, "MATCH (n:Temp) WHERE n.k STARTS WITH 'named_' RETURN n.k AS k ORDER BY k", []string{"named_after_rollback", "named_outer"})

		tx, err = db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		beforeWrite1, err := tx.SavepointNamed("before_write1")
		if err != nil {
			t.Fatal(err)
		}
		_ = beforeWrite1
		mustRun(t, tx.Run("CREATE (:Temp {k:'earlier_a'})"))
		beforeWrite2, err := tx.SavepointNamed("before_write2")
		if err != nil {
			t.Fatal(err)
		}
		_ = beforeWrite2
		mustRun(t, tx.Run("CREATE (:Temp {k:'earlier_b'})"))
		mustRun(t, tx.RollbackTo("before_write1"))
		mustRun(t, tx.Run("CREATE (:Temp {k:'earlier_c'})"))
		mustRun(t, tx.ReleaseSavepoint("before_write1"))
		mustRun(t, tx.Commit())
		assertValues(t, db, "MATCH (n:Temp) WHERE n.k STARTS WITH 'earlier_' RETURN n.k AS k ORDER BY k", []string{"earlier_c"})

		tx, err = db.BeginTx()
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, tx.RunWithParams("CREATE (:Temp {k:$k, score:$score})", Params{"k": "tx_param", "score": 7}))
		txRows, err := tx.QueryWithParams(
			"MATCH (n:Temp {k:$k}) RETURN n.score AS score",
			Params{"k": "tx_param"},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(txRows) != 1 || txRows[0]["score"] != int64(7) {
			t.Fatalf("tx QueryWithParams returned %#v", txRows)
		}
		txTable, err := tx.ExecOneWithParams("MATCH (n:Temp {k:$k}) RETURN n.k AS k", Params{"k": "tx_param"})
		if err != nil {
			t.Fatal(err)
		}
		assertOneObject(t, txTable, "k", "tx_param")
		mustRun(t, txTable.Close())
		txStream, err := tx.Exec("RETURN 1 AS one; RETURN 2 AS two")
		if err != nil {
			t.Fatal(err)
		}
		txFirst, err := txStream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		assertOneObject(t, txFirst, "one", int64(1))
		mustRun(t, txFirst.Close())
		txSecond, err := txStream.NextTable()
		if err != nil {
			t.Fatal(err)
		}
		assertOneObject(t, txSecond, "two", int64(2))
		mustRun(t, txSecond.Close())
		mustRun(t, txStream.Close())
		assertTxExplainAPI(t, tx)
		mustRun(t, tx.Commit())
	})

	t.Run("file backed readonly and close semantics", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "smoke.velr")
		db, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		mustRun(t, db.Run("CREATE (:Persisted {name:'stored'})"))
		mustRun(t, db.Close())
		mustRun(t, db.Close())

		ro, err := OpenReadonly(path)
		if err != nil {
			t.Fatal(err)
		}
		defer ro.Close()
		assertCount(t, ro, "MATCH (n:Persisted {name:'stored'}) RETURN count(n) AS c", 1)
		if err := ro.Run("CREATE (:Persisted {name:'write_should_fail'})"); err == nil {
			t.Fatal("expected read-only write to fail")
		}
	})

	t.Run("vector embedder callback", func(t *testing.T) {
		db := openSmokeFileDB(t, "vector.velr")
		defer db.Close()
		assertVectorEmbedderAPI(t, db)
	})

	t.Run("fulltext index", func(t *testing.T) {
		db := openSmokeFileDB(t, "fulltext.velr")
		defer db.Close()
		assertFulltextAPI(t, db)
	})
}

func clearRuntimeOverrides(t *testing.T) {
	t.Helper()
	t.Setenv("VELR_RUNTIME_PATH", "")
	t.Setenv("VELR_NATIVE_LIBRARY", "")
	t.Setenv("VELR_LIB", "")
}

func openSmokeDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func openSmokeFileDB(t *testing.T, name string) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func mustRun(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustExecOne(t *testing.T, db *DB, cypher string, options ...QueryOptions) *Table {
	t.Helper()
	table, err := db.ExecOne(cypher, options...)
	if err != nil {
		t.Fatal(err)
	}
	return table
}

func mustRows(t *testing.T, table *Table) *Rows {
	t.Helper()
	rows, err := table.Rows()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func assertColumnNames(t *testing.T, table *Table, want []string) {
	t.Helper()
	got, err := table.ColumnNames()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("column names got %#v, want %#v", got, want)
	}
}

func assertColumnCount(t *testing.T, table *Table, want int) {
	t.Helper()
	got, err := table.ColumnCount()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("column count got %d, want %d", got, want)
	}
}

func collectOneRow(t *testing.T, table *Table) []Cell {
	t.Helper()
	rows, err := table.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %d", len(rows))
	}
	return rows[0]
}

func assertOneObject(t *testing.T, table *Table, key string, want any) {
	t.Helper()
	rows, err := table.ToObjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %#v", rows)
	}
	if rows[0][key] != want {
		t.Fatalf("row[%q] got %#v, want %#v in %#v", key, rows[0][key], want, rows[0])
	}
}

func assertCount(t *testing.T, db *DB, cypher string, want int64) {
	t.Helper()
	rows, err := db.Query(cypher)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["c"] != want {
		t.Fatalf("count got %#v, want %d", rows, want)
	}
}

func assertValues(t *testing.T, db *DB, cypher string, want []string) {
	t.Helper()
	rows, err := db.Query(cypher)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		value, ok := row["k"].(string)
		if !ok {
			t.Fatalf("expected string k in %#v", row)
		}
		got = append(got, value)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values got %#v, want %#v", got, want)
	}
}

func assertMigrationAPI(t *testing.T, db *DB) {
	t.Helper()
	schema, err := db.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	current, err := db.CurrentSchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if schema < 0 || current < 0 {
		t.Fatalf("invalid schema versions schema=%d current=%d", schema, current)
	}
	_, err = db.NeedsMigration()
	if err != nil {
		t.Fatal(err)
	}
	report, err := db.Migrate()
	if err != nil {
		t.Fatal(err)
	}
	if report.ToVersion < report.FromVersion {
		t.Fatalf("invalid migration report: %#v", report)
	}
}

func assertPropertyDecoding(t *testing.T, db *DB) {
	t.Helper()
	table := mustExecOne(t, db, "MATCH p=(a:Person {name:'Alice'})-[r:KNOWS]->(b:Person {name:'Bob'}) RETURN a AS node, r AS rel, p AS path")
	row := collectOneRow(t, table)
	mustRun(t, table.Close())

	nodeValue, err := row[0].AsProperty()
	if err != nil {
		t.Fatal(err)
	}
	node, ok := nodeValue.(Node)
	if !ok {
		t.Fatalf("expected Node, got %T %#v", nodeValue, nodeValue)
	}
	if !contains(node.Labels, "Person") || node.Properties["name"].GoValue() != "Alice" || node.Properties["age"].GoValue() != int64(42) {
		t.Fatalf("unexpected node: %#v", node)
	}

	relValue, err := row[1].AsProperty()
	if err != nil {
		t.Fatal(err)
	}
	rel, ok := relValue.(Relationship)
	if !ok {
		t.Fatalf("expected Relationship, got %T %#v", relValue, relValue)
	}
	if rel.Type != "KNOWS" || rel.Properties["since"].GoValue() != int64(2024) {
		t.Fatalf("unexpected relationship: %#v", rel)
	}

	pathValue, err := row[2].AsProperty()
	if err != nil {
		t.Fatal(err)
	}
	path, ok := pathValue.(Path)
	if !ok {
		t.Fatalf("expected Path, got %T %#v", pathValue, pathValue)
	}
	if len(path.Nodes()) != 2 || len(path.Relationships()) != 1 {
		t.Fatalf("unexpected path: %#v", path)
	}

	table = mustExecOne(t, db, "RETURN [1, 'two', true, null] AS xs, '[3][4] citation text' AS cited")
	row = collectOneRow(t, table)
	mustRun(t, table.Close())

	list, err := row[0].AsPropertyValue()
	if err != nil {
		t.Fatal(err)
	}
	if list.Type != PropertyList || len(list.List) != 4 {
		t.Fatalf("expected property list, got %#v", list)
	}
	if list.List[0].GoValue() != int64(1) || list.List[1].GoValue() != "two" || list.List[2].GoValue() != true || list.List[3].GoValue() != nil {
		t.Fatalf("unexpected property list values: %#v", list)
	}
	var scanned PropertyValue
	if err := row[0].AssignTo(&scanned); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scanned.GoValue(), list.GoValue()) {
		t.Fatalf("scanned property list %#v does not match %#v", scanned, list)
	}

	cited, err := row[1].AsPropertyValue()
	if err != nil {
		t.Fatal(err)
	}
	if cited.Type != PropertyString || cited.GoValue() != "[3][4] citation text" {
		t.Fatalf("bracket-looking text decoded incorrectly: %#v", cited)
	}
}

func assertExplainAPI(t *testing.T, db *DB) {
	t.Helper()
	trace, err := db.Explain("MATCH (p:Person) RETURN p.name AS name")
	if err != nil {
		t.Fatal(err)
	}
	assertTrace(t, trace)
	mustRun(t, trace.Close())

	trace, err = db.ExplainAnalyze("MATCH (p:Person) RETURN p.name AS name")
	if err != nil {
		t.Fatal(err)
	}
	assertTrace(t, trace)
	mustRun(t, trace.Close())
}

func assertTxExplainAPI(t *testing.T, tx *Tx) {
	t.Helper()
	trace, err := tx.Explain("MATCH (n:Temp) RETURN n.k AS k")
	if err != nil {
		t.Fatal(err)
	}
	assertTrace(t, trace)
	mustRun(t, trace.Close())

	trace, err = tx.ExplainAnalyze("MATCH (n:Temp) RETURN n.k AS k")
	if err != nil {
		t.Fatal(err)
	}
	assertTrace(t, trace)
	mustRun(t, trace.Close())
}

func assertTrace(t *testing.T, trace *ExplainTrace) {
	t.Helper()
	count, err := trace.PlanCount()
	if err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("expected at least one explain plan, got %d", count)
	}
	snapshot, err := trace.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot) != count {
		t.Fatalf("snapshot plan count got %d, want %d", len(snapshot), count)
	}
	for planIdx, plan := range snapshot {
		for stepIdx, step := range plan.Steps {
			for stmtIdx, stmt := range step.Statements {
				details, err := trace.SQLitePlanDetails(planIdx, stepIdx, stmtIdx)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(details, stmt.SQLitePlanDetails) {
					t.Fatalf("SQLite plan details mismatch: direct=%#v snapshot=%#v", details, stmt.SQLitePlanDetails)
				}
			}
		}
	}
	compactLen, err := trace.CompactLen()
	if err != nil {
		t.Fatal(err)
	}
	compactBytes, err := trace.CompactBytes()
	if err != nil {
		t.Fatal(err)
	}
	if compactLen != len(compactBytes) {
		t.Fatalf("compact length got %d, bytes len %d", compactLen, len(compactBytes))
	}
	var written bytes.Buffer
	n, err := trace.WriteCompact(&written)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(compactBytes)) || !bytes.Equal(written.Bytes(), compactBytes) {
		t.Fatalf("WriteCompact wrote %d/%d bytes", n, len(compactBytes))
	}
	text, err := trace.CompactString()
	if err != nil {
		t.Fatal(err)
	}
	if text != string(compactBytes) {
		t.Fatal("CompactString does not match CompactBytes")
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("empty explain compact string")
	}
}

func assertArrowIPCAPI(t *testing.T, db *DB) {
	t.Helper()
	table := mustExecOne(t, db, "UNWIND [1,2] AS i RETURN CASE i WHEN 1 THEN 'Iris' ELSE 'Jules' END AS name, i + 20 AS age ORDER BY age")
	ipc, err := table.ToArrowIPC()
	if err != nil {
		t.Fatal(err)
	}
	if len(ipc) == 0 {
		t.Fatal("empty Arrow IPC export")
	}
	mustRun(t, table.Close())

	mustRun(t, db.BindArrowIPC("_people_ipc", ipc))
	mustRun(t, db.Run(`
		UNWIND BIND('_people_ipc') AS r
		CREATE (:Imported {name:r.name, age:r.age})
	`))
	assertCount(t, db, "MATCH (n:Imported) RETURN count(n) AS c", 2)

	tx, err := db.BeginTx()
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, tx.BindArrowIPC("_people_ipc_tx", ipc))
	txRows, err := tx.Query(`
		UNWIND BIND('_people_ipc_tx') AS r
		RETURN r.name AS name
		ORDER BY name
	`)
	if err != nil {
		t.Fatal(err)
	}
	if len(txRows) != 2 || txRows[0]["name"] != "Iris" || txRows[1]["name"] != "Jules" {
		t.Fatalf("tx Arrow IPC bind returned %#v", txRows)
	}
	mustRun(t, tx.Rollback())
}

func assertArrowCDataAPI(t *testing.T, db *DB) {
	t.Helper()
	alloc := mallocator.NewMallocator()

	columns, cleanup := exportArrowColumns(t, []string{"name", "b", "i", "f"}, []arrow.Array{
		arrowStringArray(t, alloc, []string{"Alice", "Bob"}),
		arrowBoolArray(t, alloc, []bool{true, false}),
		arrowInt64Array(t, alloc, []int64{123, -7}),
		arrowFloat64Array(t, alloc, []float64{3.75, 2.1}),
	})
	err := db.BindArrow("_people_cdata", columns)
	cleanup()
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, db.Run(`
		UNWIND BIND('_people_cdata') AS r
		CREATE (:ImportedCData {name:r.name, b:r.b, i:r.i, f:r.f})
	`))
	assertArrowPeople(t, db, `
		MATCH (r:ImportedCData)
		RETURN r.name AS name, r.b AS b, r.i AS i, r.f AS f
		ORDER BY name
	`, []arrowPerson{
		{name: "Alice", b: true, i: 123, f: 3.75},
		{name: "Bob", b: false, i: -7, f: 2.1},
	})

	tx, err := db.BeginTx()
	if err != nil {
		t.Fatal(err)
	}
	txColumns, txCleanup := exportArrowColumns(t, []string{"name", "b", "i", "f"}, []arrow.Array{
		arrowStringArray(t, alloc, []string{"Carol", "Dave"}),
		arrowBoolArray(t, alloc, []bool{true, false}),
		arrowInt64Array(t, alloc, []int64{7, -100}),
		arrowFloat64Array(t, alloc, []float64{1.25, 9.0}),
	})
	err = tx.BindArrow("_people_cdata_tx", txColumns)
	txCleanup()
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, tx.Run(`
		UNWIND BIND('_people_cdata_tx') AS r
		CREATE (:ImportedCDataTx {name:r.name, b:r.b, i:r.i, f:r.f})
	`))
	txRows, err := tx.Query(`
		MATCH (r:ImportedCDataTx)
		RETURN r.name AS name, r.b AS b, r.i AS i, r.f AS f
		ORDER BY name
	`)
	if err != nil {
		t.Fatal(err)
	}
	assertArrowPeopleRows(t, txRows, []arrowPerson{
		{name: "Carol", b: true, i: 7, f: 1.25},
		{name: "Dave", b: false, i: -100, f: 9.0},
	})
	mustRun(t, tx.Commit())

	chunks, chunkCleanup := exportArrowChunkColumns(t, []string{"name", "b", "i", "f"}, [][]arrow.Array{
		{
			arrowStringArray(t, alloc, []string{"Eve"}),
			arrowStringArray(t, alloc, []string{"Frank", "Grace"}),
		},
		{
			arrowBoolArray(t, alloc, []bool{true}),
			arrowBoolArray(t, alloc, []bool{false, true}),
		},
		{
			arrowInt64Array(t, alloc, []int64{42}),
			arrowInt64Array(t, alloc, []int64{-1, 64}),
		},
		{
			arrowFloat64Array(t, alloc, []float64{0.5}),
			arrowFloat64Array(t, alloc, []float64{8.25, 4.5}),
		},
	})
	err = db.BindArrowChunks("_people_cdata_chunks", chunks)
	chunkCleanup()
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, db.Run(`
		UNWIND BIND('_people_cdata_chunks') AS r
		CREATE (:ImportedCDataChunk {name:r.name, b:r.b, i:r.i, f:r.f})
	`))
	assertArrowPeople(t, db, `
		MATCH (r:ImportedCDataChunk)
		RETURN r.name AS name, r.b AS b, r.i AS i, r.f AS f
		ORDER BY name
	`, []arrowPerson{
		{name: "Eve", b: true, i: 42, f: 0.5},
		{name: "Frank", b: false, i: -1, f: 8.25},
		{name: "Grace", b: true, i: 64, f: 4.5},
	})

	tx, err = db.BeginTx()
	if err != nil {
		t.Fatal(err)
	}
	txChunks, txChunkCleanup := exportArrowChunkColumns(t, []string{"name", "b", "i", "f"}, [][]arrow.Array{
		{
			arrowStringArray(t, alloc, []string{"Heidi"}),
			arrowStringArray(t, alloc, []string{"Ivan"}),
			arrowStringArray(t, alloc, []string{"Judy", "Mallory"}),
		},
		{
			arrowBoolArray(t, alloc, []bool{true, false}),
			arrowBoolArray(t, alloc, []bool{true, false}),
		},
		{
			arrowInt64Array(t, alloc, []int64{10, 20, 30, 40}),
		},
		{
			arrowFloat64Array(t, alloc, []float64{1.1}),
			arrowFloat64Array(t, alloc, []float64{2.2, 3.3}),
			arrowFloat64Array(t, alloc, []float64{4.4}),
		},
	})
	err = tx.BindArrowChunks("_people_cdata_chunks_tx", txChunks)
	txChunkCleanup()
	if err != nil {
		t.Fatal(err)
	}
	txRows, err = tx.Query(`
		UNWIND BIND('_people_cdata_chunks_tx') AS r
		RETURN r.name AS name, r.b AS b, r.i AS i, r.f AS f
		ORDER BY name
	`)
	if err != nil {
		t.Fatal(err)
	}
	assertArrowPeopleRows(t, txRows, []arrowPerson{
		{name: "Heidi", b: true, i: 10, f: 1.1},
		{name: "Ivan", b: false, i: 20, f: 2.2},
		{name: "Judy", b: true, i: 30, f: 3.3},
		{name: "Mallory", b: false, i: 40, f: 4.4},
	})
	mustRun(t, tx.Rollback())
}

type arrowPerson struct {
	name string
	b    bool
	i    int64
	f    float64
}

func assertArrowPeople(t *testing.T, db *DB, query string, want []arrowPerson) {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	assertArrowPeopleRows(t, rows, want)
}

func assertArrowPeopleRows(t *testing.T, rows []map[string]any, want []arrowPerson) {
	t.Helper()
	if len(rows) != len(want) {
		t.Fatalf("Arrow C Data row count got %d, want %d: %#v", len(rows), len(want), rows)
	}
	for i, row := range rows {
		got := arrowPerson{
			name: row["name"].(string),
			b:    row["b"].(bool),
			i:    row["i"].(int64),
			f:    row["f"].(float64),
		}
		if got != want[i] {
			t.Fatalf("Arrow C Data row %d got %#v, want %#v", i, got, want[i])
		}
	}
}

func exportArrowColumns(t *testing.T, names []string, arrays []arrow.Array) ([]ArrowColumn, func()) {
	t.Helper()
	if len(names) != len(arrays) {
		t.Fatalf("Arrow C Data names=%d arrays=%d", len(names), len(arrays))
	}

	schemas := make([]cdata.CArrowSchema, len(arrays))
	carrays := make([]cdata.CArrowArray, len(arrays))
	columns := make([]ArrowColumn, len(arrays))
	for i, arr := range arrays {
		cdata.ExportArrowArray(arr, &carrays[i], &schemas[i])
		columns[i] = ArrowColumn{
			Name:   names[i],
			Schema: unsafe.Pointer(&schemas[i]),
			Array:  unsafe.Pointer(&carrays[i]),
		}
	}

	cleanup := func() {
		for i := range schemas {
			cdata.ReleaseCArrowSchema(&schemas[i])
		}
		for _, arr := range arrays {
			arr.Release()
		}
	}
	return columns, cleanup
}

func exportArrowChunkColumns(t *testing.T, names []string, chunks [][]arrow.Array) ([]ArrowColumnChunks, func()) {
	t.Helper()
	if len(names) != len(chunks) {
		t.Fatalf("Arrow C Data names=%d chunk columns=%d", len(names), len(chunks))
	}

	allSchemas := make([][]cdata.CArrowSchema, len(chunks))
	allArrays := make([][]cdata.CArrowArray, len(chunks))
	columns := make([]ArrowColumnChunks, len(chunks))
	for i, columnChunks := range chunks {
		allSchemas[i] = make([]cdata.CArrowSchema, len(columnChunks))
		allArrays[i] = make([]cdata.CArrowArray, len(columnChunks))
		arrowChunks := make([]ArrowChunk, len(columnChunks))
		for j, arr := range columnChunks {
			cdata.ExportArrowArray(arr, &allArrays[i][j], &allSchemas[i][j])
			arrowChunks[j] = ArrowChunk{
				Schema: unsafe.Pointer(&allSchemas[i][j]),
				Array:  unsafe.Pointer(&allArrays[i][j]),
			}
		}
		columns[i] = ArrowColumnChunks{Name: names[i], Chunks: arrowChunks}
	}

	cleanup := func() {
		for i := range allSchemas {
			for j := range allSchemas[i] {
				cdata.ReleaseCArrowSchema(&allSchemas[i][j])
			}
		}
		for _, columnChunks := range chunks {
			for _, arr := range columnChunks {
				arr.Release()
			}
		}
	}
	return columns, cleanup
}

func arrowStringArray(t *testing.T, alloc memory.Allocator, values []string) arrow.Array {
	t.Helper()
	builder := array.NewStringBuilder(alloc)
	defer builder.Release()
	builder.AppendValues(values, nil)
	return builder.NewArray()
}

func arrowBoolArray(t *testing.T, alloc memory.Allocator, values []bool) arrow.Array {
	t.Helper()
	builder := array.NewBooleanBuilder(alloc)
	defer builder.Release()
	builder.AppendValues(values, nil)
	return builder.NewArray()
}

func arrowInt64Array(t *testing.T, alloc memory.Allocator, values []int64) arrow.Array {
	t.Helper()
	builder := array.NewInt64Builder(alloc)
	defer builder.Release()
	builder.AppendValues(values, nil)
	return builder.NewArray()
}

func arrowFloat64Array(t *testing.T, alloc memory.Allocator, values []float64) arrow.Array {
	t.Helper()
	builder := array.NewFloat64Builder(alloc)
	defer builder.Release()
	builder.AppendValues(values, nil)
	return builder.NewArray()
}

func assertVectorEmbedderAPI(t *testing.T, db *DB) {
	t.Helper()
	var seen []VectorEmbeddingInput
	err := db.RegisterVectorEmbedder("toy", func(inputs []VectorEmbeddingInput) ([][]float32, error) {
		out := make([][]float32, 0, len(inputs))
		for _, input := range inputs {
			seen = append(seen, input)
			out = append(out, toyVector(input.Text()))
		}
		return out, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	mustRun(t, db.Run(`
		CREATE
		  (:Paper {title:'Alpha Paper', abstract:'alpha graph', rank:7, active:true, published:date('2024-05-01'), tags:['graph','alpha']}),
		  (:Paper {title:'Beta Paper', abstract:'beta graph', rank:8, active:false, published:date('2024-05-02'), tags:['graph','beta']})
	`))
	err = db.Run(`
		CREATE VECTOR INDEX paperEmbedding IF NOT EXISTS
		FOR (n:Paper)
		ON EACH [n.title, n.abstract, n.rank, n.active, n.published, n.tags]
		OPTIONS {
		  indexConfig: {
		    dimensions: 3,
		    metric: 'cosine',
		    embedder: 'toy'
		  }
		}
	`)
	if featureUnavailable(err, "vector-usearch") {
		t.Skipf("bundled runtime does not include vector-usearch: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}

	table := mustExecOne(t, db, `
		CALL db.index.vector.queryNodes('paperEmbedding', 1, 'alpha query')
		YIELD node, score
		RETURN node, score
	`)
	row := collectOneRow(t, table)
	mustRun(t, table.Close())
	nodeValue, err := row[0].AsProperty()
	if err != nil {
		t.Fatal(err)
	}
	node, ok := nodeValue.(Node)
	if !ok {
		t.Fatalf("expected vector result node, got %T %#v", nodeValue, nodeValue)
	}
	if node.Properties["title"].GoValue() != "Alpha Paper" {
		t.Fatalf("unexpected vector node: %#v", node)
	}
	score, err := row[1].AsFloat64()
	if err != nil {
		t.Fatal(err)
	}
	if score <= 0 {
		t.Fatalf("expected positive vector score, got %v", score)
	}
	if !sawVectorPurpose(seen, VectorEmbeddingIndexEntity) || !sawVectorPurpose(seen, VectorEmbeddingQuery) {
		t.Fatalf("vector callback did not see both index and query inputs: %#v", seen)
	}
	if !sawVectorField(seen, "title", PropertyString, "Alpha Paper") {
		t.Fatalf("vector callback did not expose typed title field: %#v", seen)
	}
	if !sawVectorField(seen, "rank", PropertyInt64, int64(7)) {
		t.Fatalf("vector callback did not expose typed rank field: %#v", seen)
	}
	if !sawVectorField(seen, "active", PropertyBool, true) {
		t.Fatalf("vector callback did not expose typed active field: %#v", seen)
	}
	if !sawVectorField(seen, "published", PropertyDate, "2024-05-01") {
		t.Fatalf("vector callback did not expose typed published field: %#v", seen)
	}
	if !sawVectorField(seen, "tags", PropertyList, []any{"graph", "alpha"}) {
		t.Fatalf("vector callback did not expose typed tags field: %#v", seen)
	}
	if !sawUnnamedVectorQuery(seen, "alpha query") {
		t.Fatalf("vector callback did not expose typed query payload: %#v", seen)
	}
}

func assertFulltextAPI(t *testing.T, db *DB) {
	t.Helper()
	mustRun(t, db.Run(`
		CREATE
		  (:Paper {title:'Vector Search', abstract:'graph retrieval with embeddings'}),
		  (:Paper {title:'Planner Notes', abstract:'query planning internals'})
	`))
	err := db.Run(`
		CREATE FULLTEXT INDEX paperText IF NOT EXISTS
		FOR (n:Paper) ON EACH [n.title, n.abstract]
	`)
	if featureUnavailable(err, "fulltext-tantivy") {
		t.Skipf("bundled runtime does not include fulltext-tantivy: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	table := mustExecOne(t, db, `
		CALL db.index.fulltext.queryNodes('paperText', 'title:vector')
		YIELD node, score
		RETURN node, score
	`)
	row := collectOneRow(t, table)
	mustRun(t, table.Close())
	nodeValue, err := row[0].AsProperty()
	if err != nil {
		t.Fatal(err)
	}
	node, ok := nodeValue.(Node)
	if !ok {
		t.Fatalf("expected fulltext result node, got %T %#v", nodeValue, nodeValue)
	}
	if node.Properties["title"].GoValue() != "Vector Search" {
		t.Fatalf("unexpected fulltext node: %#v", node)
	}
	if _, err := row[1].AsFloat64(); err != nil {
		t.Fatal(err)
	}
}

func toyVector(text string) []float32 {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "alpha"):
		return []float32{1, 0, 0}
	case strings.Contains(lower, "beta"):
		return []float32{0, 1, 0}
	default:
		return []float32{0, 0, 1}
	}
}

func sawVectorPurpose(inputs []VectorEmbeddingInput, purpose VectorEmbeddingPurpose) bool {
	for _, input := range inputs {
		if input.Purpose == purpose {
			return true
		}
	}
	return false
}

func sawVectorField(inputs []VectorEmbeddingInput, name string, typ PropertyValueType, want any) bool {
	for _, input := range inputs {
		if input.Purpose != VectorEmbeddingIndexEntity {
			continue
		}
		for _, field := range input.Fields {
			if field.HasName && field.Name == name && field.Value.Type == typ && reflect.DeepEqual(field.Value.GoValue(), want) {
				return true
			}
		}
	}
	return false
}

func sawUnnamedVectorQuery(inputs []VectorEmbeddingInput, want string) bool {
	for _, input := range inputs {
		if input.Purpose != VectorEmbeddingQuery {
			continue
		}
		for _, field := range input.Fields {
			if !field.HasName && field.Value.Type == PropertyString && field.Value.GoValue() == want {
				return true
			}
		}
	}
	return false
}

func featureUnavailable(err error, feature string) bool {
	return err != nil && strings.Contains(err.Error(), "requires the "+feature+" feature")
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
