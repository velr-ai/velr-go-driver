// Package velr provides Go bindings for the embedded Velr graph database.
//
// Velr executes openCypher queries with the syntax and function support
// described in the README's query language reference.
// The module bundles native runtime binaries for supported platforms.
// Applications do not need a Velr source checkout or separate runtime install.
//
// OpenInMemory opens an ephemeral graph. Open opens or creates a file-backed
// graph. OpenReadonly opens an existing graph without creating, initializing,
// migrating, or repairing it. Fulltext and vector indexes require a file-backed
// database. Close the connection only after its result handles are finished.
// Opening does not migrate automatically. Migrate supported older databases
// on a writable connection before graph queries or transactions, then reopen
// read-only connections before querying the migrated file.
//
// Query execution is synchronous. DB.Run discards result tables, DB.Exec returns
// a stream for statements that can produce multiple tables, DB.ExecOne requires
// exactly one table, and DB.Query converts one table to []map[string]any.
// Transaction methods provide the same shape on Tx. QueryOptions carries named
// parameters and an optional max-result-row cap; the WithParams,
// RunWithParams, QueryWithParams, and related helpers are convenience wrappers.
//
// Use DB.RunContext, DB.ExecContext, DB.ExecOneContext, or DB.QueryContext with a
// context.Context to cancel a query or set its deadline. Tx provides the same
// methods. The context covers execution and result fetching. Keep it active
// until you consume or close the results. Use errors.Is with context.Canceled
// or context.DeadlineExceeded to distinguish the two outcomes.
//
// Query and database failures return *Error. Use errors.As to access its Code
// and Message fields. Code is a numeric operation status. For openCypher errors,
// Message identifies the query error category and compile-time or runtime phase,
// followed by diagnostic context. Check errors when starting a query, advancing
// result tables, and fetching rows. Argument validation can also return ordinary
// Go errors.
//
// Result data is exposed as Table, Rows, and Cell. Cell.Value returns plain Go
// values; stricter conversions are available through AsBool, AsInt64,
// AsFloat64, AsString, DecodeJSON, AssignTo, Scan, and Rows.NextInto.
// Cell.AsPropertyValue returns typed scalar/list property values. Cell.AsProperty
// decodes canonical Velr values into richer graph values such as Node,
// Relationship, Path, and GeoJSON. Node and Relationship properties are exposed
// as map[string]PropertyValue. Cell.Value uses float64 for JSON numbers; use a
// json.Decoder with UseNumber on Cell.Bytes when exact JSON integers matter.
//
// Transactions are explicit. DB.Transaction commits when its callback returns
// nil and rolls back when the callback returns an error. BeginTx gives direct
// access to Commit, Rollback, Close, Savepoint, SavepointNamed, RollbackTo, and
// ReleaseSavepoint. Active savepoint handles roll back and release on Close;
// call Release to retain changes, including for named handles.
//
// SchemaVersion, CurrentSchemaVersion, NeedsMigration, and Migrate expose the
// driver migration API. Explain and ExplainAnalyze return ExplainTrace handles
// with both compact and structured plan access.
//
// Fulltext and vector SEARCH is available through Cypher with runtime 0.2.62
// or newer. Use MATCH with SEARCH and SCORE AS, or the compatible fulltext/vector
// CALL procedures. Cosine and Euclidean SEARCH scores use normalized scales;
// vector CALL procedures retain their original score scales.
// Vector indexes can index stored
// vector/list properties directly, or use RegisterVectorEmbedder to call a Go
// VectorEmbedder for text-to-vector search; callback fields carry typed
// PropertyValue values. CREATE VECTOR INDEX ... OPTIONS { indexConfig: ... }
// configures Velr vector indexes. Prefer vector.dimensions and
// vector.similarity_function where they exist; dimensions and metric are
// accepted aliases. embedder names a registered callback for ON EACH [...]
// indexes, embedder_fingerprint stores application model/version metadata, and
// cache_policy controls embedder-output caching with required, best_effort, or
// none. It defaults to required with an embedder and none without one.
// Stored-vector indexes require none. HNSW tuning accepts integer values:
// vector.hnsw.m (1..512), vector.hnsw.ef_construction (1..3200), and vector.hnsw.ef_search
// (1..3200; default 64). Increasing these settings trades memory, disk usage,
// build time, or query latency for improved recall.
// Arrow IPC bytes are supported with BindArrowIPC and Table.ToArrowIPC; Arrow C
// Data Interface columns are supported with BindArrow and BindArrowChunks.
// Bound Arrow names are consumed from Cypher with UNWIND BIND('name') AS row.
//
// Handles are connection-affine: do not use the same DB, transaction, stream,
// table, row cursor, savepoint, or explain trace concurrently from multiple
// goroutines. Close cursors before tables, tables before streams, and remaining
// results before transactions and connections. Cells own their Go data and can
// be kept after closing results. Close is idempotent. Cancellation functions
// may be invoked from another goroutine.
//
// The README links the task guide, query language reference, examples, platform
// requirements, deployment options, and community support.
package velr
