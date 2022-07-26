package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xanzy/go-gitlab"
)

// maxPermalinkReplacements sets the maximum limit to the number of
// permalink replacements that can be performed on a single message.
const maxPermalinkReplacements = 10

const permalinkReqTimeout = 5 * time.Second

// maxPreviewLines sets the maximum number of preview lines that will be shown
// while replacing a permalink.
const maxPreviewLines = 10

// permalinkLineContext shows the number of lines before and after to show
// if the link points to a single line.
const permalinkLineContext = 3

// replacement holds necessary info to replace GitLab permalinks
// in messages with a code preview block.
type replacement struct {
	index         int    // Index of the permalink in the string
	word          string // The permalink
	permalinkData permalinkInfo
}

type permalinkInfo struct { // Holds the necessary metadata of a permalink
	haswww string
	commit string
	user   string
	repo   string
	path   string
	line   string
}

// getPermalinkReplacements returns the permalink replacements that need to be performed
// on a message. The returned slice is sorted by the index in ascending order.
func (p *Plugin) getPermalinkReplacements(msg string) []replacement {
	// Find the permalinks from the msg using a regex
	matches := gitlabPermalinkRegex.FindAllStringSubmatch(msg, -1)
	indices := gitlabPermalinkRegex.FindAllStringIndex(msg, -1)
	var replacements []replacement
	for i, m := range matches {
		// Have a limit on the number of replacements to do
		if i > maxPermalinkReplacements {
			break
		}
		word := m[0]
		index := indices[i][0]
		r := replacement{
			index: index,
			word:  word,
		}
		// Ignore if the word is inside a link
		if isInsideLink(msg, index) {
			continue
		}
		// Populate the permalinkInfo with the extracted groups of the regex
		for j, name := range gitlabPermalinkRegex.SubexpNames() {
			if j == 0 {
				continue
			}
			switch name {
			case "haswww":
				r.permalinkData.haswww = m[j]
			case "user":
				r.permalinkData.user = m[j]
			case "repo":
				r.permalinkData.repo = m[j]
			case "commit":
				r.permalinkData.commit = m[j]
			case "path":
				r.permalinkData.path = m[j]
			case "line":
				r.permalinkData.line = m[j]
			}
		}
		replacements = append(replacements, r)
	}
	return replacements
}

func (p *Plugin) processReplacement(r replacement, glClient *gitlab.Client, wg *sync.WaitGroup, markdownForPermalink []string, index int) {
	defer wg.Done()
	// Quick bailout if the commit hash is not proper.
	if _, err := hex.DecodeString(r.permalinkData.commit); err != nil {
		p.API.LogDebug("Bad git commit hash in permalink", "error", err.Error(), "hash", r.permalinkData.commit)
		return
	}

	// Get the file contents
	opts := gitlab.GetFileOptions{
		Ref: &r.permalinkData.commit,
	}
	projectPath := fmt.Sprintf("%s/%s", r.permalinkData.user, r.permalinkData.repo)
	_, cancel := context.WithTimeout(context.Background(), permalinkReqTimeout)
	file, _, err := glClient.RepositoryFiles.GetFile(projectPath, r.permalinkData.path, &opts)
	defer cancel()
	if err != nil {
		p.API.LogDebug("Error while fetching file contents", "error", err.Error(), "path", r.permalinkData.path)
		return
	}
	// If this is not a file, ignore.
	if file == nil {
		p.API.LogWarn("Permalink is not a file", "file", r.permalinkData.path)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		p.API.LogDebug("Error while decoding file contents", "error", err.Error(), "path", r.permalinkData.path)
		return
	}
	// Get the required lines.
	start, end := getLineNumbers(r.permalinkData.line)
	// Bad anchor tag, ignore.
	if start == -1 || end == -1 {
		return
	}

	isTruncated := false
	if end-start > maxPreviewLines {
		end = start + maxPreviewLines
		isTruncated = true
	}

	lines, err := filterLines(string(decoded), start, end)
	if err != nil {
		p.API.LogDebug("Error while filtering lines", "error", err.Error(), "path", r.permalinkData.path)
	}

	if lines == "" {
		p.API.LogDebug("Line numbers out of range. Skipping.", "file", r.permalinkData.path, "start", start, "end", end)
		return
	}

	markdownForPermalink[index] = getCodeMarkdown(r.permalinkData.user, r.permalinkData.repo, r.permalinkData.path, r.word, lines, isTruncated)
}

// makeReplacements performs the given replacements on the msg and returns
// the new msg. The replacements slice needs to be sorted by the index in ascending order.
func (p *Plugin) makeReplacements(msg string, replacements []replacement, glClient *gitlab.Client) string {
	// Iterating the slice in reverse to preserve the replacement indices.
	wg := new(sync.WaitGroup)
	markdownForPermalink := make([]string, len(replacements))
	for i := len(replacements) - 1; i >= 0; i-- {
		wg.Add(1)
		go p.processReplacement(replacements[i], glClient, wg, markdownForPermalink, i)
	}
	wg.Wait()
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		if markdownForPermalink[i] != "" {
			// Replace word in msg starting from r.index only once.
			msg = msg[:r.index] + strings.Replace(msg[r.index:], r.word, markdownForPermalink[i], 1)
		}
	}
	return msg
}
