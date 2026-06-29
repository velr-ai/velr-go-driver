package velr

/*
#cgo !windows LDFLAGS: -ldl

#include <stdlib.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <stdio.h>

#ifdef _WIN32
#include <windows.h>
static void *velr_go_open_library(const char *path) { return (void*)LoadLibraryA(path); }
static void *velr_go_symbol(void *handle, const char *name) { return (void*)GetProcAddress((HMODULE)handle, name); }
static const char *velr_go_last_error(void) { return "failed to load Velr native runtime"; }
static void velr_go_close_library(void *handle) { if (handle) FreeLibrary((HMODULE)handle); }
#else
#include <dlfcn.h>
static void *velr_go_open_library(const char *path) { return dlopen(path, RTLD_NOW | RTLD_LOCAL); }
static void *velr_go_symbol(void *handle, const char *name) { return dlsym(handle, name); }
static const char *velr_go_last_error(void) {
	const char *err = dlerror();
	return err ? err : "unknown dynamic loader error";
}
static void velr_go_close_library(void *handle) { if (handle) dlclose(handle); }
#endif

typedef struct velr_db velr_db;
typedef struct velr_stream velr_stream;
typedef struct velr_table velr_table;
typedef struct velr_rows velr_rows;
typedef struct velr_tx velr_tx;
typedef struct velr_sp velr_sp;
typedef struct velr_stream_tx velr_stream_tx;
typedef struct velr_query_params velr_query_params;
typedef struct velr_explain_trace velr_explain_trace;
struct ArrowSchema;
struct ArrowArray;

typedef int velr_code;

enum {
	VELR_GO_OK = 0,
	VELR_GO_EARG = -1,
	VELR_GO_EUTF = -2,
	VELR_GO_ESTATE = -3,
	VELR_GO_EERR = -4
};

enum {
	VELR_GO_NULL = 0,
	VELR_GO_BOOL = 1,
	VELR_GO_INT64 = 2,
	VELR_GO_DOUBLE = 3,
	VELR_GO_TEXT = 4,
	VELR_GO_JSON = 5
};

typedef struct velr_cell {
	int ty;
	int64_t i64_;
	double f64_;
	const uint8_t *ptr;
	size_t len;
} velr_cell;

typedef struct velr_strview {
	const uint8_t *ptr;
	size_t len;
} velr_strview;

typedef struct velr_query_options {
	int has_max_result_rows;
	size_t max_result_rows;
	const velr_query_params *params;
} velr_query_options;

typedef struct velr_migration_report {
	int32_t from_version;
	int32_t to_version;
	int status;
	size_t step_count;
	char *steps;
} velr_migration_report;

typedef enum velr_vector_embedding_purpose {
	VELR_VECTOR_EMBEDDING_INDEX_ENTITY = 0,
	VELR_VECTOR_EMBEDDING_QUERY = 1
} velr_vector_embedding_purpose;

typedef enum velr_vector_entity_kind {
	VELR_VECTOR_ENTITY_NONE = 0,
	VELR_VECTOR_ENTITY_NODE = 1,
	VELR_VECTOR_ENTITY_RELATIONSHIP = 2
} velr_vector_entity_kind;

typedef enum velr_property_value_type {
	VELR_PROPERTY_NULL = 0,
	VELR_PROPERTY_BOOL = 1,
	VELR_PROPERTY_INT64 = 2,
	VELR_PROPERTY_DOUBLE = 3,
	VELR_PROPERTY_STRING = 4,
	VELR_PROPERTY_DATE = 5,
	VELR_PROPERTY_LOCAL_TIME = 6,
	VELR_PROPERTY_ZONED_TIME = 7,
	VELR_PROPERTY_LOCAL_DATETIME = 8,
	VELR_PROPERTY_ZONED_DATETIME = 9,
	VELR_PROPERTY_DURATION = 10,
	VELR_PROPERTY_POINT = 11,
	VELR_PROPERTY_GEOMETRY = 12,
	VELR_PROPERTY_GEOGRAPHY = 13,
	VELR_PROPERTY_LIST = 14,
	VELR_PROPERTY_VECTOR = 15,
	VELR_PROPERTY_BYTES = 16
} velr_property_value_type;

typedef enum velr_storage_value_type {
	VELR_STORAGE_NULL = 0,
	VELR_STORAGE_INT64 = 1,
	VELR_STORAGE_DOUBLE = 2,
	VELR_STORAGE_TEXT = 3,
	VELR_STORAGE_BLOB = 4
} velr_storage_value_type;

typedef struct velr_vector_embedding_field {
	int has_name;
	velr_strview name;
	velr_property_value_type value_type;
	velr_storage_value_type storage_type;
	int64_t i64_;
	double f64_;
	velr_strview bytes;
	velr_strview json;
	velr_strview display;
} velr_vector_embedding_field;

typedef struct velr_vector_embedding_input {
	velr_strview index_name;
	size_t dimensions;
	velr_vector_embedding_purpose purpose;
	velr_vector_entity_kind entity_kind;
	int has_entity_id;
	int64_t entity_id;
	const velr_vector_embedding_field *fields;
	size_t field_count;
} velr_vector_embedding_input;

typedef velr_code (*velr_vector_embedder_callback)(
	void *user_data,
	const velr_vector_embedding_input *inputs,
	size_t input_count,
	size_t dimensions,
	float *out_vectors,
	char *err_buf,
	size_t err_buf_len);

typedef void (*velr_vector_embedder_free_callback)(void *user_data);

typedef struct velr_arrow_chunks {
	const struct ArrowSchema* const* schemas;
	const struct ArrowArray* const* arrays;
	size_t chunk_count;
} velr_arrow_chunks;

extern velr_code velrGoVectorEmbedder(
	void *user_data,
	const velr_vector_embedding_input *inputs,
	size_t input_count,
	size_t dimensions,
	float *out_vectors,
	char *err_buf,
	size_t err_buf_len);

extern void velrGoVectorEmbedderFree(void *user_data);

typedef struct velr_explain_plan_meta {
	velr_strview plan_id;
	velr_strview cypher;
	size_t step_count;
} velr_explain_plan_meta;

typedef struct velr_explain_step_meta {
	size_t step_no;
	velr_strview group_id;
	velr_strview op_index;
	velr_strview phase;
	velr_strview title;
	velr_strview source;
	velr_strview note;
	size_t statement_count;
} velr_explain_step_meta;

typedef struct velr_explain_stmt_meta {
	velr_strview stmt_id;
	velr_strview kind;
	velr_strview sql;
	velr_strview note;
	size_t sqlite_plan_count;
} velr_explain_stmt_meta;

typedef void (*fn_velr_string_free)(char*);
typedef void (*fn_velr_free)(uint8_t*, size_t);
typedef velr_code (*fn_velr_open)(const char*, velr_db**, char**);
typedef velr_code (*fn_velr_open_existing_readonly)(const char*, velr_db**, char**);
typedef void (*fn_velr_close)(velr_db*);
typedef velr_query_params* (*fn_velr_query_params_new)(void);
typedef void (*fn_velr_query_params_free)(velr_query_params*);
typedef velr_code (*fn_velr_query_params_set_null)(velr_query_params*, velr_strview, char**);
typedef velr_code (*fn_velr_query_params_set_bool)(velr_query_params*, velr_strview, int, char**);
typedef velr_code (*fn_velr_query_params_set_i64)(velr_query_params*, velr_strview, int64_t, char**);
typedef velr_code (*fn_velr_query_params_set_f64)(velr_query_params*, velr_strview, double, char**);
typedef velr_code (*fn_velr_query_params_set_text)(velr_query_params*, velr_strview, velr_strview, char**);
typedef velr_code (*fn_velr_query_params_set_json)(velr_query_params*, velr_strview, velr_strview, char**);
typedef velr_code (*fn_velr_schema_version)(velr_db*, int32_t*, char**);
typedef velr_code (*fn_velr_current_schema_version)(velr_db*, int32_t*, char**);
typedef velr_code (*fn_velr_needs_migration)(velr_db*, int*, char**);
typedef velr_code (*fn_velr_migrate)(velr_db*, velr_migration_report*, char**);
typedef void (*fn_velr_migration_report_clear)(velr_migration_report*);
typedef velr_code (*fn_velr_register_vector_embedder)(velr_db*, velr_strview, velr_vector_embedder_callback, void*, velr_vector_embedder_free_callback, char**);
typedef velr_code (*fn_velr_exec_start)(velr_db*, const char*, velr_stream**, char**);
typedef velr_code (*fn_velr_exec_start_with_options)(velr_db*, const char*, const velr_query_options*, velr_stream**, char**);
typedef void (*fn_velr_exec_close)(velr_stream*);
typedef velr_code (*fn_velr_stream_next_table)(velr_stream*, velr_table**, int*, char**);
typedef velr_code (*fn_velr_exec_one)(velr_db*, const char*, velr_table**, char**);
typedef velr_code (*fn_velr_exec_one_with_options)(velr_db*, const char*, const velr_query_options*, velr_table**, char**);
typedef void (*fn_velr_table_close)(velr_table*);
typedef size_t (*fn_velr_table_column_count)(velr_table*);
typedef velr_code (*fn_velr_table_column_name)(velr_table*, size_t, const uint8_t**, size_t*);
typedef velr_code (*fn_velr_table_rows_open)(velr_table*, velr_rows**, char**);
typedef void (*fn_velr_rows_close)(velr_rows*);
typedef int (*fn_velr_rows_next)(velr_rows*, velr_cell*, size_t, size_t*, char**);
typedef velr_code (*fn_velr_table_ipc_file_malloc)(velr_table*, uint8_t**, size_t*, char**);
typedef velr_code (*fn_velr_bind_arrow)(velr_db*, const char*, const struct ArrowSchema* const*, const struct ArrowArray* const*, const velr_strview*, size_t, char**);
typedef velr_code (*fn_velr_bind_arrow_ipc)(velr_db*, const char*, const uint8_t*, size_t, char**);
typedef velr_code (*fn_velr_bind_arrow_chunks)(velr_db*, const char*, const velr_arrow_chunks*, const velr_strview*, size_t, char**);
typedef velr_code (*fn_velr_tx_begin)(velr_db*, velr_tx**, char**);
typedef velr_code (*fn_velr_tx_commit)(velr_tx*, char**);
typedef velr_code (*fn_velr_tx_rollback)(velr_tx*, char**);
typedef void (*fn_velr_tx_close)(velr_tx*);
typedef velr_code (*fn_velr_tx_exec_start)(velr_tx*, const char*, velr_stream_tx**, char**);
typedef velr_code (*fn_velr_tx_exec_start_with_options)(velr_tx*, const char*, const velr_query_options*, velr_stream_tx**, char**);
typedef velr_code (*fn_velr_stream_tx_next_table)(velr_stream_tx*, velr_table**, int*, char**);
typedef void (*fn_velr_exec_tx_close)(velr_stream_tx*);
typedef velr_code (*fn_velr_tx_savepoint)(velr_tx*, velr_sp**, char**);
typedef velr_code (*fn_velr_tx_savepoint_named)(velr_tx*, const char*, velr_sp**, char**);
typedef velr_code (*fn_velr_tx_rollback_to)(velr_tx*, const char*, char**);
typedef velr_code (*fn_velr_sp_release)(velr_sp*, char**);
typedef velr_code (*fn_velr_sp_rollback)(velr_sp*, char**);
typedef void (*fn_velr_sp_close)(velr_sp*);
typedef velr_code (*fn_velr_tx_bind_arrow)(velr_tx*, const char*, const struct ArrowSchema* const*, const struct ArrowArray* const*, const velr_strview*, size_t, char**);
typedef velr_code (*fn_velr_tx_bind_arrow_ipc)(velr_tx*, const char*, const uint8_t*, size_t, char**);
typedef velr_code (*fn_velr_tx_bind_arrow_chunks)(velr_tx*, const char*, const velr_arrow_chunks*, const velr_strview*, size_t, char**);
typedef velr_code (*fn_velr_explain)(velr_db*, const char*, velr_explain_trace**, char**);
typedef velr_code (*fn_velr_explain_analyze)(velr_db*, const char*, velr_explain_trace**, char**);
typedef velr_code (*fn_velr_tx_explain)(velr_tx*, const char*, velr_explain_trace**, char**);
typedef velr_code (*fn_velr_tx_explain_analyze)(velr_tx*, const char*, velr_explain_trace**, char**);
typedef void (*fn_velr_explain_trace_close)(velr_explain_trace*);
typedef size_t (*fn_velr_explain_trace_plan_count)(velr_explain_trace*);
typedef velr_code (*fn_velr_explain_trace_plan_meta)(velr_explain_trace*, size_t, velr_explain_plan_meta*);
typedef size_t (*fn_velr_explain_trace_step_count)(velr_explain_trace*, size_t);
typedef velr_code (*fn_velr_explain_trace_step_meta)(velr_explain_trace*, size_t, size_t, velr_explain_step_meta*);
typedef size_t (*fn_velr_explain_trace_statement_count)(velr_explain_trace*, size_t, size_t);
typedef velr_code (*fn_velr_explain_trace_statement_meta)(velr_explain_trace*, size_t, size_t, size_t, velr_explain_stmt_meta*);
typedef size_t (*fn_velr_explain_trace_sqlite_plan_count)(velr_explain_trace*, size_t, size_t, size_t);
typedef velr_code (*fn_velr_explain_trace_sqlite_plan_detail)(velr_explain_trace*, size_t, size_t, size_t, size_t, velr_strview*);
typedef velr_code (*fn_velr_explain_trace_compact_len)(velr_explain_trace*, size_t*, char**);
typedef velr_code (*fn_velr_explain_trace_compact_write)(velr_explain_trace*, uint8_t*, size_t, size_t*, char**);
typedef velr_code (*fn_velr_explain_trace_compact_malloc)(velr_explain_trace*, uint8_t**, size_t*, char**);

typedef struct velr_api {
	void *handle;
	char err[1024];
	fn_velr_string_free velr_string_free;
	fn_velr_free velr_free;
	fn_velr_open velr_open;
	fn_velr_open_existing_readonly velr_open_existing_readonly;
	fn_velr_close velr_close;
	fn_velr_query_params_new velr_query_params_new;
	fn_velr_query_params_free velr_query_params_free;
	fn_velr_query_params_set_null velr_query_params_set_null;
	fn_velr_query_params_set_bool velr_query_params_set_bool;
	fn_velr_query_params_set_i64 velr_query_params_set_i64;
	fn_velr_query_params_set_f64 velr_query_params_set_f64;
	fn_velr_query_params_set_text velr_query_params_set_text;
	fn_velr_query_params_set_json velr_query_params_set_json;
	fn_velr_schema_version velr_schema_version;
	fn_velr_current_schema_version velr_current_schema_version;
	fn_velr_needs_migration velr_needs_migration;
	fn_velr_migrate velr_migrate;
	fn_velr_migration_report_clear velr_migration_report_clear;
	fn_velr_register_vector_embedder velr_register_vector_embedder;
	fn_velr_exec_start velr_exec_start;
	fn_velr_exec_start_with_options velr_exec_start_with_options;
	fn_velr_exec_close velr_exec_close;
	fn_velr_stream_next_table velr_stream_next_table;
	fn_velr_exec_one velr_exec_one;
	fn_velr_exec_one_with_options velr_exec_one_with_options;
	fn_velr_table_close velr_table_close;
	fn_velr_table_column_count velr_table_column_count;
	fn_velr_table_column_name velr_table_column_name;
	fn_velr_table_rows_open velr_table_rows_open;
	fn_velr_rows_close velr_rows_close;
	fn_velr_rows_next velr_rows_next;
	fn_velr_table_ipc_file_malloc velr_table_ipc_file_malloc;
	fn_velr_bind_arrow velr_bind_arrow;
	fn_velr_bind_arrow_ipc velr_bind_arrow_ipc;
	fn_velr_bind_arrow_chunks velr_bind_arrow_chunks;
	fn_velr_tx_begin velr_tx_begin;
	fn_velr_tx_commit velr_tx_commit;
	fn_velr_tx_rollback velr_tx_rollback;
	fn_velr_tx_close velr_tx_close;
	fn_velr_tx_exec_start velr_tx_exec_start;
	fn_velr_tx_exec_start_with_options velr_tx_exec_start_with_options;
	fn_velr_stream_tx_next_table velr_stream_tx_next_table;
	fn_velr_exec_tx_close velr_exec_tx_close;
	fn_velr_tx_savepoint velr_tx_savepoint;
	fn_velr_tx_savepoint_named velr_tx_savepoint_named;
	fn_velr_tx_rollback_to velr_tx_rollback_to;
	fn_velr_sp_release velr_sp_release;
	fn_velr_sp_rollback velr_sp_rollback;
	fn_velr_sp_close velr_sp_close;
	fn_velr_tx_bind_arrow velr_tx_bind_arrow;
	fn_velr_tx_bind_arrow_ipc velr_tx_bind_arrow_ipc;
	fn_velr_tx_bind_arrow_chunks velr_tx_bind_arrow_chunks;
	fn_velr_explain velr_explain;
	fn_velr_explain_analyze velr_explain_analyze;
	fn_velr_tx_explain velr_tx_explain;
	fn_velr_tx_explain_analyze velr_tx_explain_analyze;
	fn_velr_explain_trace_close velr_explain_trace_close;
	fn_velr_explain_trace_plan_count velr_explain_trace_plan_count;
	fn_velr_explain_trace_plan_meta velr_explain_trace_plan_meta;
	fn_velr_explain_trace_step_count velr_explain_trace_step_count;
	fn_velr_explain_trace_step_meta velr_explain_trace_step_meta;
	fn_velr_explain_trace_statement_count velr_explain_trace_statement_count;
	fn_velr_explain_trace_statement_meta velr_explain_trace_statement_meta;
	fn_velr_explain_trace_sqlite_plan_count velr_explain_trace_sqlite_plan_count;
	fn_velr_explain_trace_sqlite_plan_detail velr_explain_trace_sqlite_plan_detail;
	fn_velr_explain_trace_compact_len velr_explain_trace_compact_len;
	fn_velr_explain_trace_compact_write velr_explain_trace_compact_write;
	fn_velr_explain_trace_compact_malloc velr_explain_trace_compact_malloc;
} velr_api;

static int velr_go_load_symbol(velr_api *api, const char *name, void **out) {
	*out = velr_go_symbol(api->handle, name);
	if (!*out) {
		snprintf(api->err, sizeof(api->err), "missing Velr runtime symbol %s: %s", name, velr_go_last_error());
		return -1;
	}
	return 0;
}

#define VELR_GO_LOAD(name) if (velr_go_load_symbol(api, #name, (void**)&api->name) != 0) return -1

static int velr_go_api_load(velr_api *api, const char *path) {
	memset(api, 0, sizeof(*api));
	api->handle = velr_go_open_library(path);
	if (!api->handle) {
		snprintf(api->err, sizeof(api->err), "failed to load Velr runtime %s: %s", path, velr_go_last_error());
		return -1;
	}
	VELR_GO_LOAD(velr_string_free);
	VELR_GO_LOAD(velr_free);
	VELR_GO_LOAD(velr_open);
	VELR_GO_LOAD(velr_open_existing_readonly);
	VELR_GO_LOAD(velr_close);
	VELR_GO_LOAD(velr_query_params_new);
	VELR_GO_LOAD(velr_query_params_free);
	VELR_GO_LOAD(velr_query_params_set_null);
	VELR_GO_LOAD(velr_query_params_set_bool);
	VELR_GO_LOAD(velr_query_params_set_i64);
	VELR_GO_LOAD(velr_query_params_set_f64);
	VELR_GO_LOAD(velr_query_params_set_text);
	VELR_GO_LOAD(velr_query_params_set_json);
	VELR_GO_LOAD(velr_schema_version);
	VELR_GO_LOAD(velr_current_schema_version);
	VELR_GO_LOAD(velr_needs_migration);
	VELR_GO_LOAD(velr_migrate);
	VELR_GO_LOAD(velr_migration_report_clear);
	VELR_GO_LOAD(velr_register_vector_embedder);
	VELR_GO_LOAD(velr_exec_start);
	VELR_GO_LOAD(velr_exec_start_with_options);
	VELR_GO_LOAD(velr_exec_close);
	VELR_GO_LOAD(velr_stream_next_table);
	VELR_GO_LOAD(velr_exec_one);
	VELR_GO_LOAD(velr_exec_one_with_options);
	VELR_GO_LOAD(velr_table_close);
	VELR_GO_LOAD(velr_table_column_count);
	VELR_GO_LOAD(velr_table_column_name);
	VELR_GO_LOAD(velr_table_rows_open);
	VELR_GO_LOAD(velr_rows_close);
	VELR_GO_LOAD(velr_rows_next);
	VELR_GO_LOAD(velr_table_ipc_file_malloc);
	VELR_GO_LOAD(velr_bind_arrow);
	VELR_GO_LOAD(velr_bind_arrow_ipc);
	VELR_GO_LOAD(velr_bind_arrow_chunks);
	VELR_GO_LOAD(velr_tx_begin);
	VELR_GO_LOAD(velr_tx_commit);
	VELR_GO_LOAD(velr_tx_rollback);
	VELR_GO_LOAD(velr_tx_close);
	VELR_GO_LOAD(velr_tx_exec_start);
	VELR_GO_LOAD(velr_tx_exec_start_with_options);
	VELR_GO_LOAD(velr_stream_tx_next_table);
	VELR_GO_LOAD(velr_exec_tx_close);
	VELR_GO_LOAD(velr_tx_savepoint);
	VELR_GO_LOAD(velr_tx_savepoint_named);
	VELR_GO_LOAD(velr_tx_rollback_to);
	VELR_GO_LOAD(velr_sp_release);
	VELR_GO_LOAD(velr_sp_rollback);
	VELR_GO_LOAD(velr_sp_close);
	VELR_GO_LOAD(velr_tx_bind_arrow);
	VELR_GO_LOAD(velr_tx_bind_arrow_ipc);
	VELR_GO_LOAD(velr_tx_bind_arrow_chunks);
	VELR_GO_LOAD(velr_explain);
	VELR_GO_LOAD(velr_explain_analyze);
	VELR_GO_LOAD(velr_tx_explain);
	VELR_GO_LOAD(velr_tx_explain_analyze);
	VELR_GO_LOAD(velr_explain_trace_close);
	VELR_GO_LOAD(velr_explain_trace_plan_count);
	VELR_GO_LOAD(velr_explain_trace_plan_meta);
	VELR_GO_LOAD(velr_explain_trace_step_count);
	VELR_GO_LOAD(velr_explain_trace_step_meta);
	VELR_GO_LOAD(velr_explain_trace_statement_count);
	VELR_GO_LOAD(velr_explain_trace_statement_meta);
	VELR_GO_LOAD(velr_explain_trace_sqlite_plan_count);
	VELR_GO_LOAD(velr_explain_trace_sqlite_plan_detail);
	VELR_GO_LOAD(velr_explain_trace_compact_len);
	VELR_GO_LOAD(velr_explain_trace_compact_write);
	VELR_GO_LOAD(velr_explain_trace_compact_malloc);
	return 0;
}

static const char *velr_go_api_error(velr_api *api) { return api->err; }

static void velr_go_string_free(velr_api *api, char *s) { api->velr_string_free(s); }
static void velr_go_free(velr_api *api, uint8_t *p, size_t len) { api->velr_free(p, len); }
static velr_code velr_go_open(velr_api *api, const char *path, velr_db **out, char **err) { return api->velr_open(path, out, err); }
static velr_code velr_go_open_existing_readonly(velr_api *api, const char *path, velr_db **out, char **err) { return api->velr_open_existing_readonly(path, out, err); }
static void velr_go_close(velr_api *api, velr_db *db) { api->velr_close(db); }
static velr_query_params *velr_go_query_params_new(velr_api *api) { return api->velr_query_params_new(); }
static void velr_go_query_params_free(velr_api *api, velr_query_params *params) { api->velr_query_params_free(params); }
static velr_code velr_go_query_params_set_null(velr_api *api, velr_query_params *params, velr_strview name, char **err) { return api->velr_query_params_set_null(params, name, err); }
static velr_code velr_go_query_params_set_bool(velr_api *api, velr_query_params *params, velr_strview name, int value, char **err) { return api->velr_query_params_set_bool(params, name, value, err); }
static velr_code velr_go_query_params_set_i64(velr_api *api, velr_query_params *params, velr_strview name, int64_t value, char **err) { return api->velr_query_params_set_i64(params, name, value, err); }
static velr_code velr_go_query_params_set_f64(velr_api *api, velr_query_params *params, velr_strview name, double value, char **err) { return api->velr_query_params_set_f64(params, name, value, err); }
static velr_code velr_go_query_params_set_text(velr_api *api, velr_query_params *params, velr_strview name, velr_strview value, char **err) { return api->velr_query_params_set_text(params, name, value, err); }
static velr_code velr_go_query_params_set_json(velr_api *api, velr_query_params *params, velr_strview name, velr_strview value, char **err) { return api->velr_query_params_set_json(params, name, value, err); }
static velr_code velr_go_schema_version(velr_api *api, velr_db *db, int32_t *out, char **err) { return api->velr_schema_version(db, out, err); }
static velr_code velr_go_current_schema_version(velr_api *api, velr_db *db, int32_t *out, char **err) { return api->velr_current_schema_version(db, out, err); }
static velr_code velr_go_needs_migration(velr_api *api, velr_db *db, int *out, char **err) { return api->velr_needs_migration(db, out, err); }
static velr_code velr_go_migrate(velr_api *api, velr_db *db, velr_migration_report *out, char **err) { return api->velr_migrate(db, out, err); }
static void velr_go_migration_report_clear(velr_api *api, velr_migration_report *report) { api->velr_migration_report_clear(report); }
static velr_code velr_go_register_vector_embedder(velr_api *api, velr_db *db, velr_strview name, uintptr_t user_data, char **err) {
	return api->velr_register_vector_embedder(db, name, velrGoVectorEmbedder, (void*)user_data, velrGoVectorEmbedderFree, err);
}
static velr_code velr_go_exec_start(velr_api *api, velr_db *db, const char *cypher, velr_stream **out, char **err) { return api->velr_exec_start(db, cypher, out, err); }
static velr_code velr_go_exec_start_with_options(velr_api *api, velr_db *db, const char *cypher, const velr_query_options *opts, velr_stream **out, char **err) { return api->velr_exec_start_with_options(db, cypher, opts, out, err); }
static void velr_go_exec_close(velr_api *api, velr_stream *stream) { api->velr_exec_close(stream); }
static velr_code velr_go_stream_next_table(velr_api *api, velr_stream *stream, velr_table **out, int *has, char **err) { return api->velr_stream_next_table(stream, out, has, err); }
static velr_code velr_go_exec_one(velr_api *api, velr_db *db, const char *cypher, velr_table **out, char **err) { return api->velr_exec_one(db, cypher, out, err); }
static velr_code velr_go_exec_one_with_options(velr_api *api, velr_db *db, const char *cypher, const velr_query_options *opts, velr_table **out, char **err) { return api->velr_exec_one_with_options(db, cypher, opts, out, err); }
static void velr_go_table_close(velr_api *api, velr_table *table) { api->velr_table_close(table); }
static size_t velr_go_table_column_count(velr_api *api, velr_table *table) { return api->velr_table_column_count(table); }
static velr_code velr_go_table_column_name(velr_api *api, velr_table *table, size_t idx, const uint8_t **out, size_t *len) { return api->velr_table_column_name(table, idx, out, len); }
static velr_code velr_go_table_rows_open(velr_api *api, velr_table *table, velr_rows **out, char **err) { return api->velr_table_rows_open(table, out, err); }
static void velr_go_rows_close(velr_api *api, velr_rows *rows) { api->velr_rows_close(rows); }
static int velr_go_rows_next(velr_api *api, velr_rows *rows, velr_cell *buf, size_t buf_len, size_t *written, char **err) { return api->velr_rows_next(rows, buf, buf_len, written, err); }
static velr_code velr_go_table_ipc_file_malloc(velr_api *api, velr_table *table, uint8_t **out, size_t *len, char **err) { return api->velr_table_ipc_file_malloc(table, out, len, err); }
static velr_code velr_go_bind_arrow(velr_api *api, velr_db *db, const char *logical, const struct ArrowSchema* const* schemas, const struct ArrowArray* const* arrays, const velr_strview* colnames, size_t col_count, char **err) {
	return api->velr_bind_arrow(db, logical, schemas, arrays, colnames, col_count, err);
}
static velr_code velr_go_bind_arrow_ipc(velr_api *api, velr_db *db, const char *logical, const uint8_t *ptr, size_t len, char **err) { return api->velr_bind_arrow_ipc(db, logical, ptr, len, err); }
static velr_code velr_go_bind_arrow_chunks(velr_api *api, velr_db *db, const char *logical, const velr_arrow_chunks *cols, const velr_strview *colnames, size_t col_count, char **err) {
	return api->velr_bind_arrow_chunks(db, logical, cols, colnames, col_count, err);
}
static velr_code velr_go_tx_begin(velr_api *api, velr_db *db, velr_tx **out, char **err) { return api->velr_tx_begin(db, out, err); }
static velr_code velr_go_tx_commit(velr_api *api, velr_tx *tx, char **err) { return api->velr_tx_commit(tx, err); }
static velr_code velr_go_tx_rollback(velr_api *api, velr_tx *tx, char **err) { return api->velr_tx_rollback(tx, err); }
static void velr_go_tx_close(velr_api *api, velr_tx *tx) { api->velr_tx_close(tx); }
static velr_code velr_go_tx_exec_start(velr_api *api, velr_tx *tx, const char *cypher, velr_stream_tx **out, char **err) { return api->velr_tx_exec_start(tx, cypher, out, err); }
static velr_code velr_go_tx_exec_start_with_options(velr_api *api, velr_tx *tx, const char *cypher, const velr_query_options *opts, velr_stream_tx **out, char **err) { return api->velr_tx_exec_start_with_options(tx, cypher, opts, out, err); }
static velr_code velr_go_stream_tx_next_table(velr_api *api, velr_stream_tx *stream, velr_table **out, int *has, char **err) { return api->velr_stream_tx_next_table(stream, out, has, err); }
static void velr_go_exec_tx_close(velr_api *api, velr_stream_tx *stream) { api->velr_exec_tx_close(stream); }
static velr_code velr_go_tx_savepoint(velr_api *api, velr_tx *tx, velr_sp **out, char **err) { return api->velr_tx_savepoint(tx, out, err); }
static velr_code velr_go_tx_savepoint_named(velr_api *api, velr_tx *tx, const char *name, velr_sp **out, char **err) { return api->velr_tx_savepoint_named(tx, name, out, err); }
static velr_code velr_go_tx_rollback_to(velr_api *api, velr_tx *tx, const char *name, char **err) { return api->velr_tx_rollback_to(tx, name, err); }
static velr_code velr_go_sp_release(velr_api *api, velr_sp *sp, char **err) { return api->velr_sp_release(sp, err); }
static velr_code velr_go_sp_rollback(velr_api *api, velr_sp *sp, char **err) { return api->velr_sp_rollback(sp, err); }
static void velr_go_sp_close(velr_api *api, velr_sp *sp) { api->velr_sp_close(sp); }
static velr_code velr_go_tx_bind_arrow(velr_api *api, velr_tx *tx, const char *logical, const struct ArrowSchema* const* schemas, const struct ArrowArray* const* arrays, const velr_strview* colnames, size_t col_count, char **err) {
	return api->velr_tx_bind_arrow(tx, logical, schemas, arrays, colnames, col_count, err);
}
static velr_code velr_go_tx_bind_arrow_ipc(velr_api *api, velr_tx *tx, const char *logical, const uint8_t *ptr, size_t len, char **err) { return api->velr_tx_bind_arrow_ipc(tx, logical, ptr, len, err); }
static velr_code velr_go_tx_bind_arrow_chunks(velr_api *api, velr_tx *tx, const char *logical, const velr_arrow_chunks *cols, const velr_strview *colnames, size_t col_count, char **err) {
	return api->velr_tx_bind_arrow_chunks(tx, logical, cols, colnames, col_count, err);
}
static velr_code velr_go_explain(velr_api *api, velr_db *db, const char *cypher, velr_explain_trace **out, char **err) { return api->velr_explain(db, cypher, out, err); }
static velr_code velr_go_explain_analyze(velr_api *api, velr_db *db, const char *cypher, velr_explain_trace **out, char **err) { return api->velr_explain_analyze(db, cypher, out, err); }
static velr_code velr_go_tx_explain(velr_api *api, velr_tx *tx, const char *cypher, velr_explain_trace **out, char **err) { return api->velr_tx_explain(tx, cypher, out, err); }
static velr_code velr_go_tx_explain_analyze(velr_api *api, velr_tx *tx, const char *cypher, velr_explain_trace **out, char **err) { return api->velr_tx_explain_analyze(tx, cypher, out, err); }
static void velr_go_explain_trace_close(velr_api *api, velr_explain_trace *trace) { api->velr_explain_trace_close(trace); }
static size_t velr_go_explain_trace_plan_count(velr_api *api, velr_explain_trace *trace) { return api->velr_explain_trace_plan_count(trace); }
static velr_code velr_go_explain_trace_plan_meta(velr_api *api, velr_explain_trace *trace, size_t idx, velr_explain_plan_meta *out) { return api->velr_explain_trace_plan_meta(trace, idx, out); }
static size_t velr_go_explain_trace_step_count(velr_api *api, velr_explain_trace *trace, size_t plan_idx) { return api->velr_explain_trace_step_count(trace, plan_idx); }
static velr_code velr_go_explain_trace_step_meta(velr_api *api, velr_explain_trace *trace, size_t plan_idx, size_t step_idx, velr_explain_step_meta *out) { return api->velr_explain_trace_step_meta(trace, plan_idx, step_idx, out); }
static size_t velr_go_explain_trace_statement_count(velr_api *api, velr_explain_trace *trace, size_t plan_idx, size_t step_idx) { return api->velr_explain_trace_statement_count(trace, plan_idx, step_idx); }
static velr_code velr_go_explain_trace_statement_meta(velr_api *api, velr_explain_trace *trace, size_t plan_idx, size_t step_idx, size_t stmt_idx, velr_explain_stmt_meta *out) { return api->velr_explain_trace_statement_meta(trace, plan_idx, step_idx, stmt_idx, out); }
static size_t velr_go_explain_trace_sqlite_plan_count(velr_api *api, velr_explain_trace *trace, size_t plan_idx, size_t step_idx, size_t stmt_idx) { return api->velr_explain_trace_sqlite_plan_count(trace, plan_idx, step_idx, stmt_idx); }
static velr_code velr_go_explain_trace_sqlite_plan_detail(velr_api *api, velr_explain_trace *trace, size_t plan_idx, size_t step_idx, size_t stmt_idx, size_t detail_idx, velr_strview *out) { return api->velr_explain_trace_sqlite_plan_detail(trace, plan_idx, step_idx, stmt_idx, detail_idx, out); }
static velr_code velr_go_explain_trace_compact_len(velr_api *api, velr_explain_trace *trace, size_t *out_len, char **err) { return api->velr_explain_trace_compact_len(trace, out_len, err); }
static velr_code velr_go_explain_trace_compact_write(velr_api *api, velr_explain_trace *trace, uint8_t *dst, size_t dst_len, size_t *written, char **err) { return api->velr_explain_trace_compact_write(trace, dst, dst_len, written, err); }
static velr_code velr_go_explain_trace_compact_malloc(velr_api *api, velr_explain_trace *trace, uint8_t **out, size_t *len, char **err) { return api->velr_explain_trace_compact_malloc(trace, out, len, err); }
*/
import "C"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/cgo"
	"strings"
	"sync"
	"unsafe"
)

