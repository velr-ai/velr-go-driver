package velr

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// PropertyValue is a typed Velr property value.
//
// Temporal values are exposed in their canonical openCypher text form. Spatial
// values are exposed as GeoJSON. Use GoValue when a plain Go representation is
// more convenient than switching on Type.
type PropertyValue struct {
	// Type identifies which value field is populated.
	Type PropertyValueType

	// Bool is set when Type is PropertyBool.
	Bool bool

	// Int64 is set when Type is PropertyInt64.
	Int64 int64

	// Float64 is set when Type is PropertyDouble.
	Float64 float64

	// Text is set for string, temporal, and duration values.
	Text string

	// List is set when Type is PropertyList.
	List []PropertyValue

	// Vector is set when Type is PropertyVector.
	Vector []float64

	// Bytes is set when Type is PropertyBytes.
	Bytes []byte

	// GeoJSON is set for spatial point, geometry, and geography values.
	GeoJSON *GeoJSON

	// Display is Velr's canonical display rendering when one is available.
	Display string
}

// Kind returns the Velr property value kind.
func (v PropertyValue) Kind() PropertyValueType {
	return v.Type
}

// GoValue converts the value to an idiomatic Go representation.
func (v PropertyValue) GoValue() any {
	switch v.Type {
	case PropertyNull:
		return nil
	case PropertyBool:
		return v.Bool
	case PropertyInt64:
		return v.Int64
	case PropertyDouble:
		return v.Float64
	case PropertyString, PropertyDate, PropertyLocalTime, PropertyZonedTime,
		PropertyLocalDateTime, PropertyZonedDateTime, PropertyDuration:
		return v.Text
	case PropertyPoint, PropertyGeometry, PropertyGeography:
		if v.GeoJSON == nil {
			return nil
		}
		return *v.GeoJSON
	case PropertyList:
		out := make([]any, len(v.List))
		for i, item := range v.List {
			out[i] = item.GoValue()
		}
		return out
	case PropertyVector:
		return append([]float64(nil), v.Vector...)
	case PropertyBytes:
		return append([]byte(nil), v.Bytes...)
	default:
		if v.Display != "" {
			return v.Display
		}
		return nil
	}
}

// String returns Velr's display text when available, otherwise a Go rendering.
func (v PropertyValue) String() string {
	if v.Display != "" {
		return v.Display
	}
	switch value := v.GoValue().(type) {
	case nil:
		return "null"
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

// Node is a decoded Velr graph node value.
type Node struct {
	// Identity is the runtime-local numeric node identifier.
	Identity int64

	// ElementID is the stable textual node identifier exposed by Cypher.
	ElementID string

	// Labels contains the node labels in database order.
	Labels []string

	// Properties contains decoded node properties keyed by property name.
	Properties map[string]PropertyValue
}

// Relationship is a decoded Velr graph relationship value.
type Relationship struct {
	// Identity is the runtime-local numeric relationship identifier.
	Identity int64

	// ElementID is the stable textual relationship identifier exposed by Cypher.
	ElementID string

	// Type is the relationship type.
	Type string

	// Start is the runtime-local numeric identifier of the start node.
	Start int64

	// End is the runtime-local numeric identifier of the end node.
	End int64

	// StartElementID is the stable textual identifier of the start node.
	StartElementID string

	// EndElementID is the stable textual identifier of the end node.
	EndElementID string

	// Properties contains decoded relationship properties keyed by property name.
	Properties map[string]PropertyValue
}

// Path is a decoded Velr graph path.
type Path struct {
	// Elements alternates node and relationship entries in path order.
	Elements []PathElement
}

// Nodes returns the node elements in path order.
func (p Path) Nodes() []Node {
	out := make([]Node, 0, (len(p.Elements)+1)/2)
	for _, elem := range p.Elements {
		if elem.Node != nil {
			out = append(out, *elem.Node)
		}
	}
	return out
}

// Relationships returns the relationship elements in path order.
func (p Path) Relationships() []Relationship {
	out := make([]Relationship, 0, len(p.Elements)/2)
	for _, elem := range p.Elements {
		if elem.Relationship != nil {
			out = append(out, *elem.Relationship)
		}
	}
	return out
}

// PathElement is one node or relationship in a decoded path.
type PathElement struct {
	// Node is populated when this path element is a node.
	Node *Node

	// Relationship is populated when this path element is a relationship.
	Relationship *Relationship
}

// GeoJSON is a spatial value represented in Velr's canonical JSON form.
type GeoJSON struct {
	// Type is the GeoJSON geometry type, such as Point or GeometryCollection.
	Type string

	// Coordinates contains the GeoJSON coordinates for non-collection values.
	Coordinates any

	// Geometries contains nested geometries for GeometryCollection values.
	Geometries []GeoJSON

	// Raw contains the decoded canonical GeoJSON object.
	Raw map[string]any
}

// DecodePropertyJSON decodes Velr's canonical JSON cell representation.
//
// Graph nodes, relationships, paths, and GeoJSON spatial values are returned
// as Node, Relationship, Path, and GeoJSON. Scalar values and lists are
// returned as PropertyValue. Generic objects are returned as map[string]any
// with decoded field values.
func DecodePropertyJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("property JSON contains multiple top-level values")
		}
		return nil, fmt.Errorf("property JSON contains multiple top-level values")
	}
	return decodeResultValue(raw)
}

