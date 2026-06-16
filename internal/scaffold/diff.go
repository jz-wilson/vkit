package scaffold

import (
	"fmt"
	"strings"
)

// lineDiff returns the count of added (in kit, not current) and removed (in
// current, not kit) lines, plus a simple unified-style hunk string. cur is the
// current file, kit is the template. It uses an LCS so unchanged lines line up.
func lineDiff(cur, kit string) (added, removed int, hunk string) {
	a := splitLines(cur)
	b := splitLines(kit)
	lcs := lcsTable(a, b)

	var sb strings.Builder
	i, j := 0, 0
	// Walk the tables emitting -, +, and context lines.
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			sb.WriteString("  " + a[i] + "\n")
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			sb.WriteString("- " + a[i] + "\n")
			removed++
			i++
		} else {
			sb.WriteString("+ " + b[j] + "\n")
			added++
			j++
		}
	}
	for ; i < len(a); i++ {
		sb.WriteString("- " + a[i] + "\n")
		removed++
	}
	for ; j < len(b); j++ {
		sb.WriteString("+ " + b[j] + "\n")
		added++
	}
	return added, removed, sb.String()
}

// delta renders the "+added −removed" string used in the eval report.
func delta(cur, kit string) string {
	added, removed, _ := lineDiff(cur, kit)
	return fmt.Sprintf("+%d −%d", added, removed)
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// lcsTable builds the LCS length DP table. lcs[i][j] = LCS length of a[i:] and
// b[j:], so the greedy walk above can choose the direction that preserves it.
func lcsTable(a, b []string) [][]int {
	n, m := len(a), len(b)
	t := make([][]int, n+1)
	for i := range t {
		t[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				t[i][j] = t[i+1][j+1] + 1
			} else if t[i+1][j] >= t[i][j+1] {
				t[i][j] = t[i+1][j]
			} else {
				t[i][j] = t[i][j+1]
			}
		}
	}
	return t
}
