// Package wiki2markdown2wiki converts between JIRA wiki markup and Markdown.
package wiki2markdown2wiki

import "github.com/Funcan/j2m"

// WikiToMarkdown converts JIRA wiki markup to Markdown.
func WikiToMarkdown(wiki string) string {
	return j2m.JiraToMD(wiki)
}

// MarkdownToWiki converts Markdown to JIRA wiki markup.
func MarkdownToWiki(markdown string) string {
	return j2m.MDToJira(markdown)
}
