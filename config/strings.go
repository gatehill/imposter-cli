package config

func GetFirstNonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// CombineUnique returns the unique union of originals and candidates
func CombineUnique(originals []string, candidates []string) []string {
	var merged = originals
	for _, plugin := range candidates {
		found := false
		for _, existing := range originals {
			if existing == plugin {
				found = true
			}
		}
		if !found {
			merged = append(merged, plugin)
		}
	}
	return merged
}
