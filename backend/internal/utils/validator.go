package utils

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrInvalidBase62Char = errors.New("invalid base62 character")
	ErrInvalidURL        = errors.New("invalid target URL: must be valid http or https URL")
	ErrInvalidAlias      = errors.New("invalid custom alias: only alphanumeric, hyphen, and underscore characters are allowed (3-64 chars)")
	ErrReservedAlias     = errors.New("custom alias is already reserved")
)

var (
	aliasRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	reservedSlugs = map[string]bool{
		"api":       true,
		"auth":      true,
		"urls":      true,
		"health":    true,
		"metrics":   true,
		"static":    true,
		"swagger":   true,
		"docs":      true,
		"admin":     true,
		"dashboard": true,
		"login":     true,
		"register":  true,
		"logout":    true,
	}
)

func IsReservedSlug(slug string) bool {
	return reservedSlugs[strings.ToLower(slug)]
}

func ValidateCustomAlias(alias string) error {
	if alias == "" {
		return nil
	}

	if !aliasRegex.MatchString(alias) {
		return ErrInvalidAlias
	}

	if reservedSlugs[strings.ToLower(alias)] {
		return ErrReservedAlias
	}

	return nil
}

func ValisateTargetURL(target string) error {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return ErrInvalidURL
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return ErrInvalidURL
	}

	scheme := strings.ToLower(parsed.Scheme)
	if !(scheme == "http" || scheme == "https") {
		return ErrInvalidURL
	}

	if parsed.Host == "" {
		return ErrInvalidURL
	}

	return nil
}