// AsProperty returns this cell as a decoded Velr value.
//
// JSON cells are decoded with DecodePropertyJSON. Text cells stay text even
// when they begin with JSON-looking characters.
func (c Cell) AsProperty() (any, error) {
	switch c.Type {
	case JSON:
		return DecodePropertyJSON(c.Bytes)
	case Text:
		if value, ok, err := decodeCanonicalTextValue(c.Bytes); ok || err != nil {
			return value, err
		}
		return c.AsPropertyValue()
	default:
		return c.AsPropertyValue()
	}
}

// AsPropertyValue converts this cell to a typed Velr property value.
//
// Graph result values such as nodes, relationships, and paths are not property
// values. For those JSON cells, use AsProperty and type-assert the graph value.
func (c Cell) AsPropertyValue() (PropertyValue, error) {
	switch c.Type {
	case Null:
		return PropertyValue{Type: PropertyNull}, nil
	case Bool:
		return PropertyValue{Type: PropertyBool, Bool: c.Int64 != 0}, nil
	case Int64:
		return PropertyValue{Type: PropertyInt64, Int64: c.Int64}, nil
	case Double:
		return PropertyValue{Type: PropertyDouble, Float64: c.Float64}, nil
	case Text:
		text := string(c.Bytes)
		return PropertyValue{Type: PropertyString, Text: text, Display: text}, nil
	case JSON:
		value, err := DecodePropertyJSON(c.Bytes)
		if err != nil {
			return PropertyValue{}, err
		}
		if property, ok := value.(PropertyValue); ok {
			return property, nil
		}
		return PropertyValue{}, fmt.Errorf("json cell contains %T, not a property value", value)
	default:
		return PropertyValue{}, fmt.Errorf("unknown cell type %s", c.Type)
	}
}

func decodeCanonicalTextValue(data []byte) (any, bool, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil, false, nil
	}
	if !bytes.Contains(trimmed, []byte(`"__velr$type"`)) {
		return nil, false, nil
	}
	value, err := DecodePropertyJSON(trimmed)
	return value, true, err
}

func decodeResultValue(value any) (any, error) {
	switch v := value.(type) {
	case nil, bool, string:
		return propertyValueFromJSON(v)
	case json.Number:
		return propertyValueFromJSON(v)
	case []any:
		return decodeResultArray(v)
	case map[string]any:
		return decodeResultObject(v)
	default:
		return v, nil
	}
}

