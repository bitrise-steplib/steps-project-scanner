package main

import "strings"

func redactURL(url string) string {
	firstSeparator := "://"
	firstIndex := strings.Index(url, firstSeparator)
	secondIndex := strings.Index(url, "@")

	if 0 < firstIndex && 0 < secondIndex && firstIndex < secondIndex {
		url = url[:firstIndex+len(firstSeparator)] + "..." + url[secondIndex:]
	}

	return url
}
