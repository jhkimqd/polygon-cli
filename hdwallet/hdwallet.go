package hdwallet

import (
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

var (
	wordsToBits = map[int]int{12: 128, 15: 160, 18: 192, 21: 224, 24: 256}
)

func NewMnemonic(wordCount int) (string, error) {
	entropy, err := bip39.NewEntropy(wordsToBits[wordCount])
	if err != nil {
		return "", fmt.Errorf("There was an error getting entropy: %s", err.Error())
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("There was an error creating the mnemonic: %s", err.Error())
	}
	return mnemonic, nil
}