func decodeResultArray(values []any) (any, error) {
	out := make([]any, len(values))
	for i, value := range values {
		decoded, err := decodeResultValue(value)
		if err != nil {
			return nil, fmt.Errorf("array element %d: %w", i, err)
		}
		out[i] = decoded
	}
	if path, ok := pathFromElements(out); ok {
		return path, nil
	}
	property, err := propertyListFromJSON(values)
	if err != nil {
		return nil, err
	}
	return property, nil
}

func decodeResultObject(obj map[string]any) (any, error) {
	if looksLikeNode(obj) {
		return nodeFromMap(obj)
	}
	if looksLikeRelationship(obj) {
		return relationshipFromMap(obj)
	}
	if geo, ok := geoJSONFromRawMap(obj); ok {
		return geo, nil
	}

	out := make(map[string]any, len(obj))
	for key, value := range obj {
		decoded, err := decodeResultValue(value)
		if err != nil {
			return nil, fmt.Errorf("object field %q: %w", key, err)
		}
		out[key] = decoded
	}

	if geo, ok := geoJSONFromMap(out); ok {
		return geo, nil
	}
	return out, nil
}

func propertyValueFromJSON(value any) (PropertyValue, error) {
	switch v := value.(type) {
	case nil:
		return PropertyValue{Type: PropertyNull}, nil
	case bool:
		return PropertyValue{Type: PropertyBool, Bool: v, Display: fmt.Sprint(v)}, nil
	case string:
		return PropertyValue{Type: PropertyString, Text: v, Display: v}, nil
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return PropertyValue{Type: PropertyInt64, Int64: i, Display: v.String()}, nil
		}
		f, err := v.Float64()
		if err != nil {
			return PropertyValue{}, err
		}
		return PropertyValue{Type: PropertyDouble, Float64: f, Display: v.String()}, nil
	case []any:
		return propertyListFromJSON(v)
	case map[string]any:
		if decoded, err := decodeResultObject(v); err == nil {
			if geo, ok := decoded.(GeoJSON); ok {
				return PropertyValue{Type: propertyTypeFromGeoJSON(geo), GeoJSON: &geo, Display: geo.Type}, nil
			}
		}
		return PropertyValue{}, fmt.Errorf("objects are not Velr property values")
	default:
		return PropertyValue{}, fmt.Errorf("unsupported JSON property value %T", value)
	}
}

func propertyValueFromJSONWithType(value any, typ PropertyValueType, display string) (PropertyValue, error) {
	switch typ {
	case PropertyNull:
		return PropertyValue{Type: PropertyNull, Display: display}, nil
	case PropertyBool:
		v, ok := value.(bool)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected bool JSON for %s, got %T", typ, value)
		}
		return PropertyValue{Type: typ, Bool: v, Display: display}, nil
	case PropertyInt64:
		n, ok := value.(json.Number)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected number JSON for %s, got %T", typ, value)
		}
		i, err := n.Int64()
		if err != nil {
			return PropertyValue{}, fmt.Errorf("expected integer JSON for %s: %w", typ, err)
		}
		return PropertyValue{Type: typ, Int64: i, Display: display}, nil
	case PropertyDouble:
		n, ok := value.(json.Number)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected number JSON for %s, got %T", typ, value)
		}
		f, err := n.Float64()
		if err != nil {
			return PropertyValue{}, fmt.Errorf("expected float JSON for %s: %w", typ, err)
		}
		return PropertyValue{Type: typ, Float64: f, Display: display}, nil
	case PropertyString, PropertyDate, PropertyLocalTime, PropertyZonedTime,
		PropertyLocalDateTime, PropertyZonedDateTime, PropertyDuration:
		text, ok := value.(string)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected string JSON for %s, got %T", typ, value)
		}
		if display == "" {
			display = text
		}
		return PropertyValue{Type: typ, Text: text, Display: display}, nil
	case PropertyPoint, PropertyGeometry, PropertyGeography:
		obj, ok := value.(map[string]any)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected GeoJSON object for %s, got %T", typ, value)
		}
		decoded, err := decodeResultObject(obj)
		if err != nil {
			return PropertyValue{}, err
		}
		geo, ok := decoded.(GeoJSON)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected GeoJSON object for %s", typ)
		}
		return PropertyValue{Type: typ, GeoJSON: &geo, Display: display}, nil
	case PropertyList:
		values, ok := value.([]any)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected JSON array for %s, got %T", typ, value)
		}
		out, err := propertyListFromJSON(values)
		if err != nil {
			return PropertyValue{}, err
		}
		out.Display = display
		return out, nil
	case PropertyVector:
		values, ok := value.([]any)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected JSON array for %s, got %T", typ, value)
		}
		vector, err := vectorFromJSON(values)
		if err != nil {
			return PropertyValue{}, err
		}
		return PropertyValue{Type: typ, Vector: vector, Display: display}, nil
	case PropertyBytes:
		text, ok := value.(string)
		if !ok {
			return PropertyValue{}, fmt.Errorf("expected string JSON for %s, got %T", typ, value)
		}
		bytes, err := hex.DecodeString(text)
		if err != nil {
			bytes = []byte(text)
		}
		return PropertyValue{Type: typ, Bytes: bytes, Display: display}, nil
	default:
		property, err := propertyValueFromJSON(value)
		if err != nil {
			return PropertyValue{}, err
		}
		property.Display = display
		return property, nil
	}
}

