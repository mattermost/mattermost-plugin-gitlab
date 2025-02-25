// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/pkg/errors"
)

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

func encrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	msg := pad([]byte(text))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], msg)
	finalMsg := base64.URLEncoding.EncodeToString(ciphertext)
	return finalMsg, nil
}

func decrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	decodedMsg, err := base64.URLEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}

	if (len(decodedMsg) % aes.BlockSize) != 0 {
		return "", errors.New("blocksize must be multiple of decoded message length")
	}

	iv := decodedMsg[:aes.BlockSize]
	msg := decodedMsg[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := unpad(msg)
	if err != nil {
		return "", err
	}

	return string(unpadMsg), nil
}

// normalizePath is responsible for parsing GitLab project URL leaving only <GROUP>/<SUBGROUP>/<REPO> components.
func normalizePath(full, baseURL string) string {
	if baseURL == "" {
		baseURL = "https://gitlab.com/"
	} else if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return strings.TrimSuffix(strings.TrimSpace(strings.Replace(full, baseURL, "", 1)), "/")
}

func parseGitlabUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	for _, word := range strings.FieldsFunc(text, func(c rune) bool {
		return !(c == '-' || c == '@' || unicode.IsLetter(c) || unicode.IsNumber(c))
	}) {
		if len(word) < 2 || word[0] != '@' {
			continue
		}

		if word[1] == '-' || word[len(word)-1] == '-' {
			continue
		}

		if strings.Contains(word, "--") {
			continue
		}

		name := word[1:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func fullPathFromNamespaceAndProject(namespace, project string) string {
	return fmt.Sprintf("%s/%s", namespace, project)
}

// isValidURL checks if a given URL is a valid URL with a host and a http or http scheme.
func isValidURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Errorf("URL schema must either be %q or %q", "http", "https")
	}

	if u.Host == "" {
		return errors.New("URL must contain a host")
	}

	return nil
}

func getPluginURL(client *pluginapi.Client) string {
	return getSiteURL(client) + "/" + path.Join("plugins", manifest.Id)
}

func getSiteURL(client *pluginapi.Client) string {
	siteURL := client.Configuration.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil {
		return ""
	}

	return strings.TrimSuffix(*siteURL, "/")
}

// filterLines filters lines in a string from start to end.
func filterLines(s string, start, end int) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var buf strings.Builder
	for i := 1; scanner.Scan() && i <= end; i++ {
		if i < start {
			continue
		}
		buf.Write(scanner.Bytes())
		buf.WriteByte(byte('\n'))
	}

	return buf.String(), scanner.Err()
}

// getLineNumbers returns the start and end lines from an anchor tag
// of a GitLab permalink.
func getLineNumbers(s string) (start, end int) {
	// split till -
	parts := strings.Split(s, "-")

	if len(parts) > 2 {
		return -1, -1
	}

	switch len(parts) {
	case 1:
		// just a single line
		l := getLine(parts[0])
		if l == -1 {
			return -1, -1
		}
		if l < permalinkLineContext {
			return 0, l + permalinkLineContext
		}
		return l - permalinkLineContext, l + permalinkLineContext
	case 2:
		// a line range
		start := getLine(parts[0])
		end := getLine(parts[1])
		if start != -1 && end != -1 && start > end {
			return -1, -1
		}
		return start, end
	}
	return -1, -1
}

// getLine returns the line number in int from a string
// of form L<num>.
func getLine(s string) int {
	// check starting L and minimum length.
	if !strings.HasPrefix(s, "L") || len(s) < 2 {
		return -1
	}

	line, err := strconv.Atoi(s[1:])
	if err != nil {
		return -1
	}
	return line
}

// isInsideLink reports whether the given index in a string is preceded
// by zero or more space, then (, then ].
//
// It is a poor man's version of checking markdown hyperlinks without
// using a full-blown markdown parser. The idea is to quickly confirm
// whether a permalink is inside a markdown link or not. Something like
// "text ]( permalink" is rare enough. Even then, it is okay if
// there are false positives, but there cannot be any false negatives.
//
// Note: it is fine to go one byte at a time instead of one rune because
// we are anyways looking for ASCII chars.
// Ref: mattermost plugin github (https://github.com/mattermost/mattermost-plugin-github/blob/fcc50523c17c2670cb595a8038864a8a7f7fd1e5/server/plugin/utils.go#L273)
func isInsideLink(msg string, index int) bool {
	stage := 0 // 0 is looking for space or ( and 1 for ]

	for i := index; i > 0; i-- {
		char := msg[i-1]
		switch stage {
		case 0:
			if char == ' ' {
				continue
			}
			if char == '(' {
				stage++
				continue
			}
			return false
		case 1:
			if char == ']' {
				return true
			}
			return false
		}
	}
	return false
}

// getCodeMarkdown returns the constructed markdown for a permalink.
func getCodeMarkdown(user, repo, repoPath, word, lines string, isTruncated bool) string {
	final := fmt.Sprintf("\n[%s/%s/%s](%s)\n", user, repo, repoPath, word)
	ext := path.Ext(repoPath)
	// remove the preceding dot
	if len(ext) > 1 {
		ext = strings.TrimPrefix(ext, ".")
	}
	final += "```" + ext + "\n"
	final += lines
	if isTruncated { // add an ellipsis if lines were cut off
		final += "...\n"
	}
	final += "```\n"
	return final
}

// lastN returns the last n characters of a string, with the rest replaced by *.
// At most 3 characters are replaced. The rest is cut off.
func lastN(s string, n int) string {
	if n < 0 {
		return ""
	}

	out := []byte(s)
	if len(out) > n+3 {
		out = out[len(out)-n-3:]
	}
	for i := range out {
		if i < len(out)-n {
			out[i] = '*'
		}
	}

	return string(out)
}
