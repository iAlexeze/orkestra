package note

import "text/template"

// listMapNotes registers list/map manipulation helpers.
//
// Usage:
//
//	tmpl.Funcs(note.listMapNotes())
//
// Template examples:
//
//	{{ listHas .spec.regions "us-east-1" }}
//	{{ listGet .spec.regions 0 }}
//	{{ mapGet .metadata.labels "app" }}
//	{{ mapKeys .metadata.labels }}
//	{{ mapValues .metadata.labels }}
func listMapNotes() template.FuncMap {
	return template.FuncMap{
		"listHas":   noteListHas,
		"listGet":   noteListGet,
		"listLen":   noteListLen,
		"mapGet":    noteMapGet,
		"mapKeys":   noteMapKeys,
		"mapValues": noteMapValues,
	}
}

// noteListHas checks if a list contains a value.
//
//	{{ listHas .spec.regions "us-east-1" }}
func noteListHas(list interface{}, val interface{}) bool {
	items, ok := list.([]interface{})
	if !ok {
		return false
	}
	for _, v := range items {
		if v == val {
			return true
		}
	}
	return false
}

// noteListGet returns list[index] safely.
//
//	{{ listGet .spec.regions 0 }}
func noteListGet(list interface{}, index int) interface{} {
	items, ok := list.([]interface{})
	if !ok || index < 0 || index >= len(items) {
		return nil
	}
	return items[index]
}

// noteListLen returns the length of a list.
//
//	{{ listLen .spec.regions }}
func noteListLen(list interface{}) int {
	items, ok := list.([]interface{})
	if !ok {
		return 0
	}
	return len(items)
}

// noteMapGet returns map[key] safely.
//
//	{{ mapGet .metadata.labels "app" }}
func noteMapGet(m interface{}, key string) interface{} {
	mp, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	return mp[key]
}

// noteMapKeys returns all keys of a map.
//
//	{{ mapKeys .metadata.labels }}
func noteMapKeys(m interface{}) []string {
	mp, ok := m.(map[string]interface{})
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(mp))
	for k := range mp {
		keys = append(keys, k)
	}
	return keys
}

// noteMapValues returns all values of a map.
//
//	{{ mapValues .metadata.labels }}
func noteMapValues(m interface{}) []interface{} {
	mp, ok := m.(map[string]interface{})
	if !ok {
		return []interface{}{}
	}
	values := make([]interface{}, 0, len(mp))
	for _, v := range mp {
		values = append(values, v)
	}
	return values
}
