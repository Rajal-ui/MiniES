package search

func KMP(text string, pattern string) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) < len(pattern) {
		return false
	}
	lps := computeLPS(pattern)
	i, j := 0, 0
	for i < len(text) {
		if text[i] == pattern[j] {
			i++
			j++
			if j == len(pattern) {
				return true
			}
		} else if j > 0 {
			j = lps[j-1]
		} else {
			i++
		}
	}
	return false
}

func computeLPS(pattern string) []int {
	lps := make([]int, len(pattern))
	length := 0
	i := 1
	for i < len(pattern) {
		if pattern[i] == pattern[length] {
			length++
			lps[i] = length
			i++
		} else if length > 0 {
			length = lps[length-1]
		} else {
			lps[i] = 0
			i++
		}
	}
	return lps
}

func KMPSearch(text string, pattern string) []int {
	var positions []int
	if len(pattern) == 0 {
		return positions
	}
	lps := computeLPS(pattern)
	i, j := 0, 0
	for i < len(text) {
		if text[i] == pattern[j] {
			i++
			j++
			if j == len(pattern) {
				positions = append(positions, i-j)
				j = lps[j-1]
			}
		} else if j > 0 {
			j = lps[j-1]
		} else {
			i++
		}
	}
	return positions
}
