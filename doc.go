// Package velr provides Go bindings for the embedded Velr graph database.
//
// The package wraps the Velr native runtime through its public C ABI. The
// module embeds the supported native runtime binaries, materializes the selected
// runtime into the user cache directory on first use, and loads that exact ABI
// version. Applications do not need a Velr source checkout or separate runtime
// install.
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
// Fulltext search is available through Cypher. Vector indexes can use
// RegisterVectorEmbedder to call a synchronous Go VectorEmbedder; callback
// fields carry typed PropertyValue values. Arrow IPC bytes are supported with
// BindArrowIPC and Table.ToArrowIPC; Arrow C Data Interface columns are
// supported with BindArrow and BindArrowChunks. Bound Arrow names are consumed
// from Cypher with UNWIND BIND('name') AS row.
//
// Handles are connection-affine: do not use the same DB, transaction, stream,
// table, row cursor, savepoint, or explain trace concurrently from multiple
// goroutines. Close handles explicitly when finished; finalizers are only a
// leak safety net.
package velr