const (
	errArg   = int(C.VELR_GO_EARG)
	errUTF   = int(C.VELR_GO_EUTF)
	errState = int(C.VELR_GO_ESTATE)
	errErr   = int(C.VELR_GO_EERR)
)

// Error is returned by Velr runtime and driver operations.
type Error struct {
	// Code is the numeric status code returned by the native runtime.
	Code int

	// Message is the human-readable error message returned by the runtime or driver.
	Message string
}

// Error returns the formatted Velr error message.
func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("velr error (code %d)", e.Code)
	}
	return fmt.Sprintf("velr error (code %d): %s", e.Code, e.Message)
}

func newError(code int, message string) error {
	return &Error{Code: code, Message: message}
}

type nativeRuntime struct {
	api  C.velr_api
	path string
}

var globalRuntime struct {
	once sync.Once
	rt   *nativeRuntime
	err  error
}

func runtimeAPI() (*nativeRuntime, error) {
	globalRuntime.once.Do(func() {
		path, err := resolveRuntimePath()
		if err != nil {
			globalRuntime.err = err
			return
		}

		cpath, free, err := cString(path, "runtime path")
		if err != nil {
			globalRuntime.err = err
			return
		}
		defer free()

		rt := &nativeRuntime{path: path}
		if C.velr_go_api_load(&rt.api, cpath) != 0 {
			globalRuntime.err = errors.New(C.GoString(C.velr_go_api_error(&rt.api)))
			return
		}
		globalRuntime.rt = rt
	})
	return globalRuntime.rt, globalRuntime.err
}

