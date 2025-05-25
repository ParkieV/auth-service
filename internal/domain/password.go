package domain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ParkieV/auth-service/internal/config"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidPassword = errors.New("password must be at least 8 chars")
	ErrHashFormat      = errors.New("invalid password hash format")
)

var defaultParams = config.CryptoParams{
	Time:    1,
	Memory:  64 * 1024,
	Threads: 4,
	SaltLen: 16,
	KeyLen:  32,
}

type Password struct {
	hash string
}

func (p Password) Hash() string { return p.hash }

func NewPasswordFromPlain(plain string) (Password, error) {
	if len(plain) < 8 {
		return Password{}, ErrInvalidPassword
	}
	hash, err := argonHash(plain, defaultParams)
	if err != nil {
		return Password{}, err
	}
	return Password{hash: hash}, nil
}

func NewPasswordFromHash(hash string) (Password, error) {
	if _, _, err := parsePHC(hash); err != nil {
		return Password{}, ErrHashFormat
	}
	return Password{hash: hash}, nil
}

func (p Password) Verify(plain string) bool {
	params, phc, err := parsePHC(p.hash)
	if err != nil {
		return false
	}
	calculated := argon2.IDKey([]byte(plain), phc.salt,
		params.Time, params.Memory, params.Threads, params.KeyLen)

	return subtle.ConstantTimeCompare(calculated, phc.hash) == 1
}

func (p Password) NeedsRehash() bool {
	params, _, err := parsePHC(p.hash)
	if err != nil {
		return true
	}
	return params.Memory < defaultParams.Memory ||
		params.Time < defaultParams.Time ||
		params.KeyLen < defaultParams.KeyLen
}

func (p Password) RehashIfNeeded(plain string) (Password, bool, error) {
	if !p.NeedsRehash() {
		return p, false, nil
	}
	np, err := NewPasswordFromPlain(plain)
	return np, true, err
}

type phcParts struct {
	salt []byte
	hash []byte
}

func argonHash(pwd string, prm config.CryptoParams) (string, error) {
	salt := make([]byte, prm.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := argon2.IDKey([]byte(pwd), salt, prm.Time, prm.Memory, prm.Threads, prm.KeyLen)

	b64 := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		prm.Memory, prm.Time, prm.Threads, b64(salt), b64(h)), nil
}

func parsePHC(phc string) (config.CryptoParams, phcParts, error) {
	var v uint32
	var prm config.CryptoParams
	var saltB64, hashB64 string

	_, err := fmt.Sscanf(phc,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		&v, &prm.Memory, &prm.Time, &prm.Threads, &saltB64, &hashB64)
	if err != nil {
		return config.CryptoParams{}, phcParts{}, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return config.CryptoParams{}, phcParts{}, err
	}
	hash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return config.CryptoParams{}, phcParts{}, err
	}
	prm.SaltLen = uint32(len(salt))
	prm.KeyLen = uint32(len(hash))
	return prm, phcParts{salt: salt, hash: hash}, nil
}
