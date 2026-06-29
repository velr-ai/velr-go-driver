package velr

/*
#include <stddef.h>
#include <stdint.h>

typedef int velr_code;

typedef struct velr_strview {
	const uint8_t *ptr;
	size_t len;
} velr_strview;

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
*/
import "C"

import (
	"fmt"
	"math"
	"runtime/cgo"
	"unsafe"
)

// VectorEmbedder is a synchronous embedding callback used by vector indexes.
//
// It receives a batch of inputs and must return one vector per input. Every
// vector must have input.Dimensions finite float32 values.
type VectorEmbedder func(inputs []VectorEmbeddingInput) ([][]float32, error)

// VectorEmbeddingPurpose explains why Velr is asking for embeddings.
type VectorEmbeddingPurpose int

const (
	// VectorEmbeddingIndexEntity requests embeddings for graph entities being indexed.
	VectorEmbeddingIndexEntity VectorEmbeddingPurpose = iota

	// VectorEmbeddingQuery requests embeddings for a vector search query.
	VectorEmbeddingQuery
)

// String returns the stable text form of p.
func (p VectorEmbeddingPurpose) String() string {
	switch p {
	case VectorEmbeddingIndexEntity:
		return "index_entity"
	case VectorEmbeddingQuery:
		return "query"
	default:
		return fmt.Sprintf("VectorEmbeddingPurpose(%d)", int(p))
	}
}

// VectorEntityKind identifies the graph entity kind being indexed.
type VectorEntityKind int

const (
	// VectorEntityNone means the embedding input is not tied to a graph entity.
	VectorEntityNone VectorEntityKind = iota

	// VectorEntityNode means the embedding input describes a node.
	VectorEntityNode

	// VectorEntityRelationship means the embedding input describes a relationship.
	VectorEntityRelationship
)

// String returns the stable text form of k.
func (k VectorEntityKind) String() string {
	switch k {
	case VectorEntityNone:
		return "none"
	case VectorEntityNode:
		return "node"
	case VectorEntityRelationship:
		return "relationship"
	default:
		return fmt.Sprintf("VectorEntityKind(%d)", int(k))
	}
}

// PropertyValueType describes the public Velr property value kind.
type PropertyValueType int

const (
	// PropertyNull is a null property value.
	PropertyNull PropertyValueType = iota

	// PropertyBool is a boolean property value.
	PropertyBool

	// PropertyInt64 is a signed 64-bit integer property value.
	PropertyInt64

	// PropertyDouble is a floating-point property value.
	PropertyDouble

	// PropertyString is a string property value.
	PropertyString

	// PropertyDate is a date property value in canonical openCypher text form.
	PropertyDate

	// PropertyLocalTime is a local time property value in canonical openCypher text form.
	PropertyLocalTime

	// PropertyZonedTime is a zoned time property value in canonical openCypher text form.
	PropertyZonedTime

	// PropertyLocalDateTime is a local datetime property value in canonical openCypher text form.
	PropertyLocalDateTime

	// PropertyZonedDateTime is a zoned datetime property value in canonical openCypher text form.
	PropertyZonedDateTime

	// PropertyDuration is a duration property value in canonical openCypher text form.
	PropertyDuration

	// PropertyPoint is a point property value represented as GeoJSON.
	PropertyPoint

	// PropertyGeometry is a geometry property value represented as GeoJSON.
	PropertyGeometry

	// PropertyGeography is a geography property value represented as GeoJSON.
	PropertyGeography

	// PropertyList is a list property value.
	PropertyList

	// PropertyVector is a numeric vector property value.
	PropertyVector

	// PropertyBytes is a byte-array property value.
	PropertyBytes
)

// String returns the stable text form of t.
func (t PropertyValueType) String() string {
	switch t {
	case PropertyNull:
		return "null"
	case PropertyBool:
		return "bool"
	case PropertyInt64:
		return "int64"
	case PropertyDouble:
		return "double"
	case PropertyString:
		return "string"
	case PropertyDate:
		return "date"
	case PropertyLocalTime:
		return "local_time"
	case PropertyZonedTime:
		return "zoned_time"
	case PropertyLocalDateTime:
		return "local_datetime"
	case PropertyZonedDateTime:
		return "zoned_datetime"
	case PropertyDuration:
		return "duration"
	case PropertyPoint:
		return "point"
	case PropertyGeometry:
		return "geometry"
	case PropertyGeography:
		return "geography"
	case PropertyList:
		return "list"
	case PropertyVector:
		return "vector"
	case PropertyBytes:
		return "bytes"
	default:
		return fmt.Sprintf("PropertyValueType(%d)", int(t))
	}
}

// StorageValueType describes the storage class used to reconstruct a Velr value.
type StorageValueType int