func resolveRuntimePath() (string, error) {
	for _, env := range []string{"VELR_RUNTIME_PATH", "VELR_NATIVE_LIBRARY", "VELR_LIB"} {
		if value := os.Getenv(env); value != "" {
			abs, err := filepath.Abs(value)
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(abs); err != nil {
				return "", fmt.Errorf("%s points to an unavailable Velr runtime: %w", env, err)
			}
			return abs, nil
		}
	}

	if path := resolveBundledRuntime(); path != "" {
		return path, nil
	}

	if path := resolveLocalRuntime(); path != "" {
		return path, nil
	}

	return "", fmt.Errorf(
		"unable to locate the bundled Velr native runtime for %s/%s; reinstall the Velr Go driver or set VELR_RUNTIME_PATH",
		runtime.GOOS,
		runtime.GOARCH,
	)
}

type runtimePlatform struct {
	packageDir string
	rustDir    string
	names      []string
}

func currentRuntimePlatform() (runtimePlatform, bool) {
	switch {
	case runtime.GOOS == "darwin" && (runtime.GOARCH == "arm64" || runtime.GOARCH == "amd64"):
		return runtimePlatform{
			packageDir: "darwin-universal",
			rustDir:    "macos-universal",
			names:      []string{"libvelrc.dylib", "velrc.dylib"},
		}, true
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return runtimePlatform{
			packageDir: "linux-x64-gnu",
			rustDir:    "linux-x86_64",
			names:      []string{"libvelrc.so", "velrc.so"},
		}, true
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return runtimePlatform{
			packageDir: "linux-arm64-gnu",
			rustDir:    "linux-aarch64",
			names:      []string{"libvelrc.so", "velrc.so"},
		}, true
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		return runtimePlatform{
			packageDir: "win32-x64-msvc",
			rustDir:    "windows-x86_64",
			names:      []string{"velrc.dll"},
		}, true
	default:
		return runtimePlatform{}, false
	}
}

func resolveBundledRuntime() string {
	platform, ok := currentRuntimePlatform()
	if !ok {
		return ""
	}

	for _, name := range platform.names {
		embeddedPath := filepath.ToSlash(filepath.Join("runtime", platform.packageDir, "prebuilt", name))
		bytes, err := embeddedRuntime.ReadFile(embeddedPath)
		if err != nil {
			continue
		}
		path, err := materializeRuntime(bytes, name)
		if err == nil {
			return path
		}
	}
	return ""
}

func materializeRuntime(runtimeBytes []byte, filename string) (string, error) {
	sum := sha256.Sum256(runtimeBytes)
	hash := hex.EncodeToString(sum[:])[:16]
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "velr", "go-runtime", runtime.GOOS+"-"+runtime.GOARCH+"-"+hash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, runtimeBytes) {
			return path, nil
		}
		_ = os.Remove(path)
	}
	tmp, err := os.CreateTemp(dir, filename+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(runtimeBytes)
	closeErr := tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpName)
		return "", writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpName)
		return "", closeErr
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	return path, nil
}

func resolveLocalRuntime() string {
	platform, ok := currentRuntimePlatform()
	if !ok {
		return ""
	}

	_, file, _, ok := runtime.Caller(0)
	var packageRoot string
	if ok {
		packageRoot = filepath.Dir(file)
		for _, path := range packageRuntimeCandidates(packageRoot, platform) {
			if fileExists(path) {
				return path
			}
		}
	}

	for _, start := range []string{packageRoot, "."} {
		if start == "" {
			continue
		}
		if repoRoot := findRepoRoot(start); repoRoot != "" {
			for _, path := range monorepoRuntimeCandidates(repoRoot, platform) {
				if fileExists(path) {
					return path
				}
			}
		}
	}

	return ""
}

func packageRuntimeCandidates(root string, platform runtimePlatform) []string {
	var candidates []string
	for _, name := range platform.names {
		candidates = append(candidates,
			filepath.Join(root, "runtime", platform.packageDir, "prebuilt", name),
			filepath.Join(root, "vendor", name),
			filepath.Join(root, "_vendor", name),
			filepath.Join(root, "prebuilt", name),
			filepath.Join(root, name),
		)
	}
	return candidates
}

func monorepoRuntimeCandidates(root string, platform runtimePlatform) []string {
	var candidates []string
	runtimeRoots := []string{
		filepath.Join(root, "rust", "velr-rust-driver", "runtime", platform.rustDir, "prebuilt"),
		filepath.Join(root, "rust", "target", "release"),
		filepath.Join(root, "rust", "target", "debug"),
		filepath.Join(root, "target", "release"),
		filepath.Join(root, "target", "debug"),
	}
	for _, runtimeRoot := range runtimeRoots {
		for _, name := range platform.names {
			candidates = append(candidates, filepath.Join(runtimeRoot, name))
		}
	}
	return candidates
}

func findRepoRoot(start string) string {
	current, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		if fileExists(filepath.Join(current, "Cargo.toml")) &&
			fileExists(filepath.Join(current, "rust", "velr-ffi", "Cargo.toml")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func cString(value, what string) (*C.char, func(), error) {
	if strings.IndexByte(value, 0) >= 0 {
		return nil, nil, fmt.Errorf("%s contains NUL byte", what)
	}
	p := C.CString(value)
	return p, func() { C.free(unsafe.Pointer(p)) }, nil
}

func strviewFromBytes(bytes []byte) (C.velr_strview, func()) {
	if len(bytes) == 0 {
		return C.velr_strview{}, func() {}
	}
	p := C.CBytes(bytes)
	return C.velr_strview{
			ptr: (*C.uint8_t)(p),
			len: C.size_t(len(bytes)),
		}, func() {
			C.free(p)
		}
}

func strviewFromString(value string) (C.velr_strview, func(), error) {
	if strings.IndexByte(value, 0) >= 0 {
		return C.velr_strview{}, nil, fmt.Errorf("string contains NUL byte")
	}
	view, free := strviewFromBytes([]byte(value))
	return view, free, nil
}

func (rt *nativeRuntime) takeErr(p *C.char) string {
	if p == nil {
		return ""
	}
	msg := C.GoString(p)
	C.velr_go_string_free(&rt.api, p)
	return msg
}

func (rt *nativeRuntime) rcToErr(rc C.velr_code, errPtr *C.char) error {
	code := int(rc)
	if code == int(C.VELR_GO_OK) {
		if errPtr != nil {
			C.velr_go_string_free(&rt.api, errPtr)
		}
		return nil
	}
	return newError(code, rt.takeErr(errPtr))
}

func (rt *nativeRuntime) copyOwnedBytes(ptr *C.uint8_t, length C.size_t, what string) ([]byte, error) {
	n := int(length)
	if length > C.size_t(^uint(0)>>1) {
		if ptr != nil {
			C.velr_go_free(&rt.api, ptr, length)
		}
		return nil, fmt.Errorf("%s is too large", what)
	}
	if ptr == nil {
		if n == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("%s returned null pointer with non-zero length", what)
	}
	defer C.velr_go_free(&rt.api, ptr, length)
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n)
	out := make([]byte, n)
	copy(out, data)
	return out, nil
}

func strviewToString(v C.velr_strview, what string) (string, error) {
	if v.len == 0 {
		return "", nil
	}
	if v.ptr == nil {
		return "", fmt.Errorf("%s is null with non-zero length", what)
	}
	if v.len > C.size_t(^uint(0)>>1) {
		return "", fmt.Errorf("%s is too large", what)
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(v.ptr)), int(v.len))
	return string(data), nil
}

