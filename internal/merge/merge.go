package merge

// Maps recursively merges overlay into base. Lists, scalars and nil replace
// the previous value. Inputs are copied and never mutated.
func Maps(base, overlay map[string]any) map[string]any {
	result := cloneMap(base)
	for key, overlayValue := range overlay {
		baseValue, exists := result[key]
		overlayMap, overlayIsMap := overlayValue.(map[string]any)
		baseMap, baseIsMap := baseValue.(map[string]any)
		if exists && overlayIsMap && baseIsMap {
			result[key] = Maps(baseMap, overlayMap)
			continue
		}
		result[key] = clone(overlayValue)
	}
	return result
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = clone(value)
	}
	return result
}

func clone(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = clone(typed[i])
		}
		return result
	default:
		return typed
	}
}
