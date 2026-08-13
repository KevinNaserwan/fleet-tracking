package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidPasswordHash = errors.New("invalid argon2id password hash")

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultPasswordParams = PasswordParams{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

const dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=2$ZmxlZXQtZHVtbXktc2FsdA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func VerifyPassword(password, encoded string) (bool, error) {
	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func HashPassword(password string, params PasswordParams) (string, error) {
	if err := validatePasswordParams(params); err != nil {
		return "", err
	}
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func BurnPassword(password string) {
	_, _ = VerifyPassword(password, dummyPasswordHash)
}

func parsePasswordHash(encoded string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	version, err := parseNamedUint(parts[2], "v", 32)
	if err != nil || version != argon2.Version {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	parameters := strings.Split(parts[3], ",")
	if len(parameters) != 3 {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	memory, errMemory := parseNamedUint(parameters[0], "m", 32)
	iterations, errIterations := parseNamedUint(parameters[1], "t", 32)
	parallelism, errParallelism := parseNamedUint(parameters[2], "p", 8)
	if errMemory != nil || errIterations != nil || errParallelism != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	salt, err := decodePHC(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	hash, err := decodePHC(parts[5])
	if err != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	params := PasswordParams{
		Memory:      uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hash)),
	}
	if err := validatePasswordParams(params); err != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	return params, salt, hash, nil
}

func validatePasswordParams(params PasswordParams) error {
	if params.Memory < 8*1024 || params.Memory > 256*1024 ||
		params.Iterations < 1 || params.Iterations > 10 ||
		params.Parallelism < 1 || params.Parallelism > 16 ||
		params.SaltLength < 8 || params.SaltLength > 64 ||
		params.KeyLength < 16 || params.KeyLength > 64 {
		return ErrInvalidPasswordHash
	}
	return nil
}

func parseNamedUint(value, name string, bits int) (uint64, error) {
	prefix := name + "="
	if !strings.HasPrefix(value, prefix) {
		return 0, ErrInvalidPasswordHash
	}
	return strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, bits)
}

func decodePHC(value string) ([]byte, error) {
	decoded, err := base64.RawStdEncoding.Strict().DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.StdEncoding.Strict().DecodeString(value)
}
