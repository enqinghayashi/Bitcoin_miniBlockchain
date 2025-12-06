package wallet

import (
	"bytes"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Base58Encode(input []byte) []byte {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var result []byte
	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	// Add '1' for each leading 0 byte.
	for _, b := range input {
		if b == 0x00 {
			result = append(result, b58Alphabet[0])
		} else {
			break
		}
	}

	ReverseBytes(result)
	return result
}

func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, b := range input {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		if charIndex < 0 {
			return nil
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()

	// Restore leading zero bytes.
	leadingOnes := 0
	for _, b := range input {
		if b == b58Alphabet[0] {
			leadingOnes++
		} else {
			break
		}
	}

	return append(bytes.Repeat([]byte{0x00}, leadingOnes), decoded...)
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
