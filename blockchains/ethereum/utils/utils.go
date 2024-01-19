package utils

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/tyler-smith/go-bip39"
)

// ParseDerivationPath parses the derivation path in string format into
// []uint32.
func ParseDerivationPath(path string) (accounts.DerivationPath, error) {
	return accounts.ParseDerivationPath(path)
}

// MustParseDerivationPath parses the derivation path in string format into
// []uint32 but will panic if it can't parse it.
func MustParseDerivationPath(path string) accounts.DerivationPath {
	parsed, err := accounts.ParseDerivationPath(path)
	if err != nil {
		panic(err)
	}
	return parsed
}

// NewMnemonic returns a randomly generated BIP-39 mnemonic using 128-256 bits
// of entropy.
// bitSize has to be a multiple 32 and be within the inclusive range of
// {128, 256}
func NewMnemonic(bits int) (string, error) {
	entropy, err := bip39.NewEntropy(bits)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// NewMnemonicFromEntropy returns a BIP-39 menomonic from entropy.
func NewMnemonicFromEntropy(entropy []byte) (string, error) {
	return bip39.NewMnemonic(entropy)
}

// NewEntropy returns a randomly generated entropy.
func NewEntropy(bits int) ([]byte, error) {
	return bip39.NewEntropy(bits)
}

// NewSeed returns a randomly generated BIP-39 seed.
func NewSeed() ([]byte, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	return b, err
}

// NewSeedFromMnemonic returns a BIP-39 seed based on a BIP-39 mnemonic.
func NewSeedFromMnemonic(mnemonic string, passOpt ...string) ([]byte, error) {
	if mnemonic == "" {
		return nil, errors.New("mnemonic is required")
	}

	password := ""
	if len(passOpt) > 0 {
		password = passOpt[0]
	}
	return bip39.NewSeedWithErrorChecking(mnemonic, password)
}

func Wei2Eth(wei *big.Int) *big.Float {
	if wei == nil {
		return nil
	}
	fwei := new(big.Float)
	fwei.SetString(wei.String())
	return new(big.Float).Quo(fwei, big.NewFloat(math.Pow10(18)))
}

func Eth2Wei(eth *big.Float) *big.Int {
	if eth == nil {
		return nil
	}
	fwei := new(big.Float).Quo(eth, big.NewFloat(math.Pow10(-18)))
	result := new(big.Int)
	fwei.Int(result)
	return result
}
