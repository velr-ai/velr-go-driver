# Velr

Velr is an embedded property-graph database from Velr.ai, queried using openCypher.

It runs in-process and is designed for local, embedded, and edge use cases.

This module provides the **Go bindings** for Velr and bundles the database
runtime. It exposes an idiomatic Go API for executing Cypher
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

## Compatibility

### Schema version 11 compatibility

New databases use schema version 11. Databases at schema versions 3 through 9
require an [explicit migration](#schema-migration) before graph queries can run.

Create fulltext and vector indexes with Cypher DDL and query them with `CALL`.

---

## Installation

Install the Go module:

```sh
go get github.com/velr-ai/velr-go-driver@latest
```

The module ships the supported Velr native runtime binaries. Applications do
not need a Velr source checkout, Rust toolchain, or separate runtime package.

Supported bundled platforms:

- `darwin-arm64`
- `darwin-x64`
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
file must already exist and have a supported Velr schema version. After
migrating an older database on a writable connection, reopen any read-only
connections before querying it.

Connections and active handles are not safe for concurrent use. Use one
connection per goroutine when you need parallelism.

---

## Database creation

Use `Create` with `CreateOptions` to select zero or one inline edge property
by its exact name. The selection applies across all relationship types and is
fixed at creation. Empty options and databases newly created by `Open` select
no inline edge property. `Create` returns an error if the destination file
already exists.

Eligible values include integers, floating-point numbers, and strings up to
128 UTF-8 bytes. Read and write all values through ordinary Cypher queries.

```go
db, err := velr.Create("new-graph.db", velr.CreateOptions{
    InlineEdgeProperties: []string{"timestamp"},
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()
if err := db.Run("CREATE (:Person)-[:MET {timestamp: 1700000000}]->(:Person)"); err != nil {
    log.Fatal(err)
}
```

Use `CreateInMemory` for an in-memory database. Call
`db.InlineEdgeProperties()` to inspect the selected names, including after
reopening a file-backed database.

## Schema Migration

Open the database on a writable connection and call `Migrate` or run
`MIGRATE DATABASE` outside an active transaction.

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

## Maintenance

After large deletes or prune operations on a file-backed database, call
`db.Vacuum()` from maintenance code when you want to compact the database file:

```go
if err := db.Run("MATCH (n:Expired) DETACH DELETE n"); err != nil {
    log.Fatal(err)
}
if err := db.Vacuum(); err != nil {
    log.Fatal(err)
}
```

`Vacuum()` must run on a writable current-schema connection outside an open
transaction. The equivalent command for scripts is `VACUUM DATABASE`.

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
savepoint first. `Commit` automatically releases any still-active named
savepoints from newest to oldest so that all surviving transaction changes are
committed.

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

### Regular expression matching

Use the openCypher operator `=~` to match an entire string against a regular
expression. Matching is case-sensitive by default. Use `(?i)` for
case-insensitive matching.

```cypher
RETURN 'alice' =~ '(?i)A.*' AS matched
```

This returns `true`.

The pattern string uses the Rust regex dialect. Lookaround and backreferences
are unsupported. A null or non-string operand yields `null`. An invalid
pattern raises a query error when both operands are strings.
See [supported pattern syntax](https://docs.rs/regex/latest/regex/#syntax).

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

Fulltext queries return up to 1,000 matches by default. Pass a third
argument such as `{limit: 5000}` to change the cap.

Fulltext queries see changes made earlier in the current transaction.
Opening a file-backed database for writing applies pending index updates
and repairs missing or corrupt index data. If the shared index cannot serve
the query's snapshot, Velr builds a temporary index for that search. This
can happen when updates are pending, a concurrent writer has published a
newer index, or a read-only connection opens an index that needs repair.
The temporary build scans the indexed graph.

---

## Vector Search

Velr supports two vector index shapes.

If your application already computes embeddings, store them as a vector/list
property and index that property. No embedder is needed, and queries pass a
numeric vector/list:

```go
err := db.Run(`
    CREATE (:Chunk {
      id: 'chunk-1',
      text: 'Graph databases store connected data',
      embedding: [0.12, 0.34, 0.56]
    })
`)
if err != nil {
    log.Fatal(err)
}

err = db.Run(`
    CREATE VECTOR INDEX chunkEmbedding IF NOT EXISTS
    FOR (n:Chunk)
    ON (n.embedding)
    OPTIONS { indexConfig: { dimensions: 3, metric: 'cosine' } }
`)
if err != nil {
    log.Fatal(err)
}

rows, err := db.Query(`
    CALL db.index.vector.queryNodes('chunkEmbedding', 10, [0.10, 0.30, 0.50])
    YIELD node, score
    RETURN node, score
`)
```

For exact query-time scoring without a native vector index, use the
Neo4j-compatible scalar function `vector.similarity.cosine(left, right)` inside
`WITH`, `WHERE`, `RETURN`, and `ORDER BY`.

For text-to-vector search, register an embedding callback and reference it from
`CREATE VECTOR INDEX`. Velr invokes the callback for index maintenance when
indexed source values change and for text queries passed to
`CALL db.index.vector.queryNodes(...)`.

`OPTIONS { indexConfig: ... }` configures Velr vector indexes. Prefer the
Neo4j-style names where they exist; shorter aliases are accepted for existing
queries and examples.

```cypher
OPTIONS {
  indexConfig: {
    `vector.dimensions`: 384,
    `vector.similarity_function`: 'cosine',
    embedder: 'text',
    cache_policy: 'required',
    `vector.hnsw.ef_search`: 128
  }
}
```

Core options:

| Option | Accepted aliases | Meaning |
|---|---|---|
| `` `vector.dimensions` `` | `dimensions` | Required vector width. Every stored vector, embedder output, and query vector must contain exactly this many values. |
| `` `vector.similarity_function` `` | `metric`, `similarity_function` | Distance function. Use `cosine` for most semantic embeddings, `l2`/`euclidean` for geometric distance, and `dot`/`inner_product`/`max_inner_product` for inner-product retrieval. Default: `cosine`. |
| `embedder` | `velr.embedder` | Name of a registered callback for `ON EACH [...]` indexes. Omit it for stored-vector indexes such as `ON (n.embedding)`. |
| `embedder_fingerprint` | `embedderFingerprint`, `velr.embedder_fingerprint` | Optional model/version metadata stored with the index definition. Velr does not use it to call the embedder. |
| `cache_policy` | `cachePolicy` | Embedder-backed indexes only. `required` stores generated embeddings in the database so index files can be rebuilt without recomputing them; `best_effort` skips cache write errors; `none` stores no generated embedding cache. Defaults: `required` with `embedder`, `none` without. Stored-vector indexes require `none` because the vector property is already the durable source. |

HNSW tuning options:

| Option | Integer count | Meaning |
|---|---:|---|
| `` `vector.hnsw.m` `` | `1..512` neighbors | Maximum graph connectivity per vector. For example, `16` means each vector keeps roughly 16 HNSW neighbor links. Higher values can improve recall on difficult datasets, with more memory, larger index files, and slower builds. |
| `` `vector.hnsw.ef_construction` `` | `1..3200` candidates | Build-time candidate pool per inserted vector. For example, `320` means the builder considers a working set of 320 candidate neighbors while placing each vector. Higher values can improve index quality and recall, usually with slower index creation. |
| `` `vector.hnsw.ef_search` `` | `1..3200` candidates | Query-time candidate pool. For example, `128` means each search keeps a working set of 128 candidate vectors while traversing the HNSW graph. Higher values can improve recall, usually with slower searches. Velr's default is `64`. |

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
`VectorEmbeddingInput.Text()` to join display strings for text embedders.

For file-backed databases, Velr can repair missing or corrupt vector index data
from the stored vectors or from registered embedders, depending on how the
index was created.

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

Runtime failures return `*velr.Error`. Use `errors.As` to access its `Code` and
`Message` fields. `Code` is a numeric operation status. For Cypher errors,
`Message` identifies the query category and compile-time or runtime phase. Query
categories include `SyntaxError`, `TypeError`, `ArgumentError`,
`EntityNotFound`, `ConstraintVerificationFailed`, `SemanticError`,
`ParameterMissing`, `ProcedureNotFound`, and `ArithmeticError`. Diagnostic text
explains the cause and provides context. Driver-side argument validation also
returns ordinary Go errors.

Check errors from query execution and from `Stream.NextTable`,
`TxStream.NextTable`, and `Rows.Next` while consuming results. A returned stream
or table does not mean the query has finished successfully.

```go
rows, err := db.Query("RETURN missing")
if err != nil {
    var queryErr *velr.Error
    if errors.As(err, &queryErr) {
        log.Printf("Velr status %d: %s", queryErr.Code, queryErr.Message)
    } else {
        log.Print(err)
    }
    return
}
fmt.Println(rows)
```

The example uses the standard `errors`, `fmt`, and `log` packages.

Explicitly close `DB`, `Tx`, `Stream`, `TxStream`, `Table`, `Rows`,
`Savepoint`, and `ExplainTrace` values. `Close` is idempotent on all closeable
handle types. Finalizers are present as a leak safety net, but explicit close
keeps native resources deterministic.

---

## Examples

Runnable examples live in
[github.com/velr-ai/velr-go-examples](https://github.com/velr-ai/velr-go-examples).

The examples cover queries, transactions and savepoints, schema
migration and maintenance, read-only open, fulltext search, vector search,
Arrow IPC, and Arrow C Data column binding. They use the published Go module;
you do not need a Velr source checkout.

## License

The Go driver source is licensed under the MIT license in [`LICENSE`](LICENSE).
The bundled native runtime binaries are licensed separately under
[`LICENSE.runtime`](LICENSE.runtime).
