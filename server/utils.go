package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode"

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