func optStrviewToString(v C.velr_strview, what string) (string, bool, error) {
	if v.ptr == nil && v.len == 0 {
		return "", false, nil
	}
	s, err := strviewToString(v, what)
	return s, true, err
}

// Params is a map of named Cypher parameters.
//
// Query text references parameters as $name; map keys omit the leading $.
type Params map[string]any

// QueryOptions configures query execution.
type QueryOptions struct {
	// HasMaxResultRows reports whether MaxResultRows should be applied.
	HasMaxResultRows bool

	// MaxResultRows caps emitted rows per result table when HasMaxResultRows is true.
	MaxResultRows int

	// Params contains named Cypher parameters for this query.
	Params Params
}

// WithParams creates QueryOptions with named query parameters.
func WithParams(params Params) QueryOptions {
	return QueryOptions{Params: params}
}

// MaxResultRows creates QueryOptions that cap emitted rows per result table.
func MaxResultRows(n int) QueryOptions {
	return QueryOptions{HasMaxResultRows: true, MaxResultRows: n}
}

// WithParams returns a copy of opts with named query parameters set.
func (opts QueryOptions) WithParams(params Params) QueryOptions {
	opts.Params = params
	return opts
}

// WithMaxResultRows returns a copy of opts with a row cap set.
func (opts QueryOptions) WithMaxResultRows(n int) QueryOptions {
	opts.HasMaxResultRows = true
	opts.MaxResultRows = n
	return opts
}

// WithParam returns a copy of opts with one parameter set.
func (opts QueryOptions) WithParam(name string, value any) QueryOptions {
	if opts.Params == nil {
		opts.Params = Params{}
	}
	opts.Params[name] = value
	return opts
}

type rawQueryOptions struct {
	raw    C.velr_query_options
	params *C.velr_query_params
	rt     *nativeRuntime
}

func (q *rawQueryOptions) close() {
	if q != nil && q.params != nil {
		C.velr_go_query_params_free(&q.rt.api, q.params)
		q.params = nil
	}
}

func (rt *nativeRuntime) makeQueryOptions(opts QueryOptions) (*rawQueryOptions, error) {
	if opts.HasMaxResultRows && opts.MaxResultRows < 0 {
		return nil, fmt.Errorf("max result rows cannot be negative")
	}

	raw := &rawQueryOptions{rt: rt}
	raw.raw.has_max_result_rows = 0
	if opts.HasMaxResultRows {
		raw.raw.has_max_result_rows = 1
		raw.raw.max_result_rows = C.size_t(opts.MaxResultRows)
	}

	if len(opts.Params) == 0 {
		return raw, nil
	}

	params := C.velr_go_query_params_new(&rt.api)
	if params == nil {
		return nil, newError(errErr, "velr_query_params_new returned null")
	}
	raw.params = params
	raw.raw.params = params

	for name, value := range opts.Params {
		if err := rt.setParam(params, name, value); err != nil {
			raw.close()
			return nil, err
		}
	}
	return raw, nil
}

func (rt *nativeRuntime) setParam(params *C.velr_query_params, name string, value any) error {
	if name == "" {
		return fmt.Errorf("query parameter name cannot be empty")
	}
	if strings.HasPrefix(name, "$") {
		return fmt.Errorf("query parameter name should not include the leading '$'")
	}

	nameView, freeName, err := strviewFromString(name)
	if err != nil {
		return fmt.Errorf("query parameter name %q: %w", name, err)
	}
	defer freeName()

	var errPtr *C.char
	switch v := value.(type) {
	case nil:
		return rt.rcToErr(C.velr_go_query_params_set_null(&rt.api, params, nameView, &errPtr), errPtr)
	case bool:
		b := 0
		if v {
			b = 1
		}
		return rt.rcToErr(C.velr_go_query_params_set_bool(&rt.api, params, nameView, C.int(b), &errPtr), errPtr)
	case int:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case int8:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case int16:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case int32:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case int64:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case uint:
		if uint64(v) > math.MaxInt64 {
			return fmt.Errorf("query parameter %q overflows int64", name)
		}
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case uint8:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case uint16:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case uint32:
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case uint64:
		if v > math.MaxInt64 {
			return fmt.Errorf("query parameter %q overflows int64", name)
		}
		return rt.rcToErr(C.velr_go_query_params_set_i64(&rt.api, params, nameView, C.int64_t(v), &errPtr), errPtr)
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("query parameter %q must be a finite float", name)
		}
		return rt.rcToErr(C.velr_go_query_params_set_f64(&rt.api, params, nameView, C.double(f), &errPtr), errPtr)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("query parameter %q must be a finite float", name)
		}
		return rt.rcToErr(C.velr_go_query_params_set_f64(&rt.api, params, nameView, C.double(v), &errPtr), errPtr)
	case string:
		valueView, freeValue, err := strviewFromString(v)
		if err != nil {
			return fmt.Errorf("query parameter %q: %w", name, err)
		}
		defer freeValue()
		return rt.rcToErr(C.velr_go_query_params_set_text(&rt.api, params, nameView, valueView, &errPtr), errPtr)
	case json.RawMessage:
		if !json.Valid(v) {
			return fmt.Errorf("query parameter %q contains invalid JSON", name)
		}
		valueView, freeValue := strviewFromBytes([]byte(v))
		defer freeValue()
		return rt.rcToErr(C.velr_go_query_params_set_json(&rt.api, params, nameView, valueView, &errPtr), errPtr)
	default:
		bytes, err := queryParamJSON(value, name)
		if err != nil {
			return err
		}
		valueView, freeValue := strviewFromBytes(bytes)
		defer freeValue()
		return rt.rcToErr(C.velr_go_query_params_set_json(&rt.api, params, nameView, valueView, &errPtr), errPtr)
	}
}

func queryParamJSON(value any, name string) ([]byte, error) {
	normalized, err := normalizeQueryParam(value, "$"+name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func normalizeQueryParam(value any, path string) (any, error) {
	switch v := value.(type) {
	case nil, bool, string:
		return v, nil
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return nil, fmt.Errorf("query parameter %s overflows int64", path)
		}
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return nil, fmt.Errorf("query parameter %s overflows int64", path)
		}
		return int64(v), nil
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("query parameter %s must be a finite float", path)
		}
		return f, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("query parameter %s must be a finite float", path)
		}
		return v, nil
	case json.RawMessage:
		if !json.Valid(v) {
			return nil, fmt.Errorf("query parameter %s contains invalid JSON", path)
		}
		return v, nil
	case PropertyValue:
		return normalizeQueryParam(v.GoValue(), path)
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil, nil
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil, nil
		}
		return normalizeQueryParam(rv.Elem().Interface(), path)
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return nil, nil
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return nil, fmt.Errorf("unsupported query parameter %s type %T; use json.RawMessage for JSON bytes", path, value)
		}
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			item, err := normalizeQueryParam(rv.Index(i).Interface(), fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = item
		}
		return out, nil
	case reflect.Map:
		if rv.IsNil() {
			return nil, nil
		}
		if rv.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported query parameter %s map key type %s; keys must be strings", path, rv.Type().Key())
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			item, err := normalizeQueryParam(iter.Value().Interface(), path+"."+key)
			if err != nil {
				return nil, err
			}
			out[key] = item
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported query parameter %s type %T; use bool, numbers, string, nil, json.RawMessage, lists, or maps with string keys", path, value)
	}
}

func pickOptions(options []QueryOptions) (QueryOptions, bool, error) {
	if len(options) == 0 {
		return QueryOptions{}, false, nil
	}
	if len(options) > 1 {
		return QueryOptions{}, false, fmt.Errorf("expected at most one QueryOptions value")
	}
	return options[0], true, nil
}

// CellType identifies a value returned by Velr.
type CellType int

const (
	// Null is a null result cell.
	Null CellType = iota

	// Bool is a boolean result cell.
	Bool

	// Int64 is a signed 64-bit integer result cell.
	Int64

	// Double is a floating-point result cell.
	Double

	// Text is a UTF-8 text result cell.
	Text

	// JSON is a UTF-8 JSON result cell.
	JSON
)

// String returns the stable text form of t.
func (t CellType) String() string {
	switch t {
	case Null:
		return "null"
	case Bool:
		return "bool"
	case Int64:
		return "int64"
	case Double:
		return "double"
	case Text:
		return "text"
	case JSON:
		return "json"
	default:
		return fmt.Sprintf("CellType(%d)", int(t))
	}
}

// Cell is one value in a result row.
type Cell struct {
	// Type identifies which value field is populated.
	Type CellType

	// Int64 contains integer values and boolean values encoded as 0 or 1.
	Int64 int64

	// Float64 contains floating-point values.
	Float64 float64

	// Bytes contains text or JSON bytes owned by the Go cell.
	Bytes []byte
}

// Value converts the cell to a Go value.
//
// JSON cells are decoded into interface values. Use String for the raw UTF-8
// representation when you do not want JSON parsing.
func (c Cell) Value() any {
	switch c.Type {
	case Null:
		return nil
	case Bool:
		return c.Int64 != 0
	case Int64:
		return c.Int64
	case Double:
		return c.Float64
	case Text:
		return string(c.Bytes)
	case JSON:
		var out any
		if err := json.Unmarshal(c.Bytes, &out); err != nil {
			return string(c.Bytes)
		}
		return out
	default:
		return nil
	}
}

// AsBool converts a boolean cell to bool.
func (c Cell) AsBool() (bool, error) {
	if c.Type != Bool {
		return false, fmt.Errorf("cannot convert %s cell to bool", c.Type)
	}
	return c.Int64 != 0, nil
}

// AsInt64 converts an integer cell to int64.
func (c Cell) AsInt64() (int64, error) {
	if c.Type != Int64 {
		return 0, fmt.Errorf("cannot convert %s cell to int64", c.Type)
	}
	return c.Int64, nil
}

// AsFloat64 converts a double cell to float64.
func (c Cell) AsFloat64() (float64, error) {
	if c.Type != Double {
		return 0, fmt.Errorf("cannot convert %s cell to float64", c.Type)
	}
	return c.Float64, nil
}

// AsString decodes text and JSON cells as UTF-8.
func (c Cell) AsString() (string, error) {
	if c.Type != Text && c.Type != JSON {
		return "", fmt.Errorf("cannot convert %s cell to string", c.Type)
	}
	return string(c.Bytes), nil
}

// String returns a display rendering of the cell.
func (c Cell) String() string {
	switch c.Type {
	case Text, JSON:
		return string(c.Bytes)
	default:
		return fmt.Sprint(c.Value())
	}
}

// DecodeJSON decodes a JSON cell into out.
func (c Cell) DecodeJSON(out any) error {
	if c.Type != JSON {
		return fmt.Errorf("cannot decode %s cell as JSON", c.Type)
	}
	return json.Unmarshal(c.Bytes, out)
}

// AssignTo assigns this cell to a destination pointer.
func (c Cell) AssignTo(dest any) error {
	if dest == nil {
		return fmt.Errorf("scan destination must be a non-nil pointer")
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("scan destination must be a non-nil pointer")
	}

	switch ptr := dest.(type) {
	case *any:
		*ptr = c.Value()
		return nil
	case *PropertyValue:
		value, err := c.AsPropertyValue()
		if err != nil {
			return err
		}
		*ptr = value
		return nil
	case *Node:
		value, err := c.AsProperty()
		if err != nil {
			return err
		}
		node, ok := value.(Node)
		if !ok {
			return fmt.Errorf("cannot convert %s cell to Node", c.Type)
		}
		*ptr = node
		return nil
	case *Relationship:
		value, err := c.AsProperty()
		if err != nil {
			return err
		}
		relationship, ok := value.(Relationship)
		if !ok {
			return fmt.Errorf("cannot convert %s cell to Relationship", c.Type)
		}
		*ptr = relationship
		return nil
	case *Path:
		value, err := c.AsProperty()
		if err != nil {
			return err
		}
		path, ok := value.(Path)
		if !ok {
			return fmt.Errorf("cannot convert %s cell to Path", c.Type)
		}
		*ptr = path
		return nil
	case *GeoJSON:
		value, err := c.AsProperty()
		if err != nil {
			return err
		}
		switch spatial := value.(type) {
		case GeoJSON:
			*ptr = spatial
			return nil
		case PropertyValue:
			if spatial.GeoJSON != nil {
				*ptr = *spatial.GeoJSON
				return nil
			}
		}
		return fmt.Errorf("cannot convert %s cell to GeoJSON", c.Type)
	case *bool:
		value, err := c.AsBool()
		if err != nil {
			return err
		}
		*ptr = value
		return nil
	case *int:
		value, err := c.AsInt64()
		if err != nil {
			return err
		}
		*ptr = int(value)
		return nil
	case *int64:
		value, err := c.AsInt64()
		if err != nil {
			return err
		}
		*ptr = value
		return nil
	case *float64:
		value, err := c.AsFloat64()
		if err != nil {
			return err
		}
		*ptr = value
		return nil
	case *string:
		value, err := c.AsString()
		if err != nil {
			return err
		}
		*ptr = value
		return nil
	case *[]byte:
		if c.Type != Text && c.Type != JSON {
			return fmt.Errorf("cannot convert %s cell to []byte", c.Type)
		}
		*ptr = append((*ptr)[:0], c.Bytes...)
		return nil
	case *json.RawMessage:
		if c.Type != JSON {
			return fmt.Errorf("cannot convert %s cell to json.RawMessage", c.Type)
		}
		*ptr = append((*ptr)[:0], c.Bytes...)
		return nil
	default:
		return fmt.Errorf("unsupported scan destination %T", dest)
	}
}

