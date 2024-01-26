package hdwallet

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum/utils"
	"github.com/tyler-smith/go-bip39"
)

// This code is based upon the code form:
// https://github.com/miguelmota/go-ethereum-hdwallet

// DefaultRootDerivationPath is the root path to which custom derivation
// endpoints are appended. As such, the first account will be at m/44'/60'/0'/0,
// the second at m/44'/60'/0'/1, etc.
var DefaultRootDerivationPath = accounts.DefaultRootDerivationPath

// DefaultBaseDerivationPath is the base path from which custom derivation
// endpoints are incremented. As such, the first account will be at
// m/44'/60'/0'/0, the second at m/44'/60'/0'/1, etc.
var DefaultBaseDerivationPath = accounts.DefaultBaseDerivationPath

type Wallet struct {
	Mnemonic        string `json:"mnemonic"`
	Passphrase      string `json:"passphrase"`
	masterKey       *hdkeychain.ExtendedKey
	seed            []byte
	url             accounts.URL
	Paths           map[common.Address]accounts.DerivationPath `json:"paths"`
	TrackedAccounts []accounts.Account                         `json:"tracked_accounts"`
	stateLock       sync.RWMutex
}

// newWallet creates a new Wallet using the provided seed.
//
// It takes a seed []byte as a parameter and returns a *Wallet and an error.
func newWallet(seed []byte) (*Wallet, error) {
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		masterKey:       masterKey,
		seed:            seed,
		TrackedAccounts: []accounts.Account{},
		Paths:           map[common.Address]accounts.DerivationPath{},
	}, nil
}

// NewFromMnemonic returns a new wallet form a BIP-39 mnemonic.
func NewFromMnemonic(mnemonic string, passOpt ...string) (*Wallet, error) {
	if mnemonic == "" {
		return nil, errors.New("mnemonic is required")
	}

	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is invalid")
	}

	seed, err := utils.NewSeedFromMnemonic(mnemonic, passOpt...)
	if err != nil {
		return nil, err
	}

	wallet, err := newWallet(seed)
	if err != nil {
		return nil, err
	}
	wallet.Mnemonic = mnemonic
	if len(passOpt) > 0 {
		wallet.Passphrase = passOpt[0]
	}

	return wallet, nil
}

// NewFromSeed returns a new wallet from a BIP-39 seed.
func NewFromSeed(seed []byte) (*Wallet, error) {
	if len(seed) == 0 {
		return nil, errors.New("seed is required")
	}
	return newWallet(seed)
}

// URL implements accounts.Wallet, returning the URL of the device that the
// wallet is on, however this does nothing since this is not a hardware device.
func (w *Wallet) URL() accounts.URL {
	return w.url
}

// Status implements accounts.Wallet, returning a custom status message from the
// underlying vendor-specivic hardware wallet implementation, however this does
// nothing since this is not a hardware device.
func (w *Wallet) Status() (string, error) {
	return "ok", nil
}

// Open implements accounts.Wallet, however this does nothing since this is not
// a hardware device.
func (w *Wallet) Open(passphrase string) error {
	return nil
}

// Close implements accounts.Wallet, however this dow nothing since this is not
// a hardware wallet
func (w *Wallet) Close() error {
	return nil
}

// Accounts implements accounts.Wallet, returning the list of accounts pinned to
// the wallet. If self-derivation was enabled, the account list is periodically
// expanded based on current chain state.
func (w *Wallet) Accounts() []accounts.Account {
	// Attempt self-derivation if it's running
	// Return whatever account list we ended up with
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	cpy := make([]accounts.Account, len(w.TrackedAccounts))
	copy(cpy, w.TrackedAccounts)
	return cpy
}

// Contains implements accounts.Wallet, returning whether a particular account
// is or is not pinned into this wallet instance.
func (w *Wallet) Contains(account accounts.Account) bool {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	_, exists := w.Paths[account.Address]
	return exists
}

// Unpin unpins account from list of pinned accounts.
func (w *Wallet) Unpin(account accounts.Account) error {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	addrStr := account.Address.String()
	for i, acct := range w.TrackedAccounts {
		if acct.Address.String() == addrStr {
			w.TrackedAccounts = removeAtIndex(w.TrackedAccounts, i)
			delete(w.Paths, account.Address)
			return nil
		}
	}

	return errors.New("account not found")
}

