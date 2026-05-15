package handlers

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}

	return result
}
