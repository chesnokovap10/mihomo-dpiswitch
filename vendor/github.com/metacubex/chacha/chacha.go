package chacha

import (
	"crypto/cipher"

	"github.com/metacubex/chacha/chacha"
	"github.com/metacubex/chacha/chachapoly1305"
	"github.com/metacubex/chacha/poly1305"
)

const (
	// NonceSize is the size of the ChaCha20 nonce in bytes.
	NonceSize = chacha.NonceSize

	// INonceSize is the size of the IETF-ChaCha20 nonce in bytes.
	INonceSize = chacha.INonceSize

	// XNonceSize is the size of the XChaCha20 nonce in bytes.
	XNonceSize = chacha.XNonceSize

	// KeySize is the size of the key in bytes.
	KeySize = chacha.KeySize

	// Overhead is the size of the Poly1305 authentication tag, and the
	// difference between a ciphertext length and its plaintext.
	Overhead = poly1305.TagSize
)

// NewChaCha20 returns a cipher.Stream implementing the ChaCha20 or XChaCha20
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha20:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha20(nonce, key []byte) (cipher.Stream, error) {
	return chacha.NewCipher(nonce, key, 20)
}

// NewChaCha20IgnoreCounterOverflow returns a cipher.Stream implementing the ChaCha20 or XChaCha20
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha20:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha20IgnoreCounterOverflow(nonce, key []byte) (cipher.Stream, error) {
	c, err := chacha.NewCipher(nonce, key, 20)
	if err != nil {
		return nil, err
	}
	c.SetIgnoreCounterOverflow(true)
	return c, nil
}

// NewChaCha12 returns a cipher.Stream implementing the ChaCha12 or XChaCha12
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha12:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha12(nonce, key []byte) (cipher.Stream, error) {
	return chacha.NewCipher(nonce, key, 12)
}

// NewChaCha12IgnoreCounterOverflow returns a cipher.Stream implementing the ChaCha12 or XChaCha12
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha20:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha12IgnoreCounterOverflow(nonce, key []byte) (cipher.Stream, error) {
	c, err := chacha.NewCipher(nonce, key, 12)
	if err != nil {
		return nil, err
	}
	c.SetIgnoreCounterOverflow(true)
	return c, nil
}

// NewChaCha8 returns a cipher.Stream implementing the ChaCha8 or XChaCha8
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha8:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha8(nonce, key []byte) (cipher.Stream, error) {
	return chacha.NewCipher(nonce, key, 8)
}

// NewChaCha8IgnoreCounterOverflow returns a cipher.Stream implementing the ChaCha8 or XChaCha8
// stream cipher. The nonce must be unique for one key for all time.
// The length of the nonce determinds the version of ChaCha8:
// - NonceSize:  ChaCha20/r with a 64 bit nonce and a 2^64 * 64 byte period.
// - INonceSize: ChaCha20/r as defined in RFC 7539 and a 2^32 * 64 byte period.
// - XNonceSize: XChaCha20/r with a 192 bit nonce and a 2^64 * 64 byte period.
// If the nonce is neither 64, 96 nor 192 bits long, a non-nil error is returned.
func NewChaCha8IgnoreCounterOverflow(nonce, key []byte) (cipher.Stream, error) {
	c, err := chacha.NewCipher(nonce, key, 8)
	if err != nil {
		return nil, err
	}
	c.SetIgnoreCounterOverflow(true)
	return c, nil
}

// NewChaCha20Poly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha20Poly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, NonceSize, 20)
}

// NewChaCha12Poly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha12Poly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, NonceSize, 12)
}

// NewChaCha8Poly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha8Poly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, NonceSize, 8)
}

// NewChaCha20IETFPoly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha20IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, INonceSize, 20)
}

// NewChaCha12IETFPoly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha12IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, INonceSize, 12)
}

// NewChaCha8IETFPoly1305 returns a cipher.AEAD implementing the
// ChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewChaCha8IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, INonceSize, 8)
}

// NewXChaCha20IETFPoly1305 returns a cipher.AEAD implementing the
// XChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewXChaCha20IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, XNonceSize, 20)
}

// NewXChaCha12IETFPoly1305 returns a cipher.AEAD implementing the
// XChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewXChaCha12IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, XNonceSize, 12)
}

// NewXChaCha8IETFPoly1305 returns a cipher.AEAD implementing the
// XChaCha20Poly1305 construction specified in RFC 7539 with a
// 128 bit auth. tag.
func NewXChaCha8IETFPoly1305(key []byte) (cipher.AEAD, error) {
	return chachapoly1305.NewCipher(key, XNonceSize, 8)
}