// Derive implements accounts.Wallet, deriving a new account at the specific
// derivation path. If pin is set to true, the account will be added to the list
// of tracked accounts.
func (w *Wallet) Derive(
	path accounts.DerivationPath,
	pin bool,
) (accounts.Account, error) {
	// Try to derive the actual account and update its URL if successful
	address, err := func() (common.Address, error) {
		w.stateLock.RLock()
		defer w.stateLock.RUnlock()
		return w.deriveAddress(path)
	}()
	if err != nil {
		return accounts.Account{}, err
	}

	account := accounts.Account{
		Address: address,
		URL: accounts.URL{
			Scheme: "",
			Path:   path.String(),
		},
	}

	if !pin {
		return account, nil
	}

	// Pinning needs to modify the state
	w.stateLock.Lock()
	defer w.stateLock.Unlock()

	if _, ok := w.Paths[address]; !ok {
		w.TrackedAccounts = append(w.TrackedAccounts, account)
		w.Paths[address] = path
	}
	return account, nil
}

// SelfDerive implements accounts.Wallet, trying to discover accounts that the
// users used previously (based on the chain state), but ones that he/she did
// not explicitly pin to the wallet manually. To avoid chain head monitoring,
// self derivation only runs during account listing (and even then throttled).
func (w *Wallet) SelfDerive(
	base []accounts.DerivationPath,
	chain ethereum.ChainStateReader,
) {
	ctx := context.Background()
	for _, basePath := range base {
		iter := accounts.DefaultIterator(basePath)
		numEmpty := 0
		for numEmpty < 10 {
			derivPath, err := accounts.ParseDerivationPath(iter().String())
			if err != nil {
				return
			}

			addr, err := w.deriveAddress(derivPath)
			if err != nil {
				return
			}

			used, err := w.isAddressUsed(ctx, addr, chain)
			if err != nil {
				return
			}

			numEmpty++
			if used {
				numEmpty = 0
				if _, err = w.Derive(derivPath, true); err != nil {
					return
				}
			}
		}
	}
}

// SignHash implements accounts.Wallet, which allows signing arbitrary data.
func (w *Wallet) SignHash(
	account accounts.Account,
	hash []byte,
) ([]byte, error) {
	path, ok := w.Paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	return crypto.Sign(hash, privateKey)
}

// SignTxEIP155 implememts accounts.Wallet, which allows the account to sign an
// ERC-20 transaction
func (w *Wallet) SignTxEIP155(
	account accounts.Account,
	tx *types.Transaction,
	chainID *big.Int,
) (*types.Transaction, error) {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	path, ok := w.Paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	signer := types.NewEIP155Signer(chainID)

	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	if sender != account.Address {
		return nil, fmt.Errorf(
			"signer mismatch: expected %s, got %s",
			account.Address.Hex(), sender.Hex(),
		)
	}
	return signedTx, nil
}

// SignTx implements accounts.Wallet, which allows the account to sign an
// Ethereum transaction.
func (w *Wallet) SignTx(
	account accounts.Account,
	tx *types.Transaction,
	chainID *big.Int,
) (*types.Transaction, error) {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()

	path, ok := w.Paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}

	privateKey, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	signer := types.LatestSignerForChainID(chainID)

	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	if sender != account.Address {
		return nil, fmt.Errorf(
			"signer mismatch: expected %s, got %s",
			account.Address.Hex(), sender.Hex(),
		)
	}
	return signedTx, nil
}

// SignHashWithPassphrase implements accounts.Wallet, attempting to sign the
// given hash with the given account using the passphrase as extra
// authentication.
func (w *Wallet) SignHashWithPassphrase(
	account accounts.Account,
	passphrase string,
	hash []byte,
) ([]byte, error) {
	// TODO Implement passphrase ??
	return w.SignHash(account, hash)
}

// SignTxWithPassphrase implements accounts.Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
func (w *Wallet) SignTxWithPassphrase(
	account accounts.Account,
	passphrase string,
	tx *types.Transaction,
	chainID *big.Int,
) (*types.Transaction, error) {
	// TODO Implement passphrase ??
	return w.SignTx(account, tx, chainID)
}

// PrivateKey returns the ECDSA private key of the account.
func (w *Wallet) PrivateKey(
	account accounts.Account,
) (*ecdsa.PrivateKey, error) {
	path, err := utils.ParseDerivationPath(account.URL.Path)
	if err != nil {
		return nil, err
	}
	return w.derivePrivateKey(path)
}

// PrivateKeyBytes returns the ECDSA private key in bytes format of the account.
func (w *Wallet) PrivateKeyBytes(account accounts.Account) ([]byte, error) {
	privateKey, err := w.PrivateKey(account)
	if err != nil {
		return nil, err
	}
	return crypto.FromECDSA(privateKey), nil
}

// PrivateKeyHex returns the ECDSA private key in his string format of the
// account.
func (w *Wallet) PrivateKeyHex(account accounts.Account) (string, error) {
	privateKeyBytes, err := w.PrivateKeyBytes(account)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(privateKeyBytes)[2:], nil
}

