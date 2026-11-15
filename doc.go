// Package velr provides Go bindings for the embedded Velr graph database.
//
// Velr is openCypher compliant and supports additional Cypher features.
// The module bundles native runtime binaries for supported platforms.
// Applications do not need a Velr source checkout or separate runtime install.
//
// OpenInMemory opens an ephemeral graph. Open opens or creates a file-backed
// graph. OpenReadonly opens an existing graph without creating, initializing,
// or migrating it.
//
// Query execution is synchronous. DB.Run discards result tables, DB.Exec returns
// a stream for statements that can produce multiple tables, DB.ExecOne requires
// exactly one table, and DB.Query converts one table to []map[string]any.
// Transaction methods provide the same shape on Tx. QueryOptions carries named
// parameters and an optional max-result-row cap; the WithParams,
// RunWithParams, QueryWithParams, and related helpers are convenience wrappers.
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
// as map[string]PropertyValue.
//
// Transactions are explicit. DB.Transaction commits when its callback returns
// nil and rolls back when the callback returns an error. BeginTx gives direct
// access to Commit, Rollback, Close, Savepoint, SavepointNamed, RollbackTo, and
// ReleaseSavepoint.
//
// SchemaVersion, CurrentSchemaVersion, NeedsMigration, and Migrate expose the
// driver migration API. Explain and ExplainAnalyze return ExplainTrace handles
// with both compact and structured plan access.
//
// Fulltext search is available through Cypher. Vector indexes can index stored
// vector/list properties directly, or use RegisterVectorEmbedder to call a Go
// VectorEmbedder for text-to-vector search; callback fields carry typed
// PropertyValue values. CREATE VECTOR INDEX ... OPTIONS { indexConfig: ... }
// configures Velr vector indexes. Prefer vector.dimensions and
// vector.similarity_function where they exist; dimensions and metric are
// accepted aliases. embedder names a registered callback for ON EACH [...]
// indexes, embedder_fingerprint stores application model/version metadata, and
// cache_policy controls embedder-output caching with required, best_effort, or
// none. It defaults to required with an embedder and none without one.
// Stored-vector indexes require none. HNSW tuning uses vector.hnsw.m (1..512
// neighbor links per vector), vector.hnsw.ef_construction (1..3200 build
// candidates per inserted vector), and vector.hnsw.ef_search (1..3200 query
// candidates; default 64).
// Arrow IPC bytes are supported with BindArrowIPC and Table.ToArrowIPC; Arrow C
// Data Interface columns are supported with BindArrow and BindArrowChunks.
// Bound Arrow names are consumed from Cypher with UNWIND BIND('name') AS row.
//
// Handles are connection-affine: do not use the same DB, transaction, stream,
// table, row cursor, savepoint, or explain trace concurrently from multiple
// goroutines. Close handles explicitly when finished; finalizers are only a
// leak safety net.
package velr
