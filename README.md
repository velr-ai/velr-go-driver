# Velr for Go

Velr is an embedded property-graph database queried with openCypher. This module
includes the database runtime and a synchronous Go API for memory and local-file
databases.

## Requirements and installation

- Go 1.22 or newer, cgo enabled, and a C compiler for your target.
- Bundled runtimes: macOS arm64/amd64, Linux arm64/amd64 with glibc, and Windows
  amd64. Linux musl environments, including Alpine, are unsupported by these bundles.
- A writable runtime cache that permits native-library loading, or a preinstalled
  compatible library selected with `VELR_RUNTIME_PATH`.
- `SEARCH` and creation options below require Velr runtime 0.2.62 or newer.
  Use a driver and runtime from the same release.

Run installation from your application's Go module. For a new module, first run
`go mod init example.com/velr-example` in its project directory:

```sh
go get github.com/velr-ai/velr-go-driver@latest
```

## Quick start

Save this as `main.go` in the application module and run `go run .`:

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

    if err := db.RunWithParams("CREATE (:Person {name: $name, born: $born})", velr.Params{
        "name": "Keanu Reeves", "born": 1964,
    }); err != nil {
        log.Fatal(err)
    }
    rows, err := db.QueryWithParams(
        "MATCH (p:Person {name: $name}) RETURN p.name AS name, p.born AS born",
        velr.Params{"name": "Keanu Reeves"},
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(rows[0]["name"], rows[0]["born"]) // Keanu Reeves 1964
}
```

Subsequent snippets can be placed inside `main`. They reuse the `velr`, `fmt`,
and `log` imports above and an open connection named `db`, unless they open one
explicitly. Close the connection when `main` finishes.

## Queries and results

### In-memory, persistent, and read-only databases

`OpenInMemory()` creates a private ephemeral database. `Open(path)` opens or
creates a persistent database; its parent directory must exist:

```go
db, err := velr.Open("graph.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

`OpenReadonly` requires an existing supported database and never creates,
initializes, migrates, or repairs it:

```go
reader, err := velr.OpenReadonly("graph.db")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()
rows, err := reader.Query("MATCH (n) RETURN count(n) AS count")
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows)
```

### Create a database with explicit options

`Create(path, options)` creates a new database and rejects an existing file;
`CreateInMemory(options)` creates a private in-memory database. To choose a
creation-time relationship property selection, pass zero or one case-sensitive,
nonempty name:

```go
configured, err := velr.CreateInMemory(velr.CreateOptions{
    InlineEdgeProperties: []string{"weight"},
})
if err != nil {
    log.Fatal(err)
}
defer configured.Close()
if err := configured.RunWithParams("CREATE ()-[:LINK {weight:$weight}]->()",
    velr.Params{"weight": 0.5}); err != nil {
    log.Fatal(err)
}
names, err := configured.InlineEdgeProperties()
if err != nil {
    log.Fatal(err)
}
fmt.Println(names) // [weight]
```

The selection applies across relationship types, persists when a file is reopened,
and cannot be changed after creation. Read and write every value through ordinary Cypher.
Use `CreateOptions{}` for the default empty selection.

### Choose a query method and bind parameters

Connections and transactions expose the same query choices:

| Method | Use it for | Result ownership |
|---|---|---|
| `Query` | Collect exactly one table as `[]map[string]any` | Automatically closes its table |
| `Run` | Writes or scripts whose rows you do not need | Drains and closes results |
| `ExecOne` | Exactly one table with metadata and row access | Close the table |
| `Exec` | Scripts that may produce multiple tables | Close the stream and each table |

Bind `$name` through `Params` keys without `$`. Values accept `nil`, booleans,
integers fitting `int64`, finite floats, strings, `json.RawMessage`, nested
lists/arrays, and maps with string keys. Convert structs to maps or
`json.RawMessage`. This example creates nodes from records and collects a filtered result:

```go
if err := db.RunWithParams(
    "UNWIND $people AS p CREATE (:Person {name:p.name, age:p.age})",
    velr.Params{"people": []map[string]any{
        {"name": "Alice", "age": 42}, {"name": "Bob", "age": 17},
    }},
); err != nil {
    log.Fatal(err)
}
rows, err := db.QueryWithParams(
    "MATCH (p:Person) WHERE p.age >= $minAge RETURN p.name AS name, p.age AS age",
    velr.Params{"minAge": 18},
)
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows) // [map[age:42 name:Alice]]
```

`Query` collects all rows in memory. `Cell.Value()` produces `nil`, `bool`,
`int64`, `float64`, strings, or decoded JSON. JSON numbers use `float64`; use a
`json.Decoder` with `UseNumber` on `Cell.Bytes` when exact JSON integers matter.
`AsProperty()` decodes nodes, relationships, paths, and spatial values;
`AsPropertyValue()` returns typed property values. Use unique result aliases:
Cypher rejects duplicate result column names.

### Stream tables and read typed rows

For scripts, advance tables with `NextTable`; `nil` means exhaustion:

```go
stream, err := db.Exec("RETURN 1 AS one; RETURN 2 AS two")
if err != nil {
    log.Fatal(err)
}
defer stream.Close()
for {
    table, err := stream.NextTable()
    if err != nil {
        log.Fatal(err)
    }
    if table == nil {
        break
    }
    rows, err := table.ToObjects()
    table.Close()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(rows)
}
```

For large results, open a cursor instead of collecting an array. `NextInto`
scans into matching destination types and returns `false` at exhaustion:

```go
table, err := db.ExecOne("RETURN 'Alice' AS name, 42 AS age")
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
var age int64
ok, err := rows.NextInto(&name, &age)
if err != nil {
    log.Fatal(err)
}
fmt.Println(ok, name, age) // true Alice 42
```

`Rows.Next` returns cells; `velr.Scan` and `Cell.AssignTo` support typed access.
`Table.ForEachRow` closes its temporary cursor even when the callback fails.
Keep producing connections, transactions, and streams alive while reading.
Close cursors before tables, tables before streams, and results before their
transaction/connection. Cells own their Go byte slices; collected Go values
remain usable after handles close.

### Row caps

Pass at most one `QueryOptions` value. Combine settings using its helpers:

```go
rows, err := db.Query(
    "UNWIND $values AS n RETURN n",
    velr.WithParams(velr.Params{"values": []int{1, 2, 3}}).WithMaxResultRows(2),
)
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows) // [map[n:1] map[n:2]]
```

Caps apply per table; `0` preserves metadata without rows. Existing Cypher
`LIMIT` clauses still apply. Caps do not prevent writes or bound execution time.

## Transactions

### Commit on success and use savepoints

`Transaction` commits when its callback returns `nil` and rolls back on an error
or panic. A savepoint lets you discard part of the work:

```go
err := db.Transaction(func(tx *velr.Tx) error {
    if err := tx.Run("CREATE (:Event {name:'kept'})"); err != nil {
        return err
    }
    savepoint, err := tx.Savepoint()
    if err != nil {
        return err
    }
    defer savepoint.Close()
    if err := tx.Run("CREATE (:Event {name:'discarded'})"); err != nil {
        return err
    }
    return savepoint.Rollback()
})
if err != nil {
    log.Fatal(err)
}
rows, err := db.Query("MATCH (e:Event) RETURN e.name AS name")
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows) // [map[name:kept]]
```

For manual control, `BeginTx` returns a transaction with `Commit`, `Rollback`,
and `Close`. Closing an uncommitted transaction rolls it back; commit/rollback
consumes it. Finish result handles before ending a transaction.

`Savepoint.Release` keeps changes; `Rollback` discards changes after the
savepoint and releases it. Closing an active handle also rolls back and releases
it, including named handles from `SavepointNamed`. Named savepoints release
newest first. `RollbackTo(name)` retains its target and discards newer named
savepoints. Commit releases surviving named savepoints automatically.

## Errors and concurrency

### Deadlines, cancellation, and errors

`RunContext`, `ExecContext`, `ExecOneContext`, and `QueryContext` accept a
`context.Context` covering execution and fetching. Add `context`, `errors`, and
`time` to your imports for this example:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
rows, err := db.QueryContext(ctx,
    "UNWIND range(1, 100) AS n RETURN n", velr.MaxResultRows(3))
if err != nil {
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        log.Print("query stopped: ", err)
    } else {
        log.Fatal(err)
    }
} else {
    fmt.Println(rows) // [map[n:1] map[n:2] map[n:3]]
}
```

For a caller-controlled stop, derive a context with `context.WithCancel` and
invoke its cancel function when needed, including from another goroutine. Keep
contexts active until results are consumed or closed. If interruption ends an
explicit transaction, start a new transaction.

Check errors both when starting queries and while advancing tables or rows.
Use `errors.As(err, &queryErr)` with `var queryErr *velr.Error` to inspect its
numeric `Code` and diagnostic `Message`; invalid Go arguments can return
ordinary Go errors. A connection and its handles must not be used concurrently
by multiple goroutines. Use separate connections for parallel work. Explicit
`Close` is idempotent.

## Features and integrations

### SEARCH

Use `MATCH` or `OPTIONAL MATCH` with `SEARCH` through the normal query methods.
Search indexes require a file-backed database and a mandatory `LIMIT` inside the
search parentheses. Bind query values and limits through `Params`. `SCORE AS
score` exposes a match's score; `ORDER BY score DESC` explicitly orders the rows.

`LIMIT` accepts integers from `0` through `2147483647`; fulltext `SKIP`/`OFFSET`
accepts integers from `0` through `9223372036854775807`. Pagination expressions
can use constants and parameters, but cannot depend on query variables. Query
values can refer to earlier variables, including correlated `EXISTS { ... }`
and `COUNT { ... }` subqueries.

Vector index names must be identifiers; use backticks for names containing
spaces. Fulltext index names also accept string parameters. Search patterns
contain one node or one relationship hop, with only the searched element named
or constrained. For relationship indexes, use `MATCH ()-[r]->()` with `SEARCH r`.
Null queries and zero limits produce no matches; `OPTIONAL MATCH` preserves the
input row with null entity and score values. Use a fresh name for `SCORE AS`.

### Fulltext search

Search indexes require a file-backed database such as the `Open("graph.db")`
connection above. Seed a paper, create an index, and query it:

```go
if err := db.RunWithParams(
    "MERGE (p:Paper {id:'paper-1'}) SET p.title=$title, p.abstract=$abstract",
    velr.Params{"title": "Graph databases", "abstract": "Connected data and vector search"},
); err != nil {
    log.Fatal(err)
}
if err := db.Run(`CREATE FULLTEXT INDEX paperText IF NOT EXISTS
    FOR (n:Paper) ON EACH [n.title, n.abstract]`); err != nil {
    log.Fatal(err)
}
rows, err := db.QueryWithParams(`
    MATCH (node:Paper)
    SEARCH node IN (FULLTEXT INDEX $index FOR $text
        WITH ANALYZER 'standard' OFFSET $offset LIMIT $limit)
    SCORE AS score RETURN node.title AS title, score ORDER BY score DESC
`, velr.Params{"index": "paperText", "text": "title:graph", "offset": 0, "limit": 10})
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows)
```

Search text supports terms, phrases, property scopes, boolean grouping,
prefixes, and boosts. Scores rank results within a query; they are not
normalized or comparable across queries. Writes update indexes automatically.
`WITH ANALYZER` selects `standard` (default), `keyword`, `whitespace`, `simple`,
or `english`; `standard-no-stop-words` and `default` are aliases for `standard`.
An enclosing `WHERE` filters the selected page; fulltext has no inner `WHERE`.

### Vector search

Store precomputed embeddings as numeric vector/list properties and index them:

```go
if err := db.RunWithParams(
    "MERGE (n:Chunk {id:'chunk-1'}) SET n.text=$text, n.embedding=$embedding, n.category=$category",
    velr.Params{"text": "Connected data", "embedding": []float64{0.12, 0.34, 0.56}, "category": "guide"},
); err != nil {
    log.Fatal(err)
}
if err := db.Run(`CREATE VECTOR INDEX chunkEmbedding IF NOT EXISTS
    FOR (n:Chunk) ON (n.embedding) WITH [n.category]
    OPTIONS {indexConfig: {dimensions: 3, metric: 'cosine'}}`); err != nil {
    log.Fatal(err)
}
rows, err := db.QueryWithParams(`
    MATCH (node:Chunk)
    SEARCH node IN (VECTOR INDEX chunkEmbedding FOR $embedding
        WHERE node.category = $category LIMIT $limit)
    SCORE AS score RETURN node.text AS text, score ORDER BY score DESC
`, velr.Params{"embedding": []float64{0.10, 0.30, 0.50}, "category": "guide", "limit": 10})
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows)
```

Vectors must match the configured dimensions; cosine query vectors must be
nonzero. Cosine `SEARCH` scores use `(1 + cosine) / 2`; Euclidean scores use
`1 / (1 + squaredDistance)`. Both range from 0 to 1, with higher scores better.
Dot-product indexes use unbounded dot scores. Exact scoring without an index
is available as `vector.similarity.cosine(left, right)` in Cypher.

Declare filter properties with index DDL `WITH [...]`. An inner `WHERE` filters
before nearest-neighbor selection; an enclosing `WHERE`, before or after
`SEARCH`, filters the selected matches and can return fewer than the limit.
Inner filters support property comparisons, `IN`, boolean and null tests joined
by `AND`. Each property permits one lower and one upper bound, or one equality,
`IN`, or null test. Use the enclosing `WHERE` for other predicates. Vector search
does not accept inner `SKIP`, `OFFSET`, or `WITH ANALYZER`.

For text embeddings, `RegisterVectorEmbedder(name, callback)` supplies one finite
`[]float32` per `VectorEmbeddingInput`, in order and with the requested
dimensions. Reference its name with `embedder: 'name'` in an `ON EACH [...]`
index. Do not call the same connection from the callback. See the
[vector guide](GUIDE.md#vector-search) for callback inputs and tuning.

### Compatible procedure calls

Existing fulltext and vector procedures remain supported, including
`queryRelationships` for relationship indexes. Run these queries against the
file-backed database used above:

```cypher
CALL db.index.fulltext.queryNodes('paperText', $text, {limit: 10})
YIELD node, score RETURN node, score ORDER BY score DESC
```

```cypher
CALL db.index.vector.queryNodes('chunkEmbedding', 10, $embedding)
YIELD node, score RETURN node, score ORDER BY score DESC
```

Fulltext procedure scores use the same scale as `SEARCH`; without an explicit
options limit they return at most 1,000 matches. Vector procedures retain raw
cosine or negative squared Euclidean scores, so their scores differ from
`SEARCH`. Registered embedders also accept text queries through vector procedures.

### Arrow import and export

Export owned IPC file/Feather v2 bytes and bind them as queryable input:

```go
table, err := db.ExecOne("RETURN 'Alice' AS name, 42 AS age")
if err != nil {
    log.Fatal(err)
}
defer table.Close()
ipc, err := table.ToArrowIPC()
if err != nil {
    log.Fatal(err)
}
if err := db.BindArrowIPC("_people", ipc); err != nil {
    log.Fatal(err)
}
rows, err := db.Query("UNWIND BIND('_people') AS row RETURN row.name AS name, row.age AS age")
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows) // [map[age:42 name:Alice]]
```

IPC streams are unsupported. Input bytes are borrowed only during binding;
exported bytes outlive the table. Exports can materialize all rows, and column
types are inferred per result: numeric columns may be integers/floats, mixed
incompatible values become strings, and lists/maps/graph values become JSON
UTF-8 strings rather than nested Arrow arrays. Nulls remain null;
empty/all-null columns are UTF-8. Inspect the exported schema.

`BindArrow` and `BindArrowChunks` accept C Data Interface columns from a Go Arrow
implementation. Array release ownership transfers during the native call, even
on failure; schemas/names are borrowed during the call. Consult their API
contracts before passing pointers. No Go Arrow library is needed for the IPC
byte methods above.

### Inspect indexes, graph shape, and query plans

```go
indexes, err := db.Query(`SHOW INDEXES YIELD name, type, entityType, properties
    RETURN name, type, entityType, properties ORDER BY name`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(indexes)
labels, err := db.Query("CALL db.labels() YIELD label RETURN label ORDER BY label")
if err != nil {
    log.Fatal(err)
}
fmt.Println(labels)
shape, err := db.Query(`SHOW CURRENT GRAPH SHAPE
    YIELD element_kind, element_name, property_name, observed_type
    RETURN element_kind, element_name, property_name, observed_type`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(shape)
trace, err := db.Explain("MATCH (p:Paper) RETURN count(p) AS papers")
if err != nil {
    log.Fatal(err)
}
defer trace.Close()
plan, err := trace.CompactString()
if err != nil {
    log.Fatal(err)
}
fmt.Println(plan)
```

Graph shape describes stored data rather than declaring a schema. `Explain`
does not execute a query; `ExplainAnalyze` executes it, including writes.
`Snapshot` returns owned structured data usable after the trace closes.
`SHOW INDEXES` lists range, fulltext, and vector indexes. `db.labels()`,
`db.relationshipTypes()`, and `db.propertyKeys()` list the graph's labels,
relationship types, and property names.

## Compatibility and maintenance

Opening never migrates a database automatically. New databases use schema
version 11. Databases at schema versions 3 through 9 require explicit migration
on a writable connection before graph queries or transactions can run. After
migration, reopen any read-only connections. Databases created before Velr
0.2.14 must be recreated by importing the source data.

```go
needsMigration, err := db.NeedsMigration()
if err != nil {
    log.Fatal(err)
}
if needsMigration {
    report, err := db.Migrate()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(report.Status, report.FromVersion, report.ToVersion)
}
if err := db.Vacuum(); err != nil {
    log.Fatal(err)
}
```

Migration and vacuum require a writable connection outside a transaction;
vacuum also requires the current schema. Their Cypher equivalents are
`MIGRATE DATABASE` and `VACUUM DATABASE`. Migrate supported older databases
before inspecting their graph shape.

The runtime is extracted on first use into the OS user cache. Set
`VELR_CACHE_DIR` to a writable location that permits native-library loading.
For a deployment without a writable cache, supply a compatible decompressed
library and set `VELR_RUNTIME_PATH` to its absolute path before first use.
See [deployment configuration](GUIDE.md#deployment-configuration).

## Reference and support

- [Go API reference](https://pkg.go.dev/github.com/velr-ai/velr-go-driver), also
  available with `go doc github.com/velr-ai/velr-go-driver`.
- [Query language reference](QUERY_LANGUAGE.md): syntax, functions, search grammar, and restrictions.
- [Extended driver guide](GUIDE.md): additional examples and tuning options.
- [Runnable Go examples](https://github.com/velr-ai/velr-go-examples).
- [Community questions](https://github.com/velr-ai/velr/discussions) and
  [bug reports or feature requests](https://github.com/velr-ai/velr/issues).

## License

The binding source is [MIT licensed](LICENSE). The bundled native runtime is
licensed separately under [LICENSE.runtime](LICENSE.runtime), which permits use
and redistribution of unmodified binaries under its terms.
