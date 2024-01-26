package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum/hdwallet"
	"github.com/solsticewallet/solstice-core/blockchains/networks"
)

type WalletImp interface {
	accounts.Wallet

	SignHash(accounts.Account, []byte) ([]byte, error)
	SignHashWithPassphrase(accounts.Account, string, []byte) ([]byte, error)

	Unpin(accounts.Account) error

	PrivateKey(accounts.Account) (*ecdsa.PrivateKey, error)
	PrivateKeyBytes(accounts.Account) ([]byte, error)
	PrivateKeyHex(accounts.Account) (string, error)

	PublicKey(accounts.Account) (*ecdsa.PublicKey, error)
	PublicKeyBytes(accounts.Account) ([]byte, error)
	PublicKeyHex(accounts.Account) (string, error)

	Address(accounts.Account) (common.Address, error)
	AddressBytes(accounts.Account) ([]byte, error)
	AddressHex(accounts.Account) (string, error)

	Path(accounts.Account) (string, error)
}

type Wallet interface {
	WalletImp

	Network() string

	AccountBalance(context.Context, *ethclient.Client, accounts.Account, *big.Int) (*big.Int, error)
	AccountBalanceEth(context.Context, *ethclient.Client, accounts.Account, *big.Int) (*big.Float, error)
	PendingAccountBalance(context.Context, *ethclient.Client, accounts.Account) (*big.Int, error)
	PendingAccountBallanceEth(context.Context, *ethclient.Client, accounts.Account) (*big.Float, error)
	CreateTransaction(context.Context, *ethclient.Client, accounts.Account, common.Address, *big.Int, uint64) (*types.Transaction, error)
}

func ConstructWallet(walletType string) Wallet {
	switch walletType {
	case fmt.Sprintf("%T", &hdwallet.Wallet{}):
		return &SoftwareWallet{
			WalletImp:     &hdwallet.Wallet{},
			WalletNetwork: networks.Ethereum,
			WalletType:    walletType,
		}
	}
	return nil
}
