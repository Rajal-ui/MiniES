package search

func BitapSearch(text string, pattern string, maxDistance int) bool {
	return BitapScore(text, pattern, maxDistance) <= maxDistance
}

func BitapScore(text string, pattern string, maxDistance int) int {
	m := len(pattern)
	n := len(text)
	if m == 0 {
		return 0
	}
	if maxDistance > m {
		maxDistance = m
	}

	bestDist := maxDistance + 1
	for i := 0; i <= n-m; i++ {
		sub := text[i : i+m]
		dist := damerauLevenshtein(sub, pattern)
		if dist < bestDist {
			bestDist = dist
		}
	}
	if bestDist <= maxDistance {
		return bestDist
	}
	return maxDistance + 1
}

// damerauLevenshtein is Levenshtein distance with adjacent transpositions
// counting as a single edit (so "tiemout" matches "timeout" at distance 1).
func damerauLevenshtein(s1 string, s2 string) int {
	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
			if i > 1 && j > 1 && s1[i-1] == s2[j-2] && s1[i-2] == s2[j-1] {
				dp[i][j] = min(dp[i][j], dp[i-2][j-2]+1)
			}
		}
	}
	return dp[m][n]
}

func LevenshteinDistance(s1 string, s2 string) int {
	m, n := len(s1), len(s2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[m][n]
}
