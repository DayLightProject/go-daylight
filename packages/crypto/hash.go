package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

type hashProvider int

const (
	_SHA256 hashProvider = iota
)

func Hash(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		log.Debug(HashingEmpty.Error())
	}
	switch hashProv {
	case _SHA256:
		return hashSHA256(msg), nil
	default:
		return nil, UnknownProviderError
	}
}

func DoubleHash(msg []byte) ([]byte, error) {
	if len(msg) == 0 {
		log.Debug(HashingEmpty.Error())
	}
	switch hashProv {
	case _SHA256:
		return hashDoubleSHA256(msg), nil
	default:
		return nil, UnknownProviderError
	}
}

func HMAC(secret string, message string) ([]byte, error) {
	switch hmacProv {
	case _SHA256:
		key := sha256.Sum256([]byte(secret))
		hash := hmac.New(sha256.New, key[:])
		return hash.Sum([]byte(message)), nil
	default:
		return nil, UnknownProviderError
	}
}

func hashSHA256(msg []byte) []byte {
	hash := sha256.Sum256(msg)
	return hash[:]
}

//In the previous version of this function (api v 1.0) this func worked in another way.
//First, hash has been calculated from input data
//Second, obtained hash has been converted to hex
//Third, hex value has been hashed once more time
//In this variant second step is omited.
func hashDoubleSHA256(msg []byte) []byte {
	firstHash := sha256.Sum256(msg)
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:]
}