func propertyListFromJSON(values []any) (PropertyValue, error) {
	out := make([]PropertyValue, len(values))
	for i, value := range values {
		item, err := propertyValueFromJSON(value)
		if err != nil {
			return PropertyValue{}, fmt.Errorf("list element %d: %w", i, err)
		}
		out[i] = item
	}
	return PropertyValue{Type: PropertyList, List: out}, nil
}

func vectorFromJSON(values []any) ([]float64, error) {
	out := make([]float64, len(values))
	for i, value := range values {
		number, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("vector element %d must be a number", i)
		}
		f, err := number.Float64()
		if err != nil {
			return nil, fmt.Errorf("vector element %d: %w", i, err)
		}
		out[i] = f
	}
	return out, nil
}

func propertyTypeFromGeoJSON(geo GeoJSON) PropertyValueType {
	if geo.Type == "Point" {
		return PropertyPoint
	}
	return PropertyGeometry
}

func propertyValueFromStorage(typ PropertyValueType, storage StorageValueType, i64 int64, f64 float64, raw []byte, rawJSON []byte, display string) (PropertyValue, error) {
	if len(rawJSON) != 0 {
		decoder := json.NewDecoder(bytes.NewReader(rawJSON))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return PropertyValue{}, fmt.Errorf("decode %s JSON: %w", typ, err)
		}
		property, err := propertyValueFromJSONWithType(value, typ, display)
		if err != nil {
			return PropertyValue{}, err
		}
		return property, nil
	}

	switch typ {
	case PropertyNull:
		return PropertyValue{Type: typ, Display: display}, nil
	case PropertyBool:
		return PropertyValue{Type: typ, Bool: i64 != 0, Display: display}, nil
	case PropertyInt64:
		return PropertyValue{Type: typ, Int64: i64, Display: display}, nil
	case PropertyDouble:
		return PropertyValue{Type: typ, Float64: f64, Display: display}, nil
	case PropertyString, PropertyDate, PropertyLocalTime, PropertyZonedTime,
		PropertyLocalDateTime, PropertyZonedDateTime, PropertyDuration:
		text := textFromStorage(storage, raw, display)
		return PropertyValue{Type: typ, Text: text, Display: display}, nil
	case PropertyBytes:
		bytes := append([]byte(nil), raw...)
		if storage == StorageBlob && len(bytes) > 0 {
			bytes = bytes[1:]
		}
		return PropertyValue{Type: typ, Bytes: bytes, Display: display}, nil
	default:
		if display != "" {
			return PropertyValue{Type: typ, Display: display}, nil
		}
		return PropertyValue{}, fmt.Errorf("cannot decode %s without JSON payload", typ)
	}
}

