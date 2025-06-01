package utils

import "strings"

// truncateText は文字列を指定された最大長に切り詰めます。
func TruncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return s
}

// truncateTextWithEllipsis は文字列を指定された最大文字数または最大行数に切り詰め、
// 必要に応じて省略記号を追加します。
func TruncateTextWithEllipsis(s string, maxChars int, maxLines int) string {
	var resultLines []string
	lineCount := 0
	charCount := 0
	currentLine := ""
	runes := []rune(s)

	for _, r := range runes {
		if maxChars > 0 && charCount >= maxChars-3 {
			if len(currentLine) > 0 || len(resultLines) < maxLines {
				currentLine += "..."
				if len(currentLine) > 0 {
					resultLines = append(resultLines, currentLine)
					currentLine = ""
				}
			}
			goto endLoop
		}

		if r == '\n' {
			resultLines = append(resultLines, currentLine)
			currentLine = ""
			lineCount++
			if maxLines > 0 && lineCount >= maxLines {
				if len(resultLines) > 0 {
					lastIdx := len(resultLines) - 1
					if !strings.HasSuffix(resultLines[lastIdx], "...") {
						resultLines[lastIdx] += "..."
					}
				}
				goto endLoop
			}
		} else {
			currentLine += string(r)
			charCount++
		}
	}
	if currentLine != "" {
		resultLines = append(resultLines, currentLine)
	}

endLoop:
	if maxLines > 0 && len(resultLines) > maxLines {
		resultLines = resultLines[:maxLines]
		if len(resultLines) > 0 {
			lastIdx := len(resultLines) - 1
			if !strings.HasSuffix(resultLines[lastIdx], "...") {
				if len([]rune(resultLines[lastIdx])) <= maxChars-3 || maxChars == 0 {
					resultLines[lastIdx] += "..."
				} else if maxChars > 0 {
					runesLastLine := []rune(resultLines[lastIdx])
					if len(runesLastLine) > maxChars-3 {
						resultLines[lastIdx] = string(runesLastLine[:maxChars-3]) + "..."
					}
				}
			}
		}
	}

	finalStr := strings.Join(resultLines, "\n")
	return finalStr
}