// Scan assigns a row of cells into destination pointers.
func Scan(row []Cell, dest ...any) error {
	if len(row) != len(dest) {
		return fmt.Errorf("scan expected %d destinations for %d cells", len(row), len(dest))
	}
	for i, cell := range row {
		if err := cell.AssignTo(dest[i]); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}

func cellFromRaw(raw C.velr_cell) (Cell, error) {
	switch int(raw.ty) {
	case int(C.VELR_GO_NULL):
		return Cell{Type: Null}, nil
	case int(C.VELR_GO_BOOL):
		return Cell{Type: Bool, Int64: int64(raw.i64_)}, nil
	case int(C.VELR_GO_INT64):
		return Cell{Type: Int64, Int64: int64(raw.i64_)}, nil
	case int(C.VELR_GO_DOUBLE):
		return Cell{Type: Double, Float64: float64(raw.f64_)}, nil
	case int(C.VELR_GO_TEXT), int(C.VELR_GO_JSON):
		if raw.len > C.size_t(^uint(0)>>1) {
			return Cell{}, fmt.Errorf("cell payload is too large")
		}
		if raw.len != 0 && raw.ptr == nil {
			return Cell{}, fmt.Errorf("cell payload is null with non-zero length")
		}
		n := int(raw.len)
		bytes := make([]byte, n)
		if n > 0 {
			copy(bytes, unsafe.Slice((*byte)(unsafe.Pointer(raw.ptr)), n))
		}
		typ := Text
		if int(raw.ty) == int(C.VELR_GO_JSON) {
			typ = JSON
		}
		return Cell{Type: typ, Bytes: bytes}, nil
	default:
		return Cell{}, fmt.Errorf("unknown Velr cell type %d", int(raw.ty))
	}
}

// DB is a Velr database connection.
type DB struct {
	rt     *nativeRuntime
	mu     sync.Mutex
	ptr    *C.velr_db
	closed bool
}

// OpenInMemory opens an in-memory Velr database.
func OpenInMemory() (*DB, error) {
	return open(nil)
}

// Open opens a file-backed Velr database for reading and writing.
func Open(path string) (*DB, error) {
	return open(&path)
}

func open(path *string) (*DB, error) {
	rt, err := runtimeAPI()
	if err != nil {
		return nil, err
	}

	var cpath *C.char
	var free func()
	if path != nil {
		cpath, free, err = cString(*path, "database path")
		if err != nil {
			return nil, err
		}
		defer free()
	}

	var out *C.velr_db
	var errPtr *C.char
	if err := rt.rcToErr(C.velr_go_open(&rt.api, cpath, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_open returned null database")
	}
	return newDB(rt, out), nil
}

// OpenReadonly opens an existing database in read-only mode.
func OpenReadonly(path string) (*DB, error) {
	rt, err := runtimeAPI()
	if err != nil {
		return nil, err
	}
	cpath, free, err := cString(path, "database path")
	if err != nil {
		return nil, err
	}
	defer free()

	var out *C.velr_db
	var errPtr *C.char
	if err := rt.rcToErr(C.velr_go_open_existing_readonly(&rt.api, cpath, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_open_existing_readonly returned null database")
	}
	return newDB(rt, out), nil
}

func newDB(rt *nativeRuntime, ptr *C.velr_db) *DB {
	db := &DB{rt: rt, ptr: ptr}
	runtime.SetFinalizer(db, (*DB).finalize)
	return db
}

func (db *DB) finalize() {
	_ = db.Close()
}

func (db *DB) dbPtr() (*C.velr_db, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed || db.ptr == nil {
		return nil, newError(errState, "Velr connection is closed")
	}
	return db.ptr, nil
}

// Close closes the database connection. It is safe to call more than once.
func (db *DB) Close() error {
	db.mu.Lock()
	ptr := db.ptr
	if db.closed || ptr == nil {
		db.mu.Unlock()
		return nil
	}
	db.closed = true
	db.ptr = nil
	db.mu.Unlock()

	runtime.SetFinalizer(db, nil)
	C.velr_go_close(&db.rt.api, ptr)
	return nil
}

// SchemaVersion returns the schema version of the opened database.
func (db *DB) SchemaVersion() (int, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return 0, err
	}
	var out C.int32_t
	var errPtr *C.char
	if err := db.rt.rcToErr(C.velr_go_schema_version(&db.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return 0, err
	}
	return int(out), nil
}

// CurrentSchemaVersion returns the schema version supported by the runtime.
func (db *DB) CurrentSchemaVersion() (int, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return 0, err
	}
	var out C.int32_t
	var errPtr *C.char
	if err := db.rt.rcToErr(C.velr_go_current_schema_version(&db.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return 0, err
	}
	return int(out), nil
}

// NeedsMigration reports whether the opened database is older than the runtime schema.
func (db *DB) NeedsMigration() (bool, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return false, err
	}
	var out C.int
	var errPtr *C.char
	if err := db.rt.rcToErr(C.velr_go_needs_migration(&db.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return false, err
	}
	return out != 0, nil
}

// MigrationStatus is returned by Migrate.
type MigrationStatus string

const (
	// MigrationAlreadyCurrent means no schema migration was needed.
	MigrationAlreadyCurrent MigrationStatus = "already_current"

	// MigrationMigrated means the database was migrated to the current schema.
	MigrationMigrated MigrationStatus = "migrated"
)

// MigrationReport describes a schema migration operation.
type MigrationReport struct {
	// FromVersion is the schema version before migration.
	FromVersion int

	// ToVersion is the schema version after migration.
	ToVersion int

	// Status reports whether migration work was performed.
	Status MigrationStatus

	// Steps lists migration steps reported by the runtime.
	Steps []string
}

// Migrate explicitly migrates the opened database to the current schema.
func (db *DB) Migrate() (MigrationReport, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return MigrationReport{}, err
	}
	var raw C.velr_migration_report
	var errPtr *C.char
	if err := db.rt.rcToErr(C.velr_go_migrate(&db.rt.api, ptr, &raw, &errPtr), errPtr); err != nil {
		return MigrationReport{}, err
	}
	defer C.velr_go_migration_report_clear(&db.rt.api, &raw)

	stepsText := ""
	if raw.steps != nil {
		stepsText = C.GoString(raw.steps)
	}
	status := MigrationAlreadyCurrent
	if int(raw.status) == 1 {
		status = MigrationMigrated
	}
	var steps []string
	if stepsText != "" {
		for _, step := range strings.Split(stepsText, ",") {
			if step != "" {
				steps = append(steps, step)
			}
		}
	}
	return MigrationReport{
		FromVersion: int(raw.from_version),
		ToVersion:   int(raw.to_version),
		Status:      status,
		Steps:       steps,
	}, nil
}

// RegisterVectorEmbedder registers or replaces a named vector embedding callback.
//
// Vector indexes refer to this name with
// OPTIONS { indexConfig: { embedder: 'name' } }. The callback is invoked
// synchronously by the native runtime, so it must not perform asynchronous work.
func (db *DB) RegisterVectorEmbedder(name string, embedder VectorEmbedder) error {
	ptr, err := db.dbPtr()
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return newError(errArg, "vector embedder name cannot be empty")
	}
	if embedder == nil {
		return newError(errArg, "vector embedder cannot be nil")
	}

	nameView, freeName, err := strviewFromString(name)
	if err != nil {
		return fmt.Errorf("vector embedder name: %w", err)
	}
	defer freeName()

	handle := cgo.NewHandle(embedder)
	var errPtr *C.char
	err = db.rt.rcToErr(C.velr_go_register_vector_embedder(&db.rt.api, ptr, nameView, C.uintptr_t(handle), &errPtr), errPtr)
	// The native runtime owns the handle after this call. On validation failure
	// it calls velrGoVectorEmbedderFree before returning.
	return err
}

// Exec executes Cypher and returns a stream of result tables.
func (db *DB) Exec(cypher string, options ...QueryOptions) (*Stream, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return nil, err
	}
	ccypher, free, err := cString(cypher, "openCypher")
	if err != nil {
		return nil, err
	}
	defer free()

	opts, hasOpts, err := pickOptions(options)
	if err != nil {
		return nil, err
	}

	var rawOpts *rawQueryOptions
	if hasOpts {
		rawOpts, err = db.rt.makeQueryOptions(opts)
		if err != nil {
			return nil, err
		}
		defer rawOpts.close()
	}

	var out *C.velr_stream
	var errPtr *C.char
	if hasOpts {
		err = db.rt.rcToErr(C.velr_go_exec_start_with_options(&db.rt.api, ptr, ccypher, &rawOpts.raw, &out, &errPtr), errPtr)
	} else {
		err = db.rt.rcToErr(C.velr_go_exec_start(&db.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_exec_start returned null stream")
	}
	return newStream(db.rt, out), nil
}

// ExecOne executes Cypher and requires exactly one result table.
func (db *DB) ExecOne(cypher string, options ...QueryOptions) (*Table, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return nil, err
	}
	ccypher, free, err := cString(cypher, "openCypher")
	if err != nil {
		return nil, err
	}
	defer free()

	opts, hasOpts, err := pickOptions(options)
	if err != nil {
		return nil, err
	}

	var rawOpts *rawQueryOptions
	if hasOpts {
		rawOpts, err = db.rt.makeQueryOptions(opts)
		if err != nil {
			return nil, err
		}
		defer rawOpts.close()
	}

	var out *C.velr_table
	var errPtr *C.char
	if hasOpts {
		err = db.rt.rcToErr(C.velr_go_exec_one_with_options(&db.rt.api, ptr, ccypher, &rawOpts.raw, &out, &errPtr), errPtr)
	} else {
		err = db.rt.rcToErr(C.velr_go_exec_one(&db.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_exec_one returned null table")
	}
	return newTable(db.rt, out), nil
}

// Run executes Cypher and discards all result tables.
func (db *DB) Run(cypher string, options ...QueryOptions) error {
	stream, err := db.Exec(cypher, options...)
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		table, err := stream.NextTable()
		if err != nil {
			return err
		}
		if table == nil {
			return nil
		}
		if err := table.Close(); err != nil {
			return err
		}
	}
}

// Query executes Cypher and converts the single result table to row objects.
func (db *DB) Query(cypher string, options ...QueryOptions) ([]map[string]any, error) {
	table, err := db.ExecOne(cypher, options...)
	if err != nil {
		return nil, err
	}
	defer table.Close()
	return table.ToObjects()
}

// ExecWithParams executes Cypher with named parameters.
func (db *DB) ExecWithParams(cypher string, params Params) (*Stream, error) {
	return db.Exec(cypher, QueryOptions{Params: params})
}

// ExecOneWithParams executes Cypher with named parameters and requires exactly one result table.
func (db *DB) ExecOneWithParams(cypher string, params Params) (*Table, error) {
	return db.ExecOne(cypher, QueryOptions{Params: params})
}

// RunWithParams executes Cypher with named parameters and discards result tables.
func (db *DB) RunWithParams(cypher string, params Params) error {
	return db.Run(cypher, QueryOptions{Params: params})
}

// QueryWithParams executes Cypher with named parameters and returns row objects.
func (db *DB) QueryWithParams(cypher string, params Params) ([]map[string]any, error) {
	return db.Query(cypher, QueryOptions{Params: params})
}

// BindArrowIPC binds Arrow IPC file bytes under a logical table name.
func (db *DB) BindArrowIPC(logical string, ipc []byte) error {
	ptr, err := db.dbPtr()
	if err != nil {
		return err
	}
	if len(ipc) == 0 {
		return fmt.Errorf("Arrow IPC bytes cannot be empty")
	}
	clogical, free, err := cString(logical, "logical table name")
	if err != nil {
		return err
	}
	defer free()
	var errPtr *C.char
	return db.rt.rcToErr(C.velr_go_bind_arrow_ipc(&db.rt.api, ptr, clogical, (*C.uint8_t)(unsafe.Pointer(&ipc[0])), C.size_t(len(ipc)), &errPtr), errPtr)
}

// BindArrow binds Arrow C Data Interface columns under a logical table name.
//
// The ArrowArray pointers are transferred to Velr on success or failure of the
// ABI call. Do not use or release those ArrowArray values after calling.
// ArrowSchema pointers and column names are borrowed only during the call.
func (db *DB) BindArrow(logical string, columns []ArrowColumn) error {
	ptr, err := db.dbPtr()
	if err != nil {
		return err
	}
	return db.rt.bindArrow(ptr, nil, logical, columns)
}

// BindArrowChunks binds chunked Arrow C Data Interface columns under a logical table name.
func (db *DB) BindArrowChunks(logical string, columns []ArrowColumnChunks) error {
	ptr, err := db.dbPtr()
	if err != nil {
		return err
	}
	return db.rt.bindArrowChunks(ptr, nil, logical, columns)
}

// ArrowColumn is one Arrow C Data Interface column.
//
// Schema must point to a struct ArrowSchema and Array must point to a struct
// ArrowArray. The ArrowArray is transferred to Velr by BindArrow; the schema is
// borrowed only during the call.
type ArrowColumn struct {
	// Name is the logical column name exposed to Cypher.
	Name string

	// Schema points to a struct ArrowSchema borrowed for the duration of BindArrow.
	Schema unsafe.Pointer

	// Array points to a struct ArrowArray whose release ownership is transferred to Velr.
	Array unsafe.Pointer
}

// ArrowChunk is one Arrow C Data Interface chunk for a logical column.
type ArrowChunk struct {
	// Schema points to a struct ArrowSchema borrowed for the duration of BindArrowChunks.
	Schema unsafe.Pointer

	// Array points to a struct ArrowArray whose release ownership is transferred to Velr.
	Array unsafe.Pointer
}

// ArrowColumnChunks is a chunked Arrow C Data Interface column.
type ArrowColumnChunks struct {
	// Name is the logical column name exposed to Cypher.
	Name string

	// Chunks contains Arrow arrays for this logical column in row order.
	Chunks []ArrowChunk
}

func (rt *nativeRuntime) bindArrow(db *C.velr_db, tx *C.velr_tx, logical string, columns []ArrowColumn) error {
	if len(columns) == 0 {
		return fmt.Errorf("at least one Arrow column is required")
	}
	clogical, freeLogical, err := cString(logical, "logical table name")
	if err != nil {
		return err
	}
	defer freeLogical()

	names, freeNames, err := arrowNameViewsFromColumns(columns)
	if err != nil {
		return err
	}
	defer freeNames()

	n := len(columns)
	schemasMem := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	arraysMem := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	if schemasMem == nil || arraysMem == nil {
		if schemasMem != nil {
			C.free(schemasMem)
		}
		if arraysMem != nil {
			C.free(arraysMem)
		}
		return fmt.Errorf("failed to allocate Arrow pointer arrays")
	}
	defer C.free(schemasMem)
	defer C.free(arraysMem)

	schemasPtr := (**C.struct_ArrowSchema)(schemasMem)
	arraysPtr := (**C.struct_ArrowArray)(arraysMem)
	schemas := unsafe.Slice(schemasPtr, n)
	arrays := unsafe.Slice(arraysPtr, n)
	for i, column := range columns {
		if column.Schema == nil {
			return fmt.Errorf("Arrow column %q schema pointer is nil", column.Name)
		}
		if column.Array == nil {
			return fmt.Errorf("Arrow column %q array pointer is nil", column.Name)
		}
		schemas[i] = (*C.struct_ArrowSchema)(column.Schema)
		arrays[i] = (*C.struct_ArrowArray)(column.Array)
	}

	var errPtr *C.char
	if db != nil {
		err = rt.rcToErr(C.velr_go_bind_arrow(&rt.api, db, clogical, schemasPtr, arraysPtr, names, C.size_t(n), &errPtr), errPtr)
	} else {
		err = rt.rcToErr(C.velr_go_tx_bind_arrow(&rt.api, tx, clogical, schemasPtr, arraysPtr, names, C.size_t(n), &errPtr), errPtr)
	}
	runtime.KeepAlive(columns)
	return err
}

func (rt *nativeRuntime) bindArrowChunks(db *C.velr_db, tx *C.velr_tx, logical string, columns []ArrowColumnChunks) error {
	if len(columns) == 0 {
		return fmt.Errorf("at least one Arrow column is required")
	}
	clogical, freeLogical, err := cString(logical, "logical table name")
	if err != nil {
		return err
	}
	defer freeLogical()

	names, freeNames, err := arrowNameViewsFromChunkColumns(columns)
	if err != nil {
		return err
	}
	defer freeNames()

	n := len(columns)
	colsMem := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(C.velr_arrow_chunks{})))
	if colsMem == nil {
		return fmt.Errorf("failed to allocate Arrow chunk descriptors")
	}
	defer C.free(colsMem)
	rawColsPtr := (*C.velr_arrow_chunks)(colsMem)
	rawCols := unsafe.Slice(rawColsPtr, n)

	var frees []func()
	defer func() {
		for _, free := range frees {
			free()
		}
	}()

	for i, column := range columns {
		if len(column.Chunks) == 0 {
			return fmt.Errorf("Arrow column %q has no chunks", column.Name)
		}
		chunkCount := len(column.Chunks)
		schemasMem := C.malloc(C.size_t(chunkCount) * C.size_t(unsafe.Sizeof(uintptr(0))))
		arraysMem := C.malloc(C.size_t(chunkCount) * C.size_t(unsafe.Sizeof(uintptr(0))))
		if schemasMem == nil || arraysMem == nil {
			if schemasMem != nil {
				C.free(schemasMem)
			}
			if arraysMem != nil {
				C.free(arraysMem)
			}
			return fmt.Errorf("failed to allocate Arrow chunk pointer arrays")
		}
		frees = append(frees, func() {
			C.free(schemasMem)
			C.free(arraysMem)
		})
		schemasPtr := (**C.struct_ArrowSchema)(schemasMem)
		arraysPtr := (**C.struct_ArrowArray)(arraysMem)
		schemas := unsafe.Slice(schemasPtr, chunkCount)
		arrays := unsafe.Slice(arraysPtr, chunkCount)
		for j, chunk := range column.Chunks {
			if chunk.Schema == nil {
				return fmt.Errorf("Arrow column %q chunk %d schema pointer is nil", column.Name, j)
			}
			if chunk.Array == nil {
				return fmt.Errorf("Arrow column %q chunk %d array pointer is nil", column.Name, j)
			}
			schemas[j] = (*C.struct_ArrowSchema)(chunk.Schema)
			arrays[j] = (*C.struct_ArrowArray)(chunk.Array)
		}
		rawCols[i].schemas = schemasPtr
		rawCols[i].arrays = arraysPtr
		rawCols[i].chunk_count = C.size_t(chunkCount)
	}

	var errPtr *C.char
	if db != nil {
		err = rt.rcToErr(C.velr_go_bind_arrow_chunks(&rt.api, db, clogical, rawColsPtr, names, C.size_t(n), &errPtr), errPtr)
	} else {
		err = rt.rcToErr(C.velr_go_tx_bind_arrow_chunks(&rt.api, tx, clogical, rawColsPtr, names, C.size_t(n), &errPtr), errPtr)
	}
	runtime.KeepAlive(columns)
	return err
}