func textFromStorage(storage StorageValueType, raw []byte, display string) string {
	switch storage {
	case StorageText:
		return string(raw)
	case StorageBlob:
		if len(raw) > 1 && utf8.Valid(raw[1:]) {
			return string(raw[1:])
		}
		if utf8.Valid(raw) {
			return string(raw)
		}
	}
	return display
}

func looksLikeNode(obj map[string]any) bool {
	_, hasID := obj["identity"]
	_, hasLabels := obj["labels"]
	return hasID && hasLabels
}

func looksLikeRelationship(obj map[string]any) bool {
	_, hasID := obj["identity"]
	_, hasType := obj["type"]
	_, hasStart := obj["start"]
	_, hasEnd := obj["end"]
	return hasID && hasType && hasStart && hasEnd
}

func nodeFromMap(obj map[string]any) (Node, error) {
	identity, err := requiredInt64(obj, "identity")
	if err != nil {
		return Node{}, err
	}
	labels, err := stringSliceField(obj, "labels")
	if err != nil {
		return Node{}, err
	}
	properties, err := propertiesField(obj)
	if err != nil {
		return Node{}, err
	}
	elementID, err := optionalString(obj, "elementId")
	if err != nil {
		return Node{}, err
	}
	return Node{
		Identity:   identity,
		ElementID:  elementID,
		Labels:     labels,
		Properties: properties,
	}, nil
}

func relationshipFromMap(obj map[string]any) (Relationship, error) {
	identity, err := requiredInt64(obj, "identity")
	if err != nil {
		return Relationship{}, err
	}
	relType, err := requiredString(obj, "type")
	if err != nil {
		return Relationship{}, err
	}
	start, err := requiredInt64(obj, "start")
	if err != nil {
		return Relationship{}, err
	}
	end, err := requiredInt64(obj, "end")
	if err != nil {
		return Relationship{}, err
	}
	properties, err := propertiesField(obj)
	if err != nil {
		return Relationship{}, err
	}
	elementID, err := optionalString(obj, "elementId")
	if err != nil {
		return Relationship{}, err
	}
	startElementID, err := optionalString(obj, "startElementId")
	if err != nil {
		return Relationship{}, err
	}
	endElementID, err := optionalString(obj, "endElementId")
	if err != nil {
		return Relationship{}, err
	}
	return Relationship{
		Identity:       identity,
		ElementID:      elementID,
		Type:           relType,
		Start:          start,
		End:            end,
		StartElementID: startElementID,
		EndElementID:   endElementID,
		Properties:     properties,
	}, nil
}

func propertiesField(obj map[string]any) (map[string]PropertyValue, error) {
	raw, ok := obj["properties"]
	if !ok || raw == nil {
		return map[string]PropertyValue{}, nil
	}
	props, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("properties must be an object")
	}
	out := make(map[string]PropertyValue, len(props))
	for key, value := range props {
		decoded, err := propertyValueFromJSON(value)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", key, err)
		}
		out[key] = decoded
	}
	return out, nil
}

func pathFromElements(values []any) (Path, bool) {
	if len(values) < 3 || len(values)%2 == 0 {
		return Path{}, false
	}
	elements := make([]PathElement, len(values))
	for i, value := range values {
		if i%2 == 0 {
			node, ok := value.(Node)
			if !ok {
				return Path{}, false
			}
			nodeCopy := node
			elements[i] = PathElement{Node: &nodeCopy}
			continue
		}
		rel, ok := value.(Relationship)
		if !ok {
			return Path{}, false
		}
		relCopy := rel
		elements[i] = PathElement{Relationship: &relCopy}
	}
	return Path{Elements: elements}, true
}

