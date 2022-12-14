package slice

// Contains checks if a string is present in a slice.
func Contains(slice []string, element string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}

	return false
}
