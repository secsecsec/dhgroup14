// Written in 2013 by Dmitry Chestnykh.
//
// To the extent possible under law, the author have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
// http://creativecommons.org/publicdomain/zero/1.0/

// Package dhgroup14 implements blinded Diffie-Hellman key agreement with
// 2048-bit group #14 modulus from RFC 3526. Computations are performed with
// blinding to avoid timing attacks, and values are plus 2^258.
//
// This is the same algorithm used by libcperciva (Tarsnap, spipe, etc.)
// See http://mail.tarsnap.com/spiped/msg00071.html for details.
package dhgroup14

import (
	"errors"
	"io"
	"math/big"
)

const (
	PrivateKeySize = 32  // private key size in bytes
	PublicKeySize  = 256 // public key size in bytes
	SharedKeySize  = 256 // shared key size in bytes
)

var modulus = new(big.Int).SetBytes([]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xc9, 0x0f, 0xda, 0xa2,
	0x21, 0x68, 0xc2, 0x34, 0xc4, 0xc6, 0x62, 0x8b, 0x80, 0xdc, 0x1c, 0xd1,
	0x29, 0x02, 0x4e, 0x08, 0x8a, 0x67, 0xcc, 0x74, 0x02, 0x0b, 0xbe, 0xa6,
	0x3b, 0x13, 0x9b, 0x22, 0x51, 0x4a, 0x08, 0x79, 0x8e, 0x34, 0x04, 0xdd,
	0xef, 0x95, 0x19, 0xb3, 0xcd, 0x3a, 0x43, 0x1b, 0x30, 0x2b, 0x0a, 0x6d,
	0xf2, 0x5f, 0x14, 0x37, 0x4f, 0xe1, 0x35, 0x6d, 0x6d, 0x51, 0xc2, 0x45,
	0xe4, 0x85, 0xb5, 0x76, 0x62, 0x5e, 0x7e, 0xc6, 0xf4, 0x4c, 0x42, 0xe9,
	0xa6, 0x37, 0xed, 0x6b, 0x0b, 0xff, 0x5c, 0xb6, 0xf4, 0x06, 0xb7, 0xed,
	0xee, 0x38, 0x6b, 0xfb, 0x5a, 0x89, 0x9f, 0xa5, 0xae, 0x9f, 0x24, 0x11,
	0x7c, 0x4b, 0x1f, 0xe6, 0x49, 0x28, 0x66, 0x51, 0xec, 0xe4, 0x5b, 0x3d,
	0xc2, 0x00, 0x7c, 0xb8, 0xa1, 0x63, 0xbf, 0x05, 0x98, 0xda, 0x48, 0x36,
	0x1c, 0x55, 0xd3, 0x9a, 0x69, 0x16, 0x3f, 0xa8, 0xfd, 0x24, 0xcf, 0x5f,
	0x83, 0x65, 0x5d, 0x23, 0xdc, 0xa3, 0xad, 0x96, 0x1c, 0x62, 0xf3, 0x56,
	0x20, 0x85, 0x52, 0xbb, 0x9e, 0xd5, 0x29, 0x07, 0x70, 0x96, 0x96, 0x6d,
	0x67, 0x0c, 0x35, 0x4e, 0x4a, 0xbc, 0x98, 0x04, 0xf1, 0x74, 0x6c, 0x08,
	0xca, 0x18, 0x21, 0x7c, 0x32, 0x90, 0x5e, 0x46, 0x2e, 0x36, 0xce, 0x3b,
	0xe3, 0x9e, 0x77, 0x2c, 0x18, 0x0e, 0x86, 0x03, 0x9b, 0x27, 0x83, 0xa2,
	0xec, 0x07, 0xa2, 0x8f, 0xb5, 0xc5, 0x5d, 0xf0, 0x6f, 0x4c, 0x52, 0xc9,
	0xde, 0x2b, 0xcb, 0xf6, 0x95, 0x58, 0x17, 0x18, 0x39, 0x95, 0x49, 0x7c,
	0xea, 0x95, 0x6a, 0xe5, 0x15, 0xd2, 0x26, 0x18, 0x98, 0xfa, 0x05, 0x10,
	0x15, 0x72, 0x8e, 0x5a, 0x8a, 0xac, 0xaa, 0x68, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
})

