package redis

import "strings"

// parseInfo parses Redis INFO command output into a map
func parseInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\n")

	for _, line := range lines {
		// Trim whitespace and carriage returns
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return result
}

// parseInt64 parses a string to int64
func parseInt64(s string) int64 {
	var result int64
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			result = result*10 + int64(s[i]-'0')
		}
	}
	return result
}

// parseFloat64 parses a string to float64
func parseFloat64(s string) float64 {
	var result float64
	var fraction float64
	var divisor float64 = 1.0
	inFraction := false

	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			inFraction = true
			continue
		}
		if s[i] >= '0' && s[i] <= '9' {
			digit := float64(s[i] - '0')
			if inFraction {
				divisor *= 10.0
				fraction = fraction*10.0 + digit
			} else {
				result = result*10.0 + digit
			}
		}
	}

	if inFraction && divisor > 1.0 {
		result += fraction / divisor
	}

	return result
}

// formatSeconds formats seconds into a human-readable string
func formatSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	days := seconds / 86400
	seconds %= 86400
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60
	seconds %= 60

	var parts []string
	if days > 0 {
		parts = append(parts, formatInt(days)+"d")
	}
	if hours > 0 {
		parts = append(parts, formatInt(hours)+"h")
	}
	if minutes > 0 {
		parts = append(parts, formatInt(minutes)+"m")
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, formatInt(seconds)+"s")
	}

	return joinStrings(parts, " ")
}

// formatInt formats an int64 to string
func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append(digits, byte(n%10)+'0')
		n /= 10
	}

	// Reverse
	for i := 0; i < len(digits)/2; i++ {
		digits[i], digits[len(digits)-1-i] = digits[len(digits)-1-i], digits[i]
	}

	return string(digits)
}

// joinStrings joins string slices with a separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}

	totalLen := len(parts) - 1
	for _, p := range parts {
		totalLen += len(p)
	}

	result := make([]byte, 0, totalLen)
	for i, p := range parts {
		if i > 0 {
			result = append(result, sep...)
		}
		result = append(result, p...)
	}

	return string(result)
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(bytes) + "B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"K", "M", "G", "T", "P", "E"}

	// Calculate the value with 2 decimal places
	val := float64(bytes) / float64(div)

	// Format as string with 2 decimal places
	intPart := int64(val)
	fracPart := int64((val - float64(intPart)) * 100)

	result := formatInt(intPart) + "."
	if fracPart < 10 {
		result += "0"
	}
	result += formatInt(fracPart) + units[exp]

	return result
}
