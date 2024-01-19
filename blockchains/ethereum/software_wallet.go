package ethereum

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum/hdwallet"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum/utils"
)

type SoftwareWallet struct {
	WalletImp
}

func NewSoftwareWalletFromMnemonic(
	mnemonic string,
	passOpt ...string,
) (Wallet, error) {
	imp, err := hdwallet.NewFromMnemonic(mnemonic, passOpt...)
	if err != nil {
		return nil, err
	}

	return &SoftwareWallet{
		WalletImp: imp,
	}, nil
}

func NewSoftwareWalletFromSeed(seed []byte) (Wallet, error) {
	imp, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		return nil, err
	}

	return &SoftwareWallet{
		WalletImp: imp,
	}, nil
}

func (w *SoftwareWallet) AccountBalance(
	ctx context.Context,
	client *ethclient.Client,
	account accounts.Account,
	blockNumber *big.Int,
) (*big.Int, error) {
	return client.BalanceAt(ctx, account.Address, blockNumber)
}

func (w *SoftwareWallet) AccountBalanceEth(
	ctx context.Context,
	client *ethclient.Client,
	account accounts.Account,
	blockNumber *big.Int,
) (*big.Float, error) {
	wei, err := w.AccountBalance(ctx, client, account, blockNumber)
	if err != nil {
		return nil, err
	}
	return utils.Wei2Eth(wei), nil
}

func (w *SoftwareWallet) PendingAccountBalance(
	ctx context.Context,
	client *ethclient.Client,
	account accounts.Account,
) (*big.Int, error) {
	return client.PendingBalanceAt(ctx, account.Address)
}

func (w *SoftwareWallet) PendingAccountBallanceEth(
	ctx context.Context,
	client *ethclient.Client,
	account accounts.Account,
) (*big.Float, error) {
	wei, err := w.PendingAccountBalance(ctx, client, account)
	if err != nil {
		return nil, err
	}
	return utils.Wei2Eth(wei), nil
}

func (w *SoftwareWallet) CreateTransaction(
	ctx context.Context,
	client *ethclient.Client,
	account accounts.Account,
	toAddress common.Address,
	value *big.Int,
	gassLimit uint64,
) (*types.Transaction, error) {
	nonce, err := client.NonceAt(ctx, account.Address, nil)
	if err != nil {
		return nil, err
	}

	gassPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(
		nonce, toAddress, value, gassLimit, gassPrice, []byte{})
	return tx, nil
}
