package main

import "regexp"

var (
	reChapterID   = regexp.MustCompile(`chapterid\s*=\s*(\d+)`)
	reImageCount  = regexp.MustCompile(`imagecount\s*=\s*(\d+)`)
	rePix         = regexp.MustCompile(`pix\s*=\s*["']([^"']+)["']`)
	rePath        = regexp.MustCompile(`["'](/[^"']+\.(?:jpg|jpeg|png|webp|gif)(?:\?[^"']*)?)["']`)
	reRelative    = regexp.MustCompile(`(?i)^(\d+)\s+(hour|day|week|month)s?\s+ago`)
	reChapterNum  = regexp.MustCompile(`(?i)(?:ch\.?|chapter)\s*(\d+(?:\.\d+)?)`)
	reFallbackNum = regexp.MustCompile(`(\d+(?:\.\d+)?)`)
)