func arrowNameViewsFromColumns(columns []ArrowColumn) (*C.velr_strview, func(), error) {
	names := make([]string, len(columns))
	for i, column := range columns {
		names[i] = column.Name
	}
	return arrowNameViews(names)
}

func arrowNameViewsFromChunkColumns(columns []ArrowColumnChunks) (*C.velr_strview, func(), error) {
	names := make([]string, len(columns))
	for i, column := range columns {
		names[i] = column.Name
	}
	return arrowNameViews(names)
}

func arrowNameViews(names []string) (*C.velr_strview, func(), error) {
	mem := C.malloc(C.size_t(len(names)) * C.size_t(unsafe.Sizeof(C.velr_strview{})))
	if mem == nil {
		return nil, nil, fmt.Errorf("failed to allocate Arrow column names")
	}
	viewsPtr := (*C.velr_strview)(mem)
	views := unsafe.Slice(viewsPtr, len(names))
	var frees []func()
	freeAll := func() {
		for _, free := range frees {
			free()
		}
		C.free(mem)
	}
	for i, name := range names {
		if name == "" {
			freeAll()
			return nil, nil, fmt.Errorf("Arrow column name at index %d is empty", i)
		}
		view, free, err := strviewFromString(name)
		if err != nil {
			freeAll()
			return nil, nil, fmt.Errorf("Arrow column name %q: %w", name, err)
		}
		views[i] = view
		frees = append(frees, free)
	}
	return viewsPtr, freeAll, nil
}

// BeginTx starts an explicit transaction.
func (db *DB) BeginTx() (*Tx, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return nil, err
	}
	var out *C.velr_tx
	var errPtr *C.char
	if err := db.rt.rcToErr(C.velr_go_tx_begin(&db.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_tx_begin returned null transaction")
	}
	return newTx(db.rt, out), nil
}

// Transaction runs fn inside a transaction, committing on nil error and rolling back otherwise.
func (db *DB) Transaction(fn func(*Tx) error) error {
	tx, err := db.BeginTx()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// Explain builds an explain trace without executing the query.
func (db *DB) Explain(cypher string) (*ExplainTrace, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return nil, err
	}
	return db.explain(ptr, cypher, false)
}

// ExplainAnalyze executes the query and returns an analyzed explain trace.
func (db *DB) ExplainAnalyze(cypher string) (*ExplainTrace, error) {
	ptr, err := db.dbPtr()
	if err != nil {
		return nil, err
	}
	return db.explain(ptr, cypher, true)
}