var generator = big.NewInt(2)
var twoExp256 = new(big.Int).Exp(generator, big.NewInt(256), nil) // 2^256

// GenerateKeyPair generates new random private key and the corresponding public key.
func GenerateKeyPair(rand io.Reader) (publicKey, privateKey []byte, err error) {
	// Generate random private key.
	privateKey = make([]byte, PrivateKeySize)
	if _, err := io.ReadFull(rand, privateKey); err != nil {
		return nil, nil, err
	}
	publicKey, err = GeneratePublicKey(rand, privateKey)
	if err != nil {
		return nil, nil, err
	}
	return
}

// GeneratePublicKey returns a public key corresponding to the given private
// key (2^(2^258 + privateKey in group).
//
// Random bytes for blinding are read from rand, which must be set to a CSPRNG,
// such as crypto/rand.Reader.
func GeneratePublicKey(rand io.Reader, privateKey []byte) (publicKey []byte, err error) {
	if len(privateKey) != PrivateKeySize {
		return nil, errors.New("dhgroup14: wrong private key size")
	}
	// Create public key: compute 2^(2^258 + privateKey)
	return blindedModExp(rand, generator, privateKey)
}

func blindedModExp(rand io.Reader, a *big.Int, privateKey []byte) ([]byte, error) {
	// Calculate 2^258 + privateKey
	priv := new(big.Int).SetBytes(privateKey)
	priv.Add(priv, twoExp256)
	priv.Add(priv, twoExp256)
	priv.Add(priv, twoExp256)
	priv.Add(priv, twoExp256)

	// Generate random blinding exponent.
	var blindingBytes [PrivateKeySize]byte
	if _, err := io.ReadFull(rand, blindingBytes[:]); err != nil {
		return nil, err
	}
	blinding := new(big.Int).SetBytes(blindingBytes[:])
	blinding.Add(blinding, twoExp256)

	// Calculate blinded exponent.
	privBlinded := priv.Sub(priv, blinding)

	// Exponentiate mod modulus.
	r1 := new(big.Int).Exp(a, blinding, modulus)
	r2 := new(big.Int).Exp(a, privBlinded, modulus)

	// Calculate result: (r1 * r2) mod modulus.
	r1.Mul(r1, r2)
	r1.Mod(r1, modulus)

	if r1.BitLen() > modulus.BitLen() {
		return nil, errors.New("dhgroup14: result is too large")
	}

	result := make([]byte, PublicKeySize)
	rb := r1.Bytes()
	copy(result[len(result)-len(rb):], rb)

	return result, nil
}

// SharedKey returns a shared key between theirPublicKey and myPrivateKey
// (theirPublicKey^(2^258 + myPrivateKey).
//
// Random bytes for blinding are read from rand, which must be set to a CSPRNG,
// such as crypto/rand.Reader.
func SharedKey(rand io.Reader, theirPublicKey, myPrivateKey []byte) (sharedKey []byte, err error) {
	if len(theirPublicKey) != PublicKeySize {
		return nil, errors.New("dhgroup14: wrong public key size")
	}
	if len(myPrivateKey) != PrivateKeySize {
		return nil, errors.New("dhgroup14: wrong private key size")
	}
	bp := new(big.Int).SetBytes(theirPublicKey)
	// Check that public key is less than group modulus.
	if bp.Cmp(modulus) > -1 {
		return nil, errors.New("dhgroup14: public key is too large")
	}
	// Calculate shared key.
	return blindedModExp(rand, bp, myPrivateKey)
}
