package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	baseLength  = 62
)

// EncodeBase62 converts a uint64 number to a Base62 string.
func EncodeBase62(num uint64) string {
	if num == 0 {
		return string(base62Chars[0])
	}

	var sb strings.Builder
	for num > 0 {
		remainder := num % uint64(baseLength)
		sb.WriteByte(base62Chars[remainder])
		num /= uint64(baseLength)
	}

	// Reverse the string
	bytes := []byte(sb.String())
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return string(bytes)
}

// DecodeBase62 converts a Base62 string back to a uint64 number.
func DecodeBase62(str string) (uint64, error) {
	var result uint64
	for i := 0; i < len(str); i++ {
		idx := strings.IndexByte(base62Chars, str[i])
		if idx == -1 {
			return 0, ErrInvalidBase62Char
		}
		result = result*uint64(baseLength) + uint64(idx)
	}
	return result, nil
}

// GenerateRandomSlug generates a cryptographically secure random Base62 slug of specified length.
func GenerateRandomSlug(length int) (string, error) {
	if length <= 0 {
		length = 7
	}

	var sb strings.Builder
	sb.Grow(length)

	max := big.NewInt(int64(baseLength))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		sb.WriteByte(base62Chars[n.Int64()])
	}

	return sb.String(), nil
}