func (db *DB) explain(ptr *C.velr_db, cypher string, analyze bool) (*ExplainTrace, error) {
	ccypher, free, err := cString(cypher, "openCypher")
	if err != nil {
		return nil, err
	}
	defer free()

	var out *C.velr_explain_trace
	var errPtr *C.char
	if analyze {
		err = db.rt.rcToErr(C.velr_go_explain_analyze(&db.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	} else {
		err = db.rt.rcToErr(C.velr_go_explain(&db.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "explain returned null trace")
	}
	return newExplainTrace(db.rt, out), nil
}

// Stream is a streaming query result.
type Stream struct {
	rt     *nativeRuntime
	mu     sync.Mutex
	ptr    *C.velr_stream
	closed bool
}

func newStream(rt *nativeRuntime, ptr *C.velr_stream) *Stream {
	stream := &Stream{rt: rt, ptr: ptr}
	runtime.SetFinalizer(stream, (*Stream).finalize)
	return stream
}

func (s *Stream) finalize() {
	_ = s.Close()
}

func (s *Stream) streamPtr() (*C.velr_stream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptr == nil {
		return nil, newError(errState, "Velr stream is closed")
	}
	return s.ptr, nil
}

// NextTable returns the next table, or nil when the stream is exhausted.
func (s *Stream) NextTable() (*Table, error) {
	ptr, err := s.streamPtr()
	if err != nil {
		return nil, err
	}
	var out *C.velr_table
	var has C.int
	var errPtr *C.char
	if err := s.rt.rcToErr(C.velr_go_stream_next_table(&s.rt.api, ptr, &out, &has, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if has == 0 {
		return nil, nil
	}
	if out == nil {
		return nil, newError(errErr, "stream returned null table")
	}
	return newTable(s.rt, out), nil
}

// Close releases the stream. It is safe to call more than once.
func (s *Stream) Close() error {
	s.mu.Lock()
	ptr := s.ptr
	if s.closed || ptr == nil {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.ptr = nil
	s.mu.Unlock()
	runtime.SetFinalizer(s, nil)
	C.velr_go_exec_close(&s.rt.api, ptr)
	return nil
}

// Table is one result table.
type Table struct {
	rt          *nativeRuntime
	mu          sync.Mutex
	ptr         *C.velr_table
	closed      bool
	columnNames []string
}

func newTable(rt *nativeRuntime, ptr *C.velr_table) *Table {
	table := &Table{rt: rt, ptr: ptr}
	runtime.SetFinalizer(table, (*Table).finalize)
	return table
}

func (t *Table) finalize() {
	_ = t.Close()
}

func (t *Table) tablePtr() (*C.velr_table, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.ptr == nil {
		return nil, newError(errState, "Velr table is closed")
	}
	return t.ptr, nil
}

// Close releases the table. It is safe to call more than once.
func (t *Table) Close() error {
	t.mu.Lock()
	ptr := t.ptr
	if t.closed || ptr == nil {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.ptr = nil
	t.mu.Unlock()
	runtime.SetFinalizer(t, nil)
	C.velr_go_table_close(&t.rt.api, ptr)
	return nil
}

// ColumnCount returns the number of result columns.
func (t *Table) ColumnCount() (int, error) {
	ptr, err := t.tablePtr()
	if err != nil {
		return 0, err
	}
	return int(C.velr_go_table_column_count(&t.rt.api, ptr)), nil
}

// ColumnNames returns result column names.
func (t *Table) ColumnNames() ([]string, error) {
	t.mu.Lock()
	if t.columnNames != nil {
		out := append([]string(nil), t.columnNames...)
		t.mu.Unlock()
		return out, nil
	}
	t.mu.Unlock()

	ptr, err := t.tablePtr()
	if err != nil {
		return nil, err
	}
	count := int(C.velr_go_table_column_count(&t.rt.api, ptr))
	names := make([]string, 0, count)
	for i := 0; i < count; i++ {
		var data *C.uint8_t
		var length C.size_t
		rc := C.velr_go_table_column_name(&t.rt.api, ptr, C.size_t(i), &data, &length)
		if int(rc) != int(C.VELR_GO_OK) {
			return nil, newError(int(rc), "failed to fetch column name")
		}
		view := C.velr_strview{ptr: data, len: length}
		name, err := strviewToString(view, "column name")
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	t.mu.Lock()
	t.columnNames = append([]string(nil), names...)
	t.mu.Unlock()
	return names, nil
}

// Rows opens a streaming row cursor for the table.
func (t *Table) Rows() (*Rows, error) {
	ptr, err := t.tablePtr()
	if err != nil {
		return nil, err
	}
	count := int(C.velr_go_table_column_count(&t.rt.api, ptr))
	var out *C.velr_rows
	var errPtr *C.char
	if err := t.rt.rcToErr(C.velr_go_table_rows_open(&t.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_table_rows_open returned null rows")
	}
	return newRows(t.rt, out, count), nil
}

// ForEachRow visits every row and closes the row cursor afterwards.
func (t *Table) ForEachRow(fn func([]Cell) error) error {
	rows, err := t.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for {
		row, err := rows.Next()
		if err != nil {
			return err
		}
		if row == nil {
			return nil
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}

// Collect returns every row as cells.
func (t *Table) Collect() ([][]Cell, error) {
	var out [][]Cell
	err := t.ForEachRow(func(row []Cell) error {
		copied := append([]Cell(nil), row...)
		out = append(out, copied)
		return nil
	})
	return out, err
}

// ToObjects converts all rows to maps keyed by column name.
func (t *Table) ToObjects() ([]map[string]any, error) {
	names, err := t.ColumnNames()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	err = t.ForEachRow(func(row []Cell) error {
		object := make(map[string]any, len(row))
		for i, cell := range row {
			name := fmt.Sprint(i)
			if i < len(names) {
				name = names[i]
			}
			object[name] = cell.Value()
		}
		out = append(out, object)
		return nil
	})
	return out, err
}

// ToArrowIPC exports the table as Arrow IPC file bytes.
func (t *Table) ToArrowIPC() ([]byte, error) {
	ptr, err := t.tablePtr()
	if err != nil {
		return nil, err
	}
	var out *C.uint8_t
	var length C.size_t
	var errPtr *C.char
	if err := t.rt.rcToErr(C.velr_go_table_ipc_file_malloc(&t.rt.api, ptr, &out, &length, &errPtr), errPtr); err != nil {
		return nil, err
	}
	return t.rt.copyOwnedBytes(out, length, "Arrow IPC buffer")
}

// Rows is a streaming row cursor.
type Rows struct {
	rt          *nativeRuntime
	mu          sync.Mutex
	ptr         *C.velr_rows
	closed      bool
	columnCount int
}

func newRows(rt *nativeRuntime, ptr *C.velr_rows, columnCount int) *Rows {
	rows := &Rows{rt: rt, ptr: ptr, columnCount: columnCount}
	runtime.SetFinalizer(rows, (*Rows).finalize)
	return rows
}

func (r *Rows) finalize() {
	_ = r.Close()
}

func (r *Rows) rowsPtr() (*C.velr_rows, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.ptr == nil {
		return nil, newError(errState, "Velr rows is closed")
	}
	return r.ptr, nil
}

// Next returns the next row, or nil at EOF.
func (r *Rows) Next() ([]Cell, error) {
	ptr, err := r.rowsPtr()
	if err != nil {
		return nil, err
	}
	var written C.size_t
	var errPtr *C.char
	var buf []C.velr_cell
	var bufPtr *C.velr_cell
	if r.columnCount > 0 {
		buf = make([]C.velr_cell, r.columnCount)
		bufPtr = &buf[0]
	}
	rc := C.velr_go_rows_next(&r.rt.api, ptr, bufPtr, C.size_t(r.columnCount), &written, &errPtr)
	if rc == 0 {
		return nil, nil
	}
	if rc < 0 {
		return nil, r.rt.rcToErr(C.velr_code(rc), errPtr)
	}
	if written > C.size_t(r.columnCount) {
		return nil, newError(errErr, "Velr row wrote more cells than the result column count")
	}
	row := make([]Cell, 0, int(written))
	for i := 0; i < int(written); i++ {
		cell, err := cellFromRaw(buf[i])
		if err != nil {
			return nil, err
		}
		row = append(row, cell)
	}
	return row, nil
}

// NextInto scans the next row into destination pointers.
//
// It returns false at EOF. The number of destinations must match the row width.
func (r *Rows) NextInto(dest ...any) (bool, error) {
	row, err := r.Next()
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	if err := Scan(row, dest...); err != nil {
		return false, err
	}
	return true, nil
}

// Close releases the row cursor. It is safe to call more than once.
func (r *Rows) Close() error {
	r.mu.Lock()
	ptr := r.ptr
	if r.closed || ptr == nil {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.ptr = nil
	r.mu.Unlock()
	runtime.SetFinalizer(r, nil)
	C.velr_go_rows_close(&r.rt.api, ptr)
	return nil
}

// Tx is an explicit Velr transaction.
type Tx struct {
	rt              *nativeRuntime
	mu              sync.Mutex
	ptr             *C.velr_tx
	closed          bool
	namedSavepoints []*Savepoint
}

func newTx(rt *nativeRuntime, ptr *C.velr_tx) *Tx {
	tx := &Tx{rt: rt, ptr: ptr}
	runtime.SetFinalizer(tx, (*Tx).finalize)
	return tx
}

func (tx *Tx) finalize() {
	_ = tx.Close()
}

func (tx *Tx) txPtr() (*C.velr_tx, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.closed || tx.ptr == nil {
		return nil, newError(errState, "Velr transaction is closed")
	}
	return tx.ptr, nil
}

// Exec executes Cypher inside the transaction.
func (tx *Tx) Exec(cypher string, options ...QueryOptions) (*TxStream, error) {
	ptr, err := tx.txPtr()
	if err != nil {
		return nil, err
	}
	ccypher, free, err := cString(cypher, "openCypher")
	if err != nil {
		return nil, err
	}
	defer free()

	opts, hasOpts, err := pickOptions(options)
	if err != nil {
		return nil, err
	}
	var rawOpts *rawQueryOptions
	if hasOpts {
		rawOpts, err = tx.rt.makeQueryOptions(opts)
		if err != nil {
			return nil, err
		}
		defer rawOpts.close()
	}

	var out *C.velr_stream_tx
	var errPtr *C.char
	if hasOpts {
		err = tx.rt.rcToErr(C.velr_go_tx_exec_start_with_options(&tx.rt.api, ptr, ccypher, &rawOpts.raw, &out, &errPtr), errPtr)
	} else {
		err = tx.rt.rcToErr(C.velr_go_tx_exec_start(&tx.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_tx_exec_start returned null stream")
	}
	return newTxStream(tx.rt, out), nil
}

// ExecOne executes Cypher inside the transaction and requires exactly one table.
func (tx *Tx) ExecOne(cypher string, options ...QueryOptions) (*Table, error) {
	stream, err := tx.Exec(cypher, options...)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	first, err := stream.NextTable()
	if err != nil {
		return nil, err
	}
	if first == nil {
		return nil, newError(errErr, "query produced no result tables")
	}
	second, err := stream.NextTable()
	if err != nil {
		_ = first.Close()
		return nil, err
	}
	if second != nil {
		_ = first.Close()
		_ = second.Close()
		return nil, newError(errErr, "query produced multiple tables; use Exec")
	}
	return first, nil
}

// Run executes Cypher inside the transaction and discards result tables.
func (tx *Tx) Run(cypher string, options ...QueryOptions) error {
	stream, err := tx.Exec(cypher, options...)
	if err != nil {
		return err
	}
	defer stream.Close()
	for {
		table, err := stream.NextTable()
		if err != nil {
			return err
		}
		if table == nil {
			return nil
		}
		if err := table.Close(); err != nil {
			return err
		}
	}
}

// Query executes Cypher inside the transaction and converts the single table to objects.
func (tx *Tx) Query(cypher string, options ...QueryOptions) ([]map[string]any, error) {
	table, err := tx.ExecOne(cypher, options...)
	if err != nil {
		return nil, err
	}
	defer table.Close()
	return table.ToObjects()
}

// ExecWithParams executes Cypher inside the transaction with named parameters.
func (tx *Tx) ExecWithParams(cypher string, params Params) (*TxStream, error) {
	return tx.Exec(cypher, QueryOptions{Params: params})
}

// ExecOneWithParams executes Cypher with named parameters and requires exactly one table.
func (tx *Tx) ExecOneWithParams(cypher string, params Params) (*Table, error) {
	return tx.ExecOne(cypher, QueryOptions{Params: params})
}

// RunWithParams executes Cypher inside the transaction with named parameters.
func (tx *Tx) RunWithParams(cypher string, params Params) error {
	return tx.Run(cypher, QueryOptions{Params: params})
}

// QueryWithParams executes Cypher with named parameters and returns row objects.
func (tx *Tx) QueryWithParams(cypher string, params Params) ([]map[string]any, error) {
	return tx.Query(cypher, QueryOptions{Params: params})
}

// BindArrowIPC binds Arrow IPC file bytes under a logical table name inside the transaction.
func (tx *Tx) BindArrowIPC(logical string, ipc []byte) error {
	ptr, err := tx.txPtr()
	if err != nil {
		return err
	}
	if len(ipc) == 0 {
		return fmt.Errorf("Arrow IPC bytes cannot be empty")
	}
	clogical, free, err := cString(logical, "logical table name")
	if err != nil {
		return err
	}
	defer free()
	var errPtr *C.char
	return tx.rt.rcToErr(C.velr_go_tx_bind_arrow_ipc(&tx.rt.api, ptr, clogical, (*C.uint8_t)(unsafe.Pointer(&ipc[0])), C.size_t(len(ipc)), &errPtr), errPtr)
}

// BindArrow binds Arrow C Data Interface columns inside the transaction.
func (tx *Tx) BindArrow(logical string, columns []ArrowColumn) error {
	ptr, err := tx.txPtr()
	if err != nil {
		return err
	}
	return tx.rt.bindArrow(nil, ptr, logical, columns)
}

// BindArrowChunks binds chunked Arrow C Data Interface columns inside the transaction.
func (tx *Tx) BindArrowChunks(logical string, columns []ArrowColumnChunks) error {
	ptr, err := tx.txPtr()
	if err != nil {
		return err
	}
	return tx.rt.bindArrowChunks(nil, ptr, logical, columns)
}

// Explain builds an explain trace inside the transaction.
func (tx *Tx) Explain(cypher string) (*ExplainTrace, error) {
	ptr, err := tx.txPtr()
	if err != nil {
		return nil, err
	}
	return tx.explain(ptr, cypher, false)
}

// ExplainAnalyze executes the query in the transaction and returns an analyzed explain trace.
func (tx *Tx) ExplainAnalyze(cypher string) (*ExplainTrace, error) {
	ptr, err := tx.txPtr()
	if err != nil {
		return nil, err
	}
	return tx.explain(ptr, cypher, true)
}

func (tx *Tx) explain(ptr *C.velr_tx, cypher string, analyze bool) (*ExplainTrace, error) {
	ccypher, free, err := cString(cypher, "openCypher")
	if err != nil {
		return nil, err
	}
	defer free()
	var out *C.velr_explain_trace
	var errPtr *C.char
	if analyze {
		err = tx.rt.rcToErr(C.velr_go_tx_explain_analyze(&tx.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	} else {
		err = tx.rt.rcToErr(C.velr_go_tx_explain(&tx.rt.api, ptr, ccypher, &out, &errPtr), errPtr)
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "transaction explain returned null trace")
	}
	return newExplainTrace(tx.rt, out), nil
}

// Savepoint creates an unnamed scoped savepoint.
func (tx *Tx) Savepoint() (*Savepoint, error) {
	ptr, err := tx.txPtr()
	if err != nil {
		return nil, err
	}
	var out *C.velr_sp
	var errPtr *C.char
	if err := tx.rt.rcToErr(C.velr_go_tx_savepoint(&tx.rt.api, ptr, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_tx_savepoint returned null savepoint")
	}
	return newSavepoint(tx.rt, out, "", nil), nil
}

// SavepointNamed creates a named transaction-owned savepoint.
func (tx *Tx) SavepointNamed(name string) (*Savepoint, error) {
	ptr, err := tx.txPtr()
	if err != nil {
		return nil, err
	}
	cname, free, err := cString(name, "savepoint name")
	if err != nil {
		return nil, err
	}
	defer free()
	var out *C.velr_sp
	var errPtr *C.char
	if err := tx.rt.rcToErr(C.velr_go_tx_savepoint_named(&tx.rt.api, ptr, cname, &out, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, newError(errErr, "velr_tx_savepoint_named returned null savepoint")
	}
	sp := newSavepoint(tx.rt, out, name, tx)
	tx.mu.Lock()
	tx.namedSavepoints = append(tx.namedSavepoints, sp)
	tx.mu.Unlock()
	return sp, nil
}

// RollbackTo rolls back to a named savepoint and keeps that savepoint active.
func (tx *Tx) RollbackTo(name string) error {
	ptr, err := tx.txPtr()
	if err != nil {
		return err
	}
	cname, free, err := cString(name, "savepoint name")
	if err != nil {
		return err
	}
	defer free()
	var errPtr *C.char
	if err := tx.rt.rcToErr(C.velr_go_tx_rollback_to(&tx.rt.api, ptr, cname, &errPtr), errPtr); err != nil {
		return err
	}
	tx.forgetNamedSavepointsFrom(name)
	_, err = tx.SavepointNamed(name)
	return err
}

// ReleaseSavepoint releases the most recently created active named savepoint.
func (tx *Tx) ReleaseSavepoint(name string) error {
	tx.mu.Lock()
	if len(tx.namedSavepoints) == 0 {
		tx.mu.Unlock()
		return newError(errState, "no active named savepoints")
	}
	idx := len(tx.namedSavepoints) - 1
	sp := tx.namedSavepoints[idx]
	if sp.name != name {
		tx.mu.Unlock()
		return newError(errState, fmt.Sprintf("ReleaseSavepoint(%q) requires it to be the most recent active named savepoint", name))
	}
	tx.namedSavepoints = tx.namedSavepoints[:idx]
	tx.mu.Unlock()
	return sp.Release()
}

func (tx *Tx) consume() (*C.velr_tx, error) {
	tx.mu.Lock()
	ptr := tx.ptr
	if tx.closed || ptr == nil {
		tx.mu.Unlock()
		return nil, newError(errState, "Velr transaction is closed")
	}
	tx.closed = true
	tx.ptr = nil
	tx.namedSavepoints = nil
	tx.mu.Unlock()
	runtime.SetFinalizer(tx, nil)
	return ptr, nil
}

// Commit commits the transaction. The transaction is consumed.
func (tx *Tx) Commit() error {
	ptr, err := tx.consume()
	if err != nil {
		return err
	}
	var errPtr *C.char
	return tx.rt.rcToErr(C.velr_go_tx_commit(&tx.rt.api, ptr, &errPtr), errPtr)
}

// Rollback rolls back the transaction. The transaction is consumed.
func (tx *Tx) Rollback() error {
	ptr, err := tx.consume()
	if err != nil {
		return err
	}
	var errPtr *C.char
	return tx.rt.rcToErr(C.velr_go_tx_rollback(&tx.rt.api, ptr, &errPtr), errPtr)
}

// Close rolls back an uncommitted transaction. It is safe to call more than once.
func (tx *Tx) Close() error {
	tx.mu.Lock()
	ptr := tx.ptr
	if tx.closed || ptr == nil {
		tx.mu.Unlock()
		return nil
	}
	tx.closed = true
	tx.ptr = nil
	tx.namedSavepoints = nil
	tx.mu.Unlock()
	runtime.SetFinalizer(tx, nil)
	C.velr_go_tx_close(&tx.rt.api, ptr)
	return nil
}

func (tx *Tx) forgetNamedSavepointsFrom(name string) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	target := -1
	for i := len(tx.namedSavepoints) - 1; i >= 0; i-- {
		if tx.namedSavepoints[i].name == name {
			target = i
			break
		}
	}
	if target < 0 {
		return
	}
	for _, sp := range tx.namedSavepoints[target:] {
		sp.markConsumedNoClose()
	}
	tx.namedSavepoints = tx.namedSavepoints[:target]
}

// TxStream is a streaming query result inside a transaction.
type TxStream struct {
	rt     *nativeRuntime
	mu     sync.Mutex
	ptr    *C.velr_stream_tx
	closed bool
}

func newTxStream(rt *nativeRuntime, ptr *C.velr_stream_tx) *TxStream {
	stream := &TxStream{rt: rt, ptr: ptr}
	runtime.SetFinalizer(stream, (*TxStream).finalize)
	return stream
}

func (s *TxStream) finalize() {
	_ = s.Close()
}

func (s *TxStream) streamPtr() (*C.velr_stream_tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptr == nil {
		return nil, newError(errState, "Velr transaction stream is closed")
	}
	return s.ptr, nil
}

// NextTable returns the next table, or nil when the stream is exhausted.
func (s *TxStream) NextTable() (*Table, error) {
	ptr, err := s.streamPtr()
	if err != nil {
		return nil, err
	}
	var out *C.velr_table
	var has C.int
	var errPtr *C.char
	if err := s.rt.rcToErr(C.velr_go_stream_tx_next_table(&s.rt.api, ptr, &out, &has, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if has == 0 {
		return nil, nil
	}
	if out == nil {
		return nil, newError(errErr, "transaction stream returned null table")
	}
	return newTable(s.rt, out), nil
}

// Close releases the transaction stream. It is safe to call more than once.
func (s *TxStream) Close() error {
	s.mu.Lock()
	ptr := s.ptr
	if s.closed || ptr == nil {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.ptr = nil
	s.mu.Unlock()
	runtime.SetFinalizer(s, nil)
	C.velr_go_exec_tx_close(&s.rt.api, ptr)
	return nil
}

// Savepoint is a transaction savepoint handle.
type Savepoint struct {
	rt     *nativeRuntime
	mu     sync.Mutex
	ptr    *C.velr_sp
	closed bool
	name   string
	tx     *Tx
}

func newSavepoint(rt *nativeRuntime, ptr *C.velr_sp, name string, tx *Tx) *Savepoint {
	sp := &Savepoint{rt: rt, ptr: ptr, name: name, tx: tx}
	runtime.SetFinalizer(sp, (*Savepoint).finalize)
	return sp
}

func (sp *Savepoint) finalize() {
	_ = sp.Close()
}

func (sp *Savepoint) consume() (*C.velr_sp, error) {
	sp.mu.Lock()
	ptr := sp.ptr
	if sp.closed || ptr == nil {
		sp.mu.Unlock()
		return nil, newError(errState, "Velr savepoint is closed")
	}
	sp.closed = true
	sp.ptr = nil
	sp.mu.Unlock()
	runtime.SetFinalizer(sp, nil)
	return ptr, nil
}

func (sp *Savepoint) markConsumedNoClose() {
	sp.mu.Lock()
	sp.closed = true
	sp.ptr = nil
	sp.mu.Unlock()
	runtime.SetFinalizer(sp, nil)
}

func (sp *Savepoint) forgetNamed() {
	if sp.tx == nil || sp.name == "" {
		return
	}
	sp.tx.mu.Lock()
	defer sp.tx.mu.Unlock()
	for i := len(sp.tx.namedSavepoints) - 1; i >= 0; i-- {
		if sp.tx.namedSavepoints[i] == sp {
			sp.tx.namedSavepoints = append(sp.tx.namedSavepoints[:i], sp.tx.namedSavepoints[i+1:]...)
			return
		}
	}
}

// Release releases the savepoint. The savepoint is consumed.
func (sp *Savepoint) Release() error {
	ptr, err := sp.consume()
	if err != nil {
		return err
	}
	sp.forgetNamed()
	var errPtr *C.char
	return sp.rt.rcToErr(C.velr_go_sp_release(&sp.rt.api, ptr, &errPtr), errPtr)
}

// Rollback rolls back to the savepoint and releases it. The savepoint is consumed.
func (sp *Savepoint) Rollback() error {
	ptr, err := sp.consume()
	if err != nil {
		return err
	}
	sp.forgetNamed()
	var errPtr *C.char
	return sp.rt.rcToErr(C.velr_go_sp_rollback(&sp.rt.api, ptr, &errPtr), errPtr)
}

// Close releases the native savepoint handle without an explicit release or rollback.
//
// If the savepoint is still active, the native runtime rolls back to it and
// releases it.
func (sp *Savepoint) Close() error {
	sp.mu.Lock()
	ptr := sp.ptr
	if sp.closed || ptr == nil {
		sp.mu.Unlock()
		return nil
	}
	sp.closed = true
	sp.ptr = nil
	sp.mu.Unlock()
	runtime.SetFinalizer(sp, nil)
	sp.forgetNamed()
	C.velr_go_sp_close(&sp.rt.api, ptr)
	return nil
}

// ExplainTrace is an EXPLAIN or EXPLAIN ANALYZE result.
type ExplainTrace struct {
	rt     *nativeRuntime
	mu     sync.Mutex
	ptr    *C.velr_explain_trace
	closed bool
}

func newExplainTrace(rt *nativeRuntime, ptr *C.velr_explain_trace) *ExplainTrace {
	trace := &ExplainTrace{rt: rt, ptr: ptr}
	runtime.SetFinalizer(trace, (*ExplainTrace).finalize)
	return trace
}

func (x *ExplainTrace) finalize() {
	_ = x.Close()
}

func (x *ExplainTrace) tracePtr() (*C.velr_explain_trace, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.closed || x.ptr == nil {
		return nil, newError(errState, "Velr explain trace is closed")
	}
	return x.ptr, nil
}

// Close releases the trace. It is safe to call more than once.
func (x *ExplainTrace) Close() error {
	x.mu.Lock()
	ptr := x.ptr
	if x.closed || ptr == nil {
		x.mu.Unlock()
		return nil
	}
	x.closed = true
	x.ptr = nil
	x.mu.Unlock()
	runtime.SetFinalizer(x, nil)
	C.velr_go_explain_trace_close(&x.rt.api, ptr)
	return nil
}

// PlanCount returns the number of plans in the trace.
func (x *ExplainTrace) PlanCount() (int, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return 0, err
	}
	return int(C.velr_go_explain_trace_plan_count(&x.rt.api, ptr)), nil
}

// ExplainPlanMeta describes one explain plan.
type ExplainPlanMeta struct {
	// PlanID is the runtime identifier for the plan.
	PlanID string

	// Cypher is the Cypher statement represented by the plan.
	Cypher string

	// StepCount is the number of explain steps in the plan.
	StepCount int
}

// ExplainStepMeta describes one explain step.
type ExplainStepMeta struct {
	// StepNo is the step number within the plan.
	StepNo int

	// GroupID identifies the logical operation group when one is available.
	GroupID string

	// OpIndex identifies the operation index inside the plan.
	OpIndex string

	// Phase describes the planning or execution phase for this step.
	Phase string

	// Title is the human-readable step title.
	Title string

	// Source identifies the Velr planner or runtime source for this step.
	Source string

	// Note contains optional extra detail for this step.
	Note *string

	// StatementCount is the number of SQL statements attached to this step.
	StatementCount int
}

// ExplainStatementMeta describes one explain statement.
type ExplainStatementMeta struct {
	// StatementID is the runtime identifier for the SQL statement.
	StatementID string

	// Kind describes the statement category.
	Kind string

	// SQL is the SQL text emitted for this explain statement.
	SQL string

	// Note contains optional extra detail for this statement.
	Note *string

	// SQLitePlanCount is the number of SQLite plan details for this statement.
	SQLitePlanCount int
}

// ExplainStatement is an owned explain statement snapshot.
type ExplainStatement struct {
	// Meta contains statement metadata copied from the trace.
	Meta ExplainStatementMeta

	// SQLitePlanDetails contains SQLite EXPLAIN QUERY PLAN detail strings.
	SQLitePlanDetails []string
}

// ExplainStep is an owned explain step snapshot.
type ExplainStep struct {
	// Meta contains step metadata copied from the trace.
	Meta ExplainStepMeta

	// Statements contains SQL statements attached to the step.
	Statements []ExplainStatement
}

// ExplainPlan is an owned explain plan snapshot.
type ExplainPlan struct {
	// Meta contains plan metadata copied from the trace.
	Meta ExplainPlanMeta

	// Steps contains the ordered explain steps in the plan.
	Steps []ExplainStep
}

// PlanMeta returns metadata for one plan.
func (x *ExplainTrace) PlanMeta(planIdx int) (ExplainPlanMeta, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return ExplainPlanMeta{}, err
	}
	if planIdx < 0 {
		return ExplainPlanMeta{}, fmt.Errorf("plan index cannot be negative")
	}
	var raw C.velr_explain_plan_meta
	rc := C.velr_go_explain_trace_plan_meta(&x.rt.api, ptr, C.size_t(planIdx), &raw)
	if int(rc) != int(C.VELR_GO_OK) {
		return ExplainPlanMeta{}, newError(int(rc), "failed to fetch explain plan metadata")
	}
	planID, err := strviewToString(raw.plan_id, "explain plan id")
	if err != nil {
		return ExplainPlanMeta{}, err
	}
	cypher, err := strviewToString(raw.cypher, "explain plan cypher")
	if err != nil {
		return ExplainPlanMeta{}, err
	}
	return ExplainPlanMeta{PlanID: planID, Cypher: cypher, StepCount: int(raw.step_count)}, nil
}

// StepCount returns the number of steps in one plan.
func (x *ExplainTrace) StepCount(planIdx int) (int, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return 0, err
	}
	if planIdx < 0 {
		return 0, fmt.Errorf("plan index cannot be negative")
	}
	return int(C.velr_go_explain_trace_step_count(&x.rt.api, ptr, C.size_t(planIdx))), nil
}

// StepMeta returns metadata for one explain step.
func (x *ExplainTrace) StepMeta(planIdx, stepIdx int) (ExplainStepMeta, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return ExplainStepMeta{}, err
	}
	if planIdx < 0 || stepIdx < 0 {
		return ExplainStepMeta{}, fmt.Errorf("explain indices cannot be negative")
	}
	var raw C.velr_explain_step_meta
	rc := C.velr_go_explain_trace_step_meta(&x.rt.api, ptr, C.size_t(planIdx), C.size_t(stepIdx), &raw)
	if int(rc) != int(C.VELR_GO_OK) {
		return ExplainStepMeta{}, newError(int(rc), "failed to fetch explain step metadata")
	}
	groupID, err := strviewToString(raw.group_id, "explain group id")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	opIndex, err := strviewToString(raw.op_index, "explain op index")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	phase, err := strviewToString(raw.phase, "explain phase")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	title, err := strviewToString(raw.title, "explain title")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	source, err := strviewToString(raw.source, "explain source")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	noteText, hasNote, err := optStrviewToString(raw.note, "explain note")
	if err != nil {
		return ExplainStepMeta{}, err
	}
	var note *string
	if hasNote {
		note = &noteText
	}
	return ExplainStepMeta{
		StepNo:         int(raw.step_no),
		GroupID:        groupID,
		OpIndex:        opIndex,
		Phase:          phase,
		Title:          title,
		Source:         source,
		Note:           note,
		StatementCount: int(raw.statement_count),
	}, nil
}

// StatementCount returns the number of statements in one step.
func (x *ExplainTrace) StatementCount(planIdx, stepIdx int) (int, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return 0, err
	}
	if planIdx < 0 || stepIdx < 0 {
		return 0, fmt.Errorf("explain indices cannot be negative")
	}
	return int(C.velr_go_explain_trace_statement_count(&x.rt.api, ptr, C.size_t(planIdx), C.size_t(stepIdx))), nil
}

// StatementMeta returns metadata for one explain statement.
func (x *ExplainTrace) StatementMeta(planIdx, stepIdx, stmtIdx int) (ExplainStatementMeta, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return ExplainStatementMeta{}, err
	}
	if planIdx < 0 || stepIdx < 0 || stmtIdx < 0 {
		return ExplainStatementMeta{}, fmt.Errorf("explain indices cannot be negative")
	}
	var raw C.velr_explain_stmt_meta
	rc := C.velr_go_explain_trace_statement_meta(&x.rt.api, ptr, C.size_t(planIdx), C.size_t(stepIdx), C.size_t(stmtIdx), &raw)
	if int(rc) != int(C.VELR_GO_OK) {
		return ExplainStatementMeta{}, newError(int(rc), "failed to fetch explain statement metadata")
	}
	statementID, err := strviewToString(raw.stmt_id, "explain statement id")
	if err != nil {
		return ExplainStatementMeta{}, err
	}
	kind, err := strviewToString(raw.kind, "explain statement kind")
	if err != nil {
		return ExplainStatementMeta{}, err
	}
	sql, err := strviewToString(raw.sql, "explain statement sql")
	if err != nil {
		return ExplainStatementMeta{}, err
	}
	noteText, hasNote, err := optStrviewToString(raw.note, "explain statement note")
	if err != nil {
		return ExplainStatementMeta{}, err
	}
	var note *string
	if hasNote {
		note = &noteText
	}
	return ExplainStatementMeta{
		StatementID:     statementID,
		Kind:            kind,
		SQL:             sql,
		Note:            note,
		SQLitePlanCount: int(raw.sqlite_plan_count),
	}, nil
}

// SQLitePlanCount returns the number of SQLite plan details for a statement.
func (x *ExplainTrace) SQLitePlanCount(planIdx, stepIdx, stmtIdx int) (int, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return 0, err
	}
	if planIdx < 0 || stepIdx < 0 || stmtIdx < 0 {
		return 0, fmt.Errorf("explain indices cannot be negative")
	}
	return int(C.velr_go_explain_trace_sqlite_plan_count(&x.rt.api, ptr, C.size_t(planIdx), C.size_t(stepIdx), C.size_t(stmtIdx))), nil
}

// SQLitePlanDetail returns one SQLite plan detail string.
func (x *ExplainTrace) SQLitePlanDetail(planIdx, stepIdx, stmtIdx, detailIdx int) (string, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return "", err
	}
	if planIdx < 0 || stepIdx < 0 || stmtIdx < 0 || detailIdx < 0 {
		return "", fmt.Errorf("explain indices cannot be negative")
	}
	var raw C.velr_strview
	rc := C.velr_go_explain_trace_sqlite_plan_detail(&x.rt.api, ptr, C.size_t(planIdx), C.size_t(stepIdx), C.size_t(stmtIdx), C.size_t(detailIdx), &raw)
	if int(rc) != int(C.VELR_GO_OK) {
		return "", newError(int(rc), "failed to fetch SQLite plan detail")
	}
	return strviewToString(raw, "SQLite plan detail")
}

// SQLitePlanDetails returns all SQLite plan detail strings for a statement.
func (x *ExplainTrace) SQLitePlanDetails(planIdx, stepIdx, stmtIdx int) ([]string, error) {
	count, err := x.SQLitePlanCount(planIdx, stepIdx, stmtIdx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		detail, err := x.SQLitePlanDetail(planIdx, stepIdx, stmtIdx, i)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, nil
}

// Snapshot materializes the trace into owned Go structs.
func (x *ExplainTrace) Snapshot() ([]ExplainPlan, error) {
	planCount, err := x.PlanCount()
	if err != nil {
		return nil, err
	}
	plans := make([]ExplainPlan, 0, planCount)
	for p := 0; p < planCount; p++ {
		meta, err := x.PlanMeta(p)
		if err != nil {
			return nil, err
		}
		stepCount, err := x.StepCount(p)
		if err != nil {
			return nil, err
		}
		plan := ExplainPlan{Meta: meta, Steps: make([]ExplainStep, 0, stepCount)}
		for s := 0; s < stepCount; s++ {
			stepMeta, err := x.StepMeta(p, s)
			if err != nil {
				return nil, err
			}
			stmtCount, err := x.StatementCount(p, s)
			if err != nil {
				return nil, err
			}
			step := ExplainStep{Meta: stepMeta, Statements: make([]ExplainStatement, 0, stmtCount)}
			for st := 0; st < stmtCount; st++ {
				stmtMeta, err := x.StatementMeta(p, s, st)
				if err != nil {
					return nil, err
				}
				detailCount, err := x.SQLitePlanCount(p, s, st)
				if err != nil {
					return nil, err
				}
				details, err := x.SQLitePlanDetails(p, s, st)
				if err != nil {
					return nil, err
				}
				if len(details) != detailCount {
					return nil, fmt.Errorf("SQLite plan detail count changed while snapshotting")
				}
				stmt := ExplainStatement{Meta: stmtMeta, SQLitePlanDetails: details}
				step.Statements = append(step.Statements, stmt)
			}
			plan.Steps = append(plan.Steps, step)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// CompactLen returns the byte length of the compact text rendering.
func (x *ExplainTrace) CompactLen() (int, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return 0, err
	}
	var length C.size_t
	var errPtr *C.char
	if err := x.rt.rcToErr(C.velr_go_explain_trace_compact_len(&x.rt.api, ptr, &length, &errPtr), errPtr); err != nil {
		return 0, err
	}
	if length > C.size_t(^uint(0)>>1) {
		return 0, fmt.Errorf("explain compact rendering is too large")
	}
	return int(length), nil
}

// CompactString renders the trace in Velr's compact text form.
func (x *ExplainTrace) CompactString() (string, error) {
	bytes, err := x.CompactBytes()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CompactBytes renders the trace in Velr's compact text form.
func (x *ExplainTrace) CompactBytes() ([]byte, error) {
	ptr, err := x.tracePtr()
	if err != nil {
		return nil, err
	}
	length, err := x.CompactLen()
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	out := make([]byte, length)
	var written C.size_t
	var errPtr *C.char
	if err := x.rt.rcToErr(C.velr_go_explain_trace_compact_write(&x.rt.api, ptr, (*C.uint8_t)(unsafe.Pointer(&out[0])), C.size_t(len(out)), &written, &errPtr), errPtr); err != nil {
		return nil, err
	}
	if written > C.size_t(len(out)) {
		return nil, fmt.Errorf("explain compact writer reported too many bytes")
	}
	return out[:int(written)], nil
}

// WriteCompact writes the compact text rendering to w.
func (x *ExplainTrace) WriteCompact(w io.Writer) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("nil compact trace writer")
	}
	bytes, err := x.CompactBytes()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(bytes)
	if err == nil && n != len(bytes) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}