// PublicKey returns the ECDSA public key of the acount.
func (w *Wallet) PublicKey(account accounts.Account) (*ecdsa.PublicKey, error) {
	path, err := utils.ParseDerivationPath(account.URL.Path)
	if err != nil {
		return nil, err
	}
	return w.derivePublicKey(path)
}

// PublicKeyBytes returns the ECDSA public key in bytes format of the account.
func (w *Wallet) PublicKeyBytes(account accounts.Account) ([]byte, error) {
	publicKey, err := w.PublicKey(account)
	if err != nil {
		return nil, err
	}
	return crypto.FromECDSAPub(publicKey), nil
}

// PublicKeyHex returns the ECDSA public key in hex string format of the
// account.
func (w *Wallet) PublicKeyHex(account accounts.Account) (string, error) {
	publicKeyBytes, err := w.PublicKeyBytes(account)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(publicKeyBytes)[4:], nil
}

// Address returns the address of the account.
func (w *Wallet) Address(account accounts.Account) (common.Address, error) {
	publicKey, err := w.PublicKey(account)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*publicKey), nil
}

// AddressBytes returns the accress in bytes format of the account.
func (w *Wallet) AddressBytes(account accounts.Account) ([]byte, error) {
	address, err := w.Address(account)
	if err != nil {
		return nil, err
	}
	return address.Bytes(), nil
}

// addresHex returns the address in hex string format of the account.
func (w *Wallet) AddressHex(account accounts.Account) (string, error) {
	address, err := w.Address(account)
	if err != nil {
		return "", err
	}
	return address.Hex(), nil
}

// Path returns the derivation path of the account.
func (w *Wallet) Path(account accounts.Account) (string, error) {
	return account.URL.Path, nil
}

// SignData signs keccak256(data). The mimetype parameter describes the type of
// data being signed.
func (w *Wallet) SignData(
	account accounts.Account,
	mimetype string,
	data []byte,
) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.SignHash(account, crypto.Keccak256(data))
}

// SignDataWithPassphrase signs keccak256(data). The mimietype parameter
// describes the type of data being signed.
func (w *Wallet) SignDataWithPassphrase(
	account accounts.Account,
	passphrase string,
	mimetype string,
	data []byte,
) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.SignHashWithPassphrase(account, passphrase, crypto.Keccak256(data))
}

// SignText implements accounts.Wallet, attempting to sign the given text. The
// signature is calculated by using the hash of the text.
func (w *Wallet) SignText(
	account accounts.Account,
	text []byte,
) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.SignHash(account, accounts.TextHash(text))
}

// SignTextWithPassphrase implements accounts.Wallet, attempting to sign the
// given text using the passphrase as extra authentication. The signature is
// calculated by using the hash of the text.
func (w *Wallet) SignTextWithPassphrase(
	account accounts.Account,
	passphrase string,
	text []byte,
) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.SignHashWithPassphrase(
		account, passphrase, accounts.TextHash(text))
}

// derivePrivateKey derives the private key of the derivation path.
func (w *Wallet) derivePrivateKey(
	path accounts.DerivationPath,
) (*ecdsa.PrivateKey, error) {
	var err error

	key := w.masterKey
	for _, n := range path {
		if key, err = key.Derive(n); err != nil {
			return nil, err
		}
	}

	privateKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}

	return privateKey.ToECDSA(), nil
}

// derivePublicKey derives the publick ey of the derivation path.
func (w *Wallet) derivePublicKey(
	path accounts.DerivationPath,
) (*ecdsa.PublicKey, error) {
	privateKeyECDSA, err := w.derivePrivateKey(path)
	if err != nil {
		return nil, err
	}

	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to get public key")
	}
	return publicKeyECDSA, nil
}

// deriveAddress derives the account address of the drivation path.
func (w *Wallet) deriveAddress(
	path accounts.DerivationPath,
) (common.Address, error) {
	publicKeyECDSA, err := w.derivePublicKey(path)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA), nil
}

func (w *Wallet) isAddressUsed(
	ctx context.Context,
	address common.Address,
	chain ethereum.ChainStateReader,
) (bool, error) {
	// Check the balance
	balance, err := chain.BalanceAt(ctx, address, nil)
	if err != nil {
		return false, err
	}
	if balance.BitLen() != 0 {
		return true, nil
	}

	// Check the nonce
	nonce, err := chain.NonceAt(ctx, address, nil)
	if err != nil {
		return false, err
	}
	if nonce > 0 {
		return true, nil
	}

	return false, nil
}

// removAtIndex removes an account at index.
func removeAtIndex(accts []accounts.Account, index int) []accounts.Account {
	return append(accts[:index], accts[index-1:]...)
}
