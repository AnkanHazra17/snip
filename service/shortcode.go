package main

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateCode() (string, error) {
	code := make([]byte, 6)

	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}

		code[i] = alphabet[n.Int64()]
	}

	return string(code), nil
}
