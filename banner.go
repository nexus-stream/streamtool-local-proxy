package main

import (
	"strings"
)

// font renders each letter as 5 rows of 5 columns. Rows are stored with
// trailing spaces so every letter is the same width.
var font = map[rune][5]string{
	'A': {" ### ", "#   #", "#####", "#   #", "#   #"},
	'C': {" ####", "#    ", "#    ", "#    ", " ####"},
	'D': {"#### ", "#   #", "#   #", "#   #", "#### "},
	'E': {"#####", "#    ", "#### ", "#    ", "#####"},
	'L': {"#    ", "#    ", "#    ", "#    ", "#####"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'O': {" ### ", "#   #", "#   #", "#   #", " ### "},
	'P': {"#### ", "#   #", "#### ", "#    ", "#    "},
	'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'S': {" ####", "#    ", " ### ", "    #", "#### "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'V': {"#   #", "#   #", "#   #", "#   #", " # # "},
}

// renderWord draws word in the block font, with two spaces between letters.
func renderWord(word string) string {
	var rows [5]string
	for i := 0; i < 5; i++ {
		parts := make([]string, 0, len(word))
		for _, r := range word {
			parts = append(parts, font[r][i])
		}
		rows[i] = strings.Join(parts, "  ")
	}
	return strings.Join(rows[:], "\n")
}

func labelFor(e environment) string {
	switch e.name {
	case "staging":
		return "DEVELOP"
	case "prod":
		return "RELEASE"
	default:
		return "CUSTOM"
	}
}