const (
	// StorageNull means the original storage value was null.
	StorageNull StorageValueType = iota

	// StorageInt64 means the original storage value was a signed 64-bit integer.
	StorageInt64

	// StorageDouble means the original storage value was a floating-point number.
	StorageDouble

	// StorageText means the original storage value was text.
	StorageText

	// StorageBlob means the original storage value was bytes.
	StorageBlob
)

// String returns the stable text form of t.
func (t StorageValueType) String() string {
	switch t {
	case StorageNull:
		return "null"
	case StorageInt64:
		return "int64"
	case StorageDouble:
		return "double"
	case StorageText:
		return "text"
	case StorageBlob:
		return "blob"
	default:
		return fmt.Sprintf("StorageValueType(%d)", int(t))
	}
}

// VectorEmbeddingField is one value passed to a vector embedder.
type VectorEmbeddingField struct {
	// Name is the source field or property name when Velr supplied one.
	Name string

	// HasName reports whether Name was supplied by the runtime.
	HasName bool

	// Value is the decoded typed property value for this field.
	Value PropertyValue

	// StorageType is the original low-level storage class of the field.
	StorageType StorageValueType

	// RawStorage contains the runtime's raw storage bytes when exposed by the ABI.
	RawStorage []byte

	// RawJSON contains the canonical JSON bytes used to decode Value when present.
	RawJSON []byte

	// Display contains Velr's display rendering for the field.
	Display string
}

// GoValue returns an idiomatic Go rendering of the field value.
func (f VectorEmbeddingField) GoValue() any {
	return f.Value.GoValue()
}

// VectorEmbeddingInput is one source row passed to a vector embedder.
type VectorEmbeddingInput struct {
	// IndexName is the vector index requesting embeddings.
	IndexName string

	// Dimensions is the exact number of float32 values the callback must return per input.
	Dimensions int

	// Purpose explains whether Velr is indexing data or embedding a query.
	Purpose VectorEmbeddingPurpose

	// EntityKind identifies the graph entity kind when HasEntity is true.
	EntityKind VectorEntityKind

	// EntityID is the runtime-local graph entity identifier when HasEntity is true.
	EntityID int64

	// HasEntity reports whether EntityKind and EntityID refer to a graph entity.
	HasEntity bool

	// Fields contains the source values Velr wants embedded.
	Fields []VectorEmbeddingField
}

// Text joins the display rendering of all fields and is useful for simple text embedders.
func (in VectorEmbeddingInput) Text() string {
	out := ""
	for i, field := range in.Fields {
		if i > 0 {
			out += "\n"
		}
		if field.Display != "" {
			out += field.Display
			continue
		}
		switch value := field.GoValue().(type) {
		case string:
			out += value
		default:
			out += fmt.Sprint(value)
		}
	}
	return out
}

//export velrGoVectorEmbedder
func velrGoVectorEmbedder(
	userData unsafe.Pointer,
	rawInputs *C.velr_vector_embedding_input,
	inputCount C.size_t,
	dimensions C.size_t,
	outVectors *C.float,
	errBuf *C.char,
	errBufLen C.size_t,
) C.velr_code {
	err := func() (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("vector embedder callback panicked: %v", recovered)
			}
		}()

		if userData == nil {
			return fmt.Errorf("vector embedder user data is null")
		}
		handle := cgo.Handle(uintptr(userData))
		embedder, ok := handle.Value().(VectorEmbedder)
		if !ok {
			return fmt.Errorf("vector embedder user data has unexpected type")
		}

		inputs, err := vectorInputsFromRaw(rawInputs, int(inputCount))
		if err != nil {
			return err
		}

		outputLen := int(inputCount) * int(dimensions)
		if outputLen > 0 && outVectors == nil {
			return fmt.Errorf("vector embedding output pointer is null")
		}
		vectors, err := embedder(inputs)
		if err != nil {
			return err
		}
		if len(vectors) != int(inputCount) {
			return fmt.Errorf("vector embedder returned %d embeddings for %d inputs", len(vectors), int(inputCount))
		}

		out := unsafe.Slice(outVectors, outputLen)
		for row, vector := range vectors {
			if len(vector) != int(dimensions) {
				return fmt.Errorf("vector embedder returned %d dimensions for input %d but the index expects %d", len(vector), row, int(dimensions))
			}
			for dim, value := range vector {
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					return fmt.Errorf("vector embedder returned a non-finite value for input %d at dimension %d: %v", row, dim, value)
				}
				if outputLen > 0 {
					out[row*int(dimensions)+dim] = C.float(value)
				}
			}
		}
		return nil
	}()

	if err != nil {
		writeVectorCallbackError(errBuf, errBufLen, err.Error())
		return C.velr_code(-4)
	}
	return C.velr_code(0)
}

