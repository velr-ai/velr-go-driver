# Go guide

Task guides supplement the [README](README.md), [Go API reference](https://pkg.go.dev/github.com/velr-ai/velr-go-driver), and [query language reference](QUERY_LANGUAGE.md).

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

### Cancel queries and set deadlines

`DB` and `Tx` provide `RunContext`, `ExecContext`, `ExecOneContext`, and
`QueryContext`. Pass a request context, or derive one with `context.WithTimeout`
to limit the time spent executing and fetching the results.

```go
import (
    "context"
    "time"

    velr "github.com/velr-ai/velr-go-driver"
)

func movieTitles(ctx context.Context, db *velr.DB) ([]map[string]any, error) {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    return db.QueryContext(ctx, "MATCH (m:Movie) RETURN m.title AS title")
}
```

Use `context.WithCancel` if a caller must be able to stop the query. The caller
can invoke the cancel function from another goroutine. For `ExecContext` and
`ExecOneContext`, keep the context active until you consume or close the
stream, table, and rows. Check errors from both the query call and result reads
with `errors.Is(err, context.Canceled)` and
`errors.Is(err, context.DeadlineExceeded)`.

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
discards them, and `Savepoint.Close` rolls back changes made after an active savepoint and releases
it. This also applies to handles returned by `SavepointNamed`; release a
savepoint explicitly when you want to preserve its changes.

## Introspection

`SHOW INDEXES YIELD name, type, properties RETURN name, type, properties ORDER BY name`
lists range, fulltext, and vector indexes. Read labels, relationship types, and
property names with `CALL db.labels()`, `CALL db.relationshipTypes()`, and
`CALL db.propertyKeys()`; their columns are `label`, `relationshipType`, and
`propertyKey` respectively. Run these through normal query methods.

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

## SEARCH

`SEARCH` requires Velr runtime 0.2.62 or newer and a file-backed database. Use
it as a subclause of `MATCH` or `OPTIONAL MATCH` through `Query`, `ExecOne`,
or the equivalent transaction methods. The following sections create indexes
and show complete queries.

```cypher
MATCH (n:Chunk)
SEARCH n IN (VECTOR INDEX chunkEmbedding FOR $embedding
  WHERE n.category = $category LIMIT $limit)
SCORE AS score
RETURN n, score ORDER BY score DESC
```

Declare vector filter properties in index DDL with `WITH [n.category]` after
`ON (n.embedding)`. The inner `WHERE` filters before nearest-neighbor selection;
an enclosing `WHERE`, before or after `SEARCH`, filters the selected candidates.
Inner filters allow `=`, `<`, `<=`, `>`, `>=`, `IN`, boolean/null property tests
joined by `AND`. Per property, use at most one lower plus one upper bound, or
one equality, `IN`, or null test. Other predicates belong in the enclosing `WHERE`.

```cypher
MATCH (n:Paper)
SEARCH n IN (FULLTEXT INDEX $index FOR $text WITH ANALYZER $analyzer
  SKIP $offset LIMIT $limit)
SCORE AS score
RETURN n, score ORDER BY score DESC
```

`LIMIT` is mandatory inside the parentheses and accepts an integer in
`0..2147483647`. Fulltext also supports `SKIP`/`OFFSET` in
`0..9223372036854775807`; vector search does not. Pagination uses static
expressions or parameters, not query variables. Vector index names must be
identifiers; fulltext names can also be string parameters. Query values and
fulltext analyzer names can refer to variables from earlier clauses, but not
to the searched element. Use a fresh name for `SCORE AS`.

Search patterns contain exactly one node or one relationship hop; only the
searched element can be named or constrained. Relationship indexes use
`MATCH ()-[r]->() SEARCH r IN (...)`. Multi-pattern matches, path bindings,
shortest paths, and variable-length hops are unsupported in the search pattern.
Null query values and zero limits yield no matches; `OPTIONAL MATCH` preserves
an input row with null entity/score values. Correlated `EXISTS { ... }` and
`COUNT { ... }` support these searches, including optional-row counting.

Cosine SEARCH scores use `(1 + cosine) / 2`, and Euclidean SEARCH scores use
`1 / (1 + squaredDistance)`, both in `0..1`. Dot scores are unbounded. The
compatible vector `CALL` procedures retain raw cosine and negative squared
Euclidean scores. Fulltext scores use the same query-specific scale in both
forms. Use `ORDER BY score DESC` to order result rows explicitly.

## Fulltext Search

Fulltext and vector indexes require a file-backed database opened with `Open`.
They cannot be created in an `OpenInMemory` database.

Fulltext search is available through normal Cypher execution. Define indexes
with `CREATE FULLTEXT INDEX` and query them with `SEARCH`:

```go
db, err := velr.Open("mygraph.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

if err := db.RunWithParams(
    "MERGE (n:Paper {id:'guide-paper'}) SET n.title=$title, n.abstract=$abstract",
    velr.Params{"title": "Graph databases", "abstract": "Connected data and vector search"},
); err != nil {
    log.Fatal(err)
}
err = db.Run(`
    CREATE FULLTEXT INDEX paperText IF NOT EXISTS
    FOR (n:Paper) ON EACH [n.title, n.abstract]
`)
if err != nil {
    log.Fatal(err)
}

rows, err := db.QueryWithParams(`
    MATCH (node:Paper)
    SEARCH node IN (FULLTEXT INDEX paperText FOR $query LIMIT $limit)
    SCORE AS score RETURN node, score ORDER BY score DESC
`, velr.Params{"query": `abstract:vector OR title:"query planning"`, "limit": 10})
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows)
```

For query-time analysis, add `WITH ANALYZER $analyzer` after `FOR`. Names are
`standard` (default), `keyword`, `whitespace`, `simple`, and `english`;
`standard-no-stop-words` and `default` alias `standard`. Fulltext has no inner
`WHERE`; filter the selected page with the enclosing `MATCH WHERE`.
The compatible procedure form is
`CALL db.index.fulltext.queryNodes('paperText', $query, {limit: 10}) YIELD node, score`.
Without the third argument, the procedure returns at most 1,000 matches.
Scores are query-specific, unnormalized relevance values in both forms;
compare them only within one query's results.

Fulltext indexes are kept up to date by writes. For writable file-backed databases,
Velr repairs missing or corrupt fulltext index data automatically when the
database is opened. Read-only opens do not repair indexes.

## Vector Search

Fulltext and vector indexes require a file-backed database opened with `Open`.
They cannot be created in an `OpenInMemory` database.

Velr supports two vector index shapes.

If your application already computes embeddings, store them as a vector/list
property and index that property. No embedder is needed, and queries pass a
numeric vector/list:

```go
err := db.Run(`
    CREATE (:Chunk {
      id: 'chunk-1',
      text: 'Graph databases store connected data',
      category: 'guide',
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
    WITH [n.category]
    OPTIONS { indexConfig: { dimensions: 3, metric: 'cosine' } }
`)
if err != nil {
    log.Fatal(err)
}

rows, err := db.QueryWithParams(`
    MATCH (node:Chunk)
    SEARCH node IN (VECTOR INDEX chunkEmbedding FOR $embedding
        WHERE node.category = $category LIMIT $limit)
    SCORE AS score RETURN node, score ORDER BY score DESC
`, velr.Params{"embedding": []float64{0.10, 0.30, 0.50}, "category": "guide", "limit": 10})
if err != nil {
    log.Fatal(err)
}
fmt.Println(rows)
```

The compatible procedure form is
`CALL db.index.vector.queryNodes('chunkEmbedding', 10, $embedding) YIELD node, score`.
It retains its original score scale, as described in [SEARCH](#search).

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
| `cache_policy` | `cachePolicy` | Embedder-backed indexes only. `required` retains generated embeddings for recovery; `best_effort` tolerates failures to retain them; `none` requires recomputing embeddings for recovery. Defaults: `required` with `embedder`, `none` without. Stored-vector indexes require `none`. |

HNSW tuning options:

| Option | Integer count | Meaning |
|---|---:|---|
| `` `vector.hnsw.m` `` | `1..512` neighbors | Controls index quality and size. Increasing it can improve recall with higher memory use, disk usage, and build time. |
| `` `vector.hnsw.ef_construction` `` | `1..3200` candidates | Controls index quality and build time. Increasing it can improve recall with slower index creation. |
| `` `vector.hnsw.ef_search` `` | `1..3200` candidates | Controls recall and query latency. Increasing it can improve recall with slower searches. Default: `64`. |

```go
err := db.RegisterVectorEmbedder("toy", func(inputs []velr.VectorEmbeddingInput) ([][]float32, error) {
    out := make([][]float32, len(inputs))
    for i, input := range inputs {
        out[i] = make([]float32, input.Dimensions)
        if input.Text() != "" && input.Dimensions > 0 {
            out[i][0] = 1 // Replace this toy embedding with your model output.
        }
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

On writable file-backed opens, Velr can repair missing or corrupt vector index
data from the stored vectors or from registered embedders, depending on how the
index was created.

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

Exported Arrow column types are inferred for each result. Numeric columns may
be integers or floating-point; incompatible mixed value kinds become strings.
Cypher lists, maps, nodes, relationships, and paths are JSON strings in UTF-8
columns, rather than nested Arrow arrays. Nulls remain null; empty/all-null
columns use UTF-8 strings. Inspect the exported schema before selecting application
data types. Export can materialize all rows and use memory proportional to the
result; the returned bytes remain usable after result handles close.

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

`ExplainAnalyze` executes the query, including writes. Close the returned trace
when finished. Structured APIs expose detailed plans for diagnostic tools;
ordinary applications can use `CompactString`.


## Schema Migration

Velr does not migrate supported older databases automatically on open. Use the
driver migration API, or run `MIGRATE DATABASE`, from maintenance code when you
intend to update the on-disk schema. New databases use schema version 11;
schema versions 3 through 9 require migration on a writable connection before
graph queries or transactions can run. Reopen read-only connections after
migration. Databases created before Velr 0.2.14 require reimporting source data.

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

## Deployment configuration

`Create(path, CreateOptions{InlineEdgeProperties: []string{"weight"}})` creates
a fresh database and rejects an existing file. `CreateInMemory(options)` uses
the same options without a file. Select zero or one nonempty, case-sensitive
relationship property name at creation. The selection applies across
relationship types, persists after reopening, and cannot be changed. All reads/writes
use ordinary Cypher. `db.InlineEdgeProperties()` returns a copy of the selection.
These APIs require runtime 0.2.62 or newer.

The module ships compressed native libraries. Each executable embeds the
library for its Go target. Build with cgo enabled and a C compiler for that
target.

Supported bundled platforms:

- `darwin-arm64`
- `darwin-x64`
- `linux-x64-gnu`
- `linux-arm64-gnu`
- `win32-x64-msvc`

The driver extracts the library on first use and reuses it from the operating
system's user cache directory. Set `VELR_CACHE_DIR` to use a different cache
location. The directory must allow native library loading and be writable when
the library needs to be extracted.

For a deployment that cannot write a runtime cache, decompress the target's
`.gz` library from the module's `runtime` directory during deployment. Install
the resulting `.dylib`, `.so`, or `.dll` in a location that allows native library
loading. Set `VELR_RUNTIME_PATH` to its absolute path before opening a database.
