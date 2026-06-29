# Velr

Velr is an embedded property-graph database from Velr.ai, written in Rust,
built on top of SQLite, and queried using the openCypher language.

It runs in-process and is designed for local, embedded, and edge use cases.

This module provides the **Go bindings** for Velr. It wraps a bundled native
runtime with a C ABI and exposes an idiomatic Go API for executing Cypher
queries, streaming result tables, working with transactions, decoding graph
values, and importing or exporting Arrow data.

For the main Velr public entry point, see
[velr-ai/velr](https://github.com/velr-ai/velr).
For the Velr website, see [velr.ai](https://velr.ai/).
For generated Go API documentation, see
[pkg.go.dev/github.com/velr-ai/velr-go-driver](https://pkg.go.dev/github.com/velr-ai/velr-go-driver).

## Community

- **Community and questions:** [GitHub Discussions](https://github.com/velr-ai/velr/discussions)
- **Bug reports and feature requests:** [GitHub Issues](https://github.com/velr-ai/velr/issues)
- **Go examples:** [velr-go-examples](https://github.com/velr-ai/velr-go-examples)

We'd love to have you join the Velr community.

---

## Release Status

Velr is currently in public **alpha**.

- The Go API is still evolving.
- Velr supports openCypher and passes all positive openCypher TCK tests. Exact
  error semantics are not guaranteed to match other openCypher implementations.

### Schema Version 7 Compatibility

This release's current on-disk schema is version 7. Supported older databases
can be opened with `velr.Open` or `velr.OpenReadonly` without changing the file.
Reads continue to work on those databases, but writes (`CREATE`, `MERGE`, `SET`,
`DELETE`, `DETACH DELETE`, and other mutating queries) are only available after
migrating to the current schema version. This is intentional: migration is an
explicit maintenance operation, not a side effect of opening a database.

Velr is already usable for real workflows and representative use cases, but
rough edges remain and the API is not yet stable.

Fulltext search and vector search are available today through Cypher DDL and
`CALL` syntax. API details may still evolve while Velr remains alpha.

---

## Installation

Install the Go module:

```sh
go get github.com/velr-ai/velr-go-driver@latest
```

The module ships the supported Velr native runtime binaries. Applications do
not need a Velr source checkout, Rust toolchain, or separate runtime package.

Supported bundled platforms:

- `darwin-universal`
- `linux-x64-gnu`
- `linux-arm64-gnu`
- `win32-x64-msvc`

### Licensing In Simple Terms

- The **Go binding source code** in this module is licensed under the MIT
  license in [`LICENSE`](LICENSE).
- The **bundled native runtime binaries** may be used and freely redistributed
  in unmodified form under the terms of [`LICENSE.runtime`](LICENSE.runtime).

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    velr "github.com/velr-ai/velr-go-driver"
)

func main() {
    db, err := velr.OpenInMemory()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    err = db.RunWithParams("CREATE (:Person {name: $name, born: $born})", velr.Params{
        "name": "Keanu Reeves",
        "born": 1964,
    })
    if err != nil {
        log.Fatal(err)
    }

    rows, err := db.Query("MATCH (p:Person) RETURN p.name AS name, p.born AS born")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(rows)
}
```

Open a file-backed database instead of an in-memory database:

```go
db, err := velr.Open("mygraph.db")
```

Open an existing database for reads only:

```go
db, err := velr.OpenReadonly("mygraph.db")
```

`OpenReadonly` never creates, initializes, migrates, or repairs a database. The
file must already exist and have a supported Velr schema version. Older
supported databases, such as schema version 3, 4, 5, or 6 databases opened by a
schema version 7 runtime, remain available for reads. Writes and features that
require the current schema fail with a normal query error until the database is
explicitly migrated.

Connections and active handles are not safe for concurrent use. Use one
connection per goroutine when you need parallelism.

---

## Schema Migration

Velr does not migrate supported older databases automatically on open. Use the
driver migration API, or run `MIGRATE DATABASE`, from maintenance code when you
intend to update the on-disk schema. See the release-status note above for the
schema version 7 read/write compatibility behavior.

```go
db, err := velr.Open("mygraph.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

needs, err := db.NeedsMigration()
if err != nil {
    log.Fatal(err)
}
if needs {
    report, err := db.Migrate()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(report.Status, report.FromVersion, report.ToVersion, report.Steps)
}
```

The equivalent Cypher command is useful for scripts and tools that already work
through query execution:

```go
table, err := db.ExecOne("MIGRATE DATABASE")
if err != nil {
    log.Fatal(err)
}
defer table.Close()
```

---

## Query Execution

Velr exposes four main ways to run Cypher:

- `Run` executes Cypher and drains all result tables.
- `Exec` returns a stream when a statement can produce multiple result tables.
- `ExecOne` returns exactly one table.
- `Query` converts one table into `[]map[string]any`.

Convenience helpers such as `RunWithParams`, `ExecOneWithParams`, and
`QueryWithParams` bind named Cypher parameters:

```go
rows, err := db.QueryWithParams(
    "MATCH (p:Person {name:$name}) RETURN p.born AS born",
    velr.Params{"name": "Keanu Reeves"},
)
```

Query text uses `$name`; map keys omit the leading `$`. Parameters are values,
not Cypher string interpolation.

Use `MaxResultRows` when you want a per-result-table row cap:

```go
rows, err := db.Query(
    "MATCH (p:Person) RETURN p.name AS name ORDER BY name",
    velr.MaxResultRows(100),
)
```

Supported parameter values are `nil`, `bool`, signed and unsigned integers that
fit `int64`, finite floats, `string`, `json.RawMessage`, lists, arrays, and maps
with string keys. Convert structs to maps or explicit `json.RawMessage` before
binding them.

---

## Reading Results

For typed row handling, use `Rows.NextInto` or `velr.Scan`:

```go
table, err := db.ExecOneWithParams(
    "MATCH (p:Person {name:$name}) RETURN p.name, p.born",
    velr.Params{"name": "Keanu Reeves"},
)
if err != nil {
    log.Fatal(err)
}
defer table.Close()

rows, err := table.Rows()
if err != nil {
    log.Fatal(err)
}
defer rows.Close()

var name string
var born int64
ok, err := rows.NextInto(&name, &born)
if err != nil {
    log.Fatal(err)
}
fmt.Println(ok, name, born)
```

Result helpers:

- `Table.ColumnCount` and `Table.ColumnNames` inspect result metadata.
- `Table.Rows` opens a row cursor.
- `Table.ForEachRow`, `Table.Collect`, and `Table.ToObjects` collect results.
- `Rows.Next` returns `[]Cell`; `Rows.NextInto` scans into pointers.
- `Cell.Value` returns plain Go values.
- `Cell.AsBool`, `AsInt64`, `AsFloat64`, `AsString`, `DecodeJSON`, and
  `AssignTo` provide stricter conversions.
- `Cell.AsPropertyValue` returns a typed `velr.PropertyValue` for scalar
  property values and lists.

`Cell.AsProperty` adds richer decoding for Velr graph and property values.
Nodes become `velr.Node`, relationships become `velr.Relationship`, paths
become `velr.Path`, and spatial GeoJSON values become `velr.GeoJSON`.
`Node.Properties` and `Relationship.Properties` are `map[string]velr.PropertyValue`.
Use `PropertyValue.Type` or `PropertyValue.Kind()` for the Velr kind, and
`PropertyValue.GoValue()` when you want ordinary Go values.

---

## Transactions And Savepoints

Use `Transaction` for commit-on-success and rollback-on-error:

```go
err := db.Transaction(func(tx *velr.Tx) error {
    if err := tx.Run("CREATE (:Event {name:'start'})"); err != nil {
        return err
    }

    sp, err := tx.Savepoint()
    if err != nil {
        return err
    }

    if err := tx.Run("CREATE (:Event {name:'discard'})"); err != nil {
        _ = sp.Close()
        return err
    }
    return sp.Rollback()
})
```

Explicit transactions are available with `BeginTx`, `Commit`, `Rollback`, and
`Close`. Closing an uncommitted transaction rolls it back.

Named savepoints use `SavepointNamed`, `RollbackTo`, and `ReleaseSavepoint`.
Named savepoints are stack-like: release the most recently created active named
savepoint first.

`Savepoint.Release` keeps changes after the savepoint, `Savepoint.Rollback`
discards them, and `Savepoint.Close` closes the native handle without an
explicit release or rollback.

---

## Introspection

Use `SHOW CURRENT GRAPH SHAPE` to inspect the observed schema of the graph. It
reports the shape present in stored data: node labels, relationship types,
properties, observed value types, and counts. It is an observed shape surface,
not a declared GQL graph type.

```go
rows, err := db.Query(`
    SHOW CURRENT GRAPH SHAPE
    YIELD element_kind, element_name, property_name, observed_type, owner_count
    WHERE element_kind = 'node_property'
    RETURN element_name, property_name, observed_type, owner_count
`)
```

Use `YIELD` to compose the command with `WHERE` and `RETURN`. Plain
`SHOW CURRENT GRAPH SHAPE` returns the default projection; `YIELD *` exposes the
full current row shape.

---

## OpenCypher Functions

The bundled runtime supports the following openCypher functions and constructors:

Graph and path: `id`, `type`, `labels`, `keys`, `properties`, `length`,
`nodes`, and `relationships`.

Lists and predicates: `size`, `head`, `last`, `tail`, `reverse`, `range`,
`all`, `any`, `none`, and `single`.

Strings and conversion: `coalesce`, `toInteger`, `toString`, `toLower`, `trim`,
`substring`, and `split`.

Numeric: `abs`, `ceil`, `rand`, `sign`, and `sqrt`.

Temporal constructors and clocks: `date`, `time`, `localtime`, `datetime`,
`localdatetime`, `duration`, `datetime.fromepoch`,
`datetime.fromepochmillis`, and the `.realtime`, `.transaction`, and
`.statement` variants for `date`, `time`, `localtime`, `datetime`, and
`localdatetime`.

Aggregates: `count`, `sum`, `avg`, `min`, `max`, `collect`,
`percentileDisc`, and `percentileCont`.

---

## Fulltext Search

Fulltext search is available through normal Cypher execution. Define indexes
with `CREATE FULLTEXT INDEX` and query them with
`CALL db.index.fulltext.queryNodes(...)`.

```go
db, err := velr.Open("mygraph.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

err = db.Run(`
    CREATE FULLTEXT INDEX paperText
    FOR (n:Paper) ON EACH [n.title, n.abstract]
`)
if err != nil {
    log.Fatal(err)
}

rows, err := db.QueryWithParams(`
    CALL db.index.fulltext.queryNodes('paperText', $query)
    YIELD node, score
    RETURN node, score
`, velr.Params{"query": `abstract:vector OR title:"query planning"`})
```

Fulltext indexes use a sidecar next to file-backed databases. The sidecar is
kept up to date by writes and rebuilt on open if it is missing or corrupt.

---

## Vector Search

Register an embedding callback, then reference it from `CREATE VECTOR INDEX`.
Velr invokes the callback for index maintenance when indexed source values
change and for text queries passed to `CALL db.index.vector.queryNodes(...)`.

```go
err := db.RegisterVectorEmbedder("toy", func(inputs []velr.VectorEmbeddingInput) ([][]float32, error) {
    out := make([][]float32, len(inputs))
    for i, input := range inputs {
        out[i] = embed(input.Text(), input.Dimensions)
    }
    return out, nil
})
```

The callback must return one finite `[]float32` embedding per input, with the
exact dimension count requested by the index.

`VectorEmbeddingInput` includes index name, dimensions, purpose, entity kind,
entity id, and selected fields. Each `VectorEmbeddingField` carries a typed
`PropertyValue`; use `field.Value.Type` or `field.Value.Kind()` to inspect the
Velr kind, `field.Value.GoValue()` for ordinary Go values, and
`VectorEmbeddingInput.Text()` to join display strings for toy or local
embedders.

Vector indexes use a sidecar next to file-backed databases.

---

## Arrow

`Table.ToArrowIPC` exports a result table as Arrow IPC file bytes.
`BindArrowIPC` imports Arrow IPC file bytes under a logical table name:

```go
table, err := db.ExecOne("MATCH (m:Movie) RETURN m.title AS title")
if err != nil {
    log.Fatal(err)
}
defer table.Close()

ipc, err := table.ToArrowIPC()
if err != nil {
    log.Fatal(err)
}

if err := db.BindArrowIPC("_movies", ipc); err != nil {
    log.Fatal(err)
}
```

`BindArrow` and `BindArrowChunks` bind Arrow C Data Interface columns from
whichever Go Arrow implementation your application uses. The `ArrowArray`
pointers passed to `BindArrow` or `BindArrowChunks` are transferred to Velr by
the ABI call. Do not release or reuse those arrays after calling; schemas and
column names are borrowed only during the call.

After binding a logical name, read the rows from Cypher with
`UNWIND BIND(...)`:

```go
if err := db.BindArrow("_people", columns); err != nil {
    log.Fatal(err)
}

if err := db.Run(`
    UNWIND BIND('_people') AS row
    CREATE (:Person {name: row.name, age: row.age})
`); err != nil {
    log.Fatal(err)
}
```

The
[arrow_columns example](https://github.com/velr-ai/velr-go-examples/tree/main/arrow_columns)
shows a complete Apache Arrow Go export using `arrow/cdata` and malloc-backed
Arrow buffers.

Transactions also expose `Tx.BindArrowIPC`, `Tx.BindArrow`, and
`Tx.BindArrowChunks`.

---

## Explain

Use `Explain` or `ExplainAnalyze` on `DB` or `Tx` to inspect a query plan:

```go
trace, err := db.Explain("MATCH (n) RETURN count(n)")
if err != nil {
    log.Fatal(err)
}
defer trace.Close()

compact, err := trace.CompactString()
```

For structured access, use `PlanCount`, `PlanMeta`, `StepCount`, `StepMeta`,
`StatementCount`, `StatementMeta`, `SQLitePlanCount`, `SQLitePlanDetail`,
`SQLitePlanDetails`, or `Snapshot`. For compact rendering use `CompactLen`,
`CompactBytes`, `CompactString`, or `WriteCompact`.

---

## Errors And Lifecycle

Runtime failures return `*velr.Error` with a native error code and message.
Driver-side validation uses ordinary Go errors where no native call was made.

Explicitly close `DB`, `Tx`, `Stream`, `TxStream`, `Table`, `Rows`,
`Savepoint`, and `ExplainTrace` values. `Close` is idempotent on all closeable
handle types. Finalizers are present as a leak safety net, but explicit close
keeps native resources deterministic.

---

## Examples

Runnable examples live in
[github.com/velr-ai/velr-go-examples](https://github.com/velr-ai/velr-go-examples).

The examples cover basic queries, transactions and savepoints, schema
migration, read-only open, fulltext search, vector search, Arrow IPC, and Arrow
C Data column binding. They use the published Go module; you do not need a
Velr source checkout.

## License

The Go driver source is licensed under the MIT license in [`LICENSE`](LICENSE).
The bundled native runtime binaries are licensed separately under
[`LICENSE.runtime`](LICENSE.runtime).