func geoJSONFromMap(obj map[string]any) (GeoJSON, bool) {
	typ, ok := obj["type"].(string)
	if !ok {
		return GeoJSON{}, false
	}
	if !isGeoJSONType(typ) {
		return GeoJSON{}, false
	}
	if _, hasCoordinates := obj["coordinates"]; !hasCoordinates {
		if _, hasGeometries := obj["geometries"]; !hasGeometries {
			return GeoJSON{}, false
		}
	}

	raw := make(map[string]any, len(obj))
	for key, value := range obj {
		raw[key] = value
	}
	geo := GeoJSON{
		Type:        typ,
		Coordinates: obj["coordinates"],
		Raw:         raw,
	}
	if values, ok := obj["geometries"].([]any); ok {
		geo.Geometries = make([]GeoJSON, 0, len(values))
		for _, value := range values {
			child, ok := value.(GeoJSON)
			if !ok {
				return geo, true
			}
			geo.Geometries = append(geo.Geometries, child)
		}
	}
	return geo, true
}

func geoJSONFromRawMap(obj map[string]any) (GeoJSON, bool) {
	typ, ok := obj["type"].(string)
	if !ok || !isGeoJSONType(typ) {
		return GeoJSON{}, false
	}
	if _, hasCoordinates := obj["coordinates"]; !hasCoordinates {
		if _, hasGeometries := obj["geometries"]; !hasGeometries {
			return GeoJSON{}, false
		}
	}

	raw := make(map[string]any, len(obj))
	for key, value := range obj {
		raw[key] = plainJSONValue(value)
	}
	geo := GeoJSON{
		Type:        typ,
		Coordinates: plainJSONValue(obj["coordinates"]),
		Raw:         raw,
	}
	if values, ok := obj["geometries"].([]any); ok {
		geo.Geometries = make([]GeoJSON, 0, len(values))
		for _, value := range values {
			childObj, ok := value.(map[string]any)
			if !ok {
				return geo, true
			}
			child, ok := geoJSONFromRawMap(childObj)
			if !ok {
				return geo, true
			}
			geo.Geometries = append(geo.Geometries, child)
		}
	}
	return geo, true
}

func plainJSONValue(value any) any {
	switch v := value.(type) {
	case json.Number:
		return numberToValue(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = plainJSONValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = plainJSONValue(item)
		}
		return out
	default:
		return v
	}
}

func isGeoJSONType(typ string) bool {
	switch typ {
	case "Point", "LineString", "Polygon", "MultiPoint", "MultiLineString", "MultiPolygon", "GeometryCollection":
		return true
	default:
		return false
	}
}

func stringSliceField(obj map[string]any, name string) ([]string, error) {
	raw, ok := obj[name]
	if !ok {
		return nil, fmt.Errorf("%s is missing", name)
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", name)
	}
	out := make([]string, len(values))
	for i, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string", name, i)
		}
		out[i] = text
	}
	return out, nil
}

func requiredString(obj map[string]any, name string) (string, error) {
	raw, ok := obj[name]
	if !ok {
		return "", fmt.Errorf("%s is missing", name)
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return text, nil
}

func optionalString(obj map[string]any, name string) (string, error) {
	raw, ok := obj[name]
	if !ok || raw == nil {
		return "", nil
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return text, nil
}

func requiredInt64(obj map[string]any, name string) (int64, error) {
	raw, ok := obj[name]
	if !ok {
		return 0, fmt.Errorf("%s is missing", name)
	}
	switch value := raw.(type) {
	case json.Number:
		i, err := value.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer: %w", name, err)
		}
		return i, nil
	case int64:
		return value, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", name)
	}
}

func numberToValue(value json.Number) any {
	if i, err := value.Int64(); err == nil {
		return i
	}
	if f, err := value.Float64(); err == nil {
		return f
	}
	return value.String()
}
