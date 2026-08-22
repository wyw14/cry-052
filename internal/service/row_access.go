package service

import (
	"reflect"
	"strings"
)

func lookupMappedValue(row Row, name string) (any, bool) {
	if value, exists := row[name]; exists {
		return value, true
	}
	// Catalog field names may use different casing than the keys carried by the
	// sample rows (e.g. schema field "Tags" vs. sample key "tags"). The compiler
	// already resolves mappings case-insensitively via TableSchema.Field, so the
	// row lookup must do the same to avoid falsely reporting the field as missing.
	lower := strings.ToLower(name)
	for key, value := range row {
		if strings.ToLower(key) == lower {
			return value, true
		}
	}
	return nil, false
}

func preserveMappedValue(value any) any {
	return cloneValue(value)
}

// cloneValue returns a deep copy of value whose slices, maps, and pointers share
// no mutable state with the original. Scalars and other immutable values are
// returned as-is. This keeps preview output isolated from the source sample
// rows, so a consumer that edits a keep-rule output (such as a tag list) cannot
// mutate the original sample through a shared reference.
func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() {
			return value
		}
		clone := reflect.MakeSlice(val.Type(), val.Len(), val.Len())
		for i := 0; i < val.Len(); i++ {
			clone.Index(i).Set(reflect.ValueOf(cloneValue(val.Index(i).Interface())))
		}
		return clone.Interface()
	case reflect.Map:
		if val.IsNil() {
			return value
		}
		clone := reflect.MakeMapWithSize(val.Type(), val.Len())
		for iter := val.MapRange(); iter.Next(); {
			clone.SetMapIndex(iter.Key(), reflect.ValueOf(cloneValue(iter.Value().Interface())))
		}
		return clone.Interface()
	case reflect.Ptr:
		if val.IsNil() {
			return value
		}
		clone := reflect.New(val.Elem().Type())
		clone.Elem().Set(reflect.ValueOf(cloneValue(val.Elem().Interface())))
		return clone.Interface()
	default:
		return value
	}
}
