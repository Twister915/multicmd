package main

import "strings"

func indent(in, indent string) string {
	lines := strings.Split(in, "\n")
	outLines := make([]string, len(lines))
	for i, l := range lines {
		outLines[i] = indent + l
	}

	return strings.Join(outLines, "\n")
}