//export velrGoVectorEmbedderFree
func velrGoVectorEmbedderFree(userData unsafe.Pointer) {
	if userData == nil {
		return
	}
	cgo.Handle(uintptr(userData)).Delete()
}

func writeVectorCallbackError(errBuf *C.char, errBufLen C.size_t, message string) {
	if errBuf == nil || errBufLen == 0 {
		return
	}
	n := int(errBufLen)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(errBuf)), n)
	copyLen := len(message)
	if copyLen > n-1 {
		copyLen = n - 1
	}
	copy(buf[:copyLen], message)
	buf[copyLen] = 0
}

func vectorInputsFromRaw(rawInputs *C.velr_vector_embedding_input, count int) ([]VectorEmbeddingInput, error) {
	if count == 0 {
		return nil, nil
	}
	if rawInputs == nil {
		return nil, fmt.Errorf("vector embedding inputs pointer is null with non-zero count")
	}
	raw := unsafe.Slice(rawInputs, count)
	out := make([]VectorEmbeddingInput, 0, count)
	for i, input := range raw {
		indexName, err := vectorStrviewToString(input.index_name, "vector index name")
		if err != nil {
			return nil, err
		}
		fields, err := vectorFieldsFromRaw(input.fields, int(input.field_count), i)
		if err != nil {
			return nil, err
		}
		out = append(out, VectorEmbeddingInput{
			IndexName:  indexName,
			Dimensions: int(input.dimensions),
			Purpose:    vectorPurposeFromRaw(input.purpose),
			EntityKind: vectorEntityKindFromRaw(input.entity_kind),
			EntityID:   int64(input.entity_id),
			HasEntity:  input.has_entity_id != 0,
			Fields:     fields,
		})
	}
	return out, nil
}

func vectorFieldsFromRaw(rawFields *C.velr_vector_embedding_field, count int, inputIndex int) ([]VectorEmbeddingField, error) {
	if count == 0 {
		return nil, nil
	}
	if rawFields == nil {
		return nil, fmt.Errorf("vector embedding input %d fields pointer is null with non-zero count", inputIndex)
	}
	raw := unsafe.Slice(rawFields, count)
	out := make([]VectorEmbeddingField, 0, count)
	for _, field := range raw {
		name := ""
		var err error
		if field.has_name != 0 {
			name, err = vectorStrviewToString(field.name, "vector field name")
			if err != nil {
				return nil, err
			}
		}
		bytes, err := vectorStrviewBytes(field.bytes, "vector field bytes")
		if err != nil {
			return nil, err
		}
		jsonBytes, err := vectorStrviewBytes(field.json, "vector field json")
		if err != nil {
			return nil, err
		}
		display, err := vectorStrviewToString(field.display, "vector field display")
		if err != nil {
			return nil, err
		}
		value, err := propertyValueFromStorage(
			PropertyValueType(field.value_type),
			StorageValueType(field.storage_type),
			int64(field.i64_),
			float64(field.f64_),
			bytes,
			jsonBytes,
			display,
		)
		if err != nil {
			return nil, fmt.Errorf("vector field %q: %w", name, err)
		}
		out = append(out, VectorEmbeddingField{
			Name:        name,
			HasName:     field.has_name != 0,
			Value:       value,
			StorageType: StorageValueType(field.storage_type),
			RawStorage:  bytes,
			RawJSON:     jsonBytes,
			Display:     display,
		})
	}
	return out, nil
}

func vectorStrviewBytes(view C.velr_strview, what string) ([]byte, error) {
	if view.len == 0 {
		return nil, nil
	}
	if view.ptr == nil {
		return nil, fmt.Errorf("%s is null with non-zero length", what)
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(view.ptr)), int(view.len))
	out := make([]byte, len(bytes))
	copy(out, bytes)
	return out, nil
}

func vectorStrviewToString(view C.velr_strview, what string) (string, error) {
	bytes, err := vectorStrviewBytes(view, what)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func vectorPurposeFromRaw(raw C.velr_vector_embedding_purpose) VectorEmbeddingPurpose {
	switch raw {
	case C.VELR_VECTOR_EMBEDDING_QUERY:
		return VectorEmbeddingQuery
	default:
		return VectorEmbeddingIndexEntity
	}
}

func vectorEntityKindFromRaw(raw C.velr_vector_entity_kind) VectorEntityKind {
	switch raw {
	case C.VELR_VECTOR_ENTITY_NODE:
		return VectorEntityNode
	case C.VELR_VECTOR_ENTITY_RELATIONSHIP:
		return VectorEntityRelationship
	default:
		return VectorEntityNone
	}
}
