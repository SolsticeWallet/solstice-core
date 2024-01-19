package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum"
	"github.com/solsticewallet/solstice-core/blockchains/ethereum/utils"
)

const mnemonic = "slice vote elbow curtain write side give rural entire under pause common"
const (
	derivpath0 = "m/44'/60'/0'/0/0"
	derivpath1 = "m/44'/60'/0'/0/1"
	derivpath2 = "m/44'/60'/0'/0/2"
	derivpath3 = "m/44'/60'/0'/0/3"
	derivpath4 = "m/44'/60'/0'/0/4"
	derivpath5 = "m/44'/60'/0'/0/5"
	derivpath6 = "m/44'/60'/0'/0/6"
	derivpath7 = "m/44'/60'/0'/0/7"
	derivpath8 = "m/44'/60'/0'/0/8"
	derivpath9 = "m/44'/60'/0'/0/9"
)

const (
	addr0 = "0x2aCeC377a19EB0A62557bac2C3F0D7B7cd88FB21"
	addr1 = "0x9095AeBF0357E66C1aA34901Cd9bAD5beF8DffB1"
	addr2 = "0x8D2a8D219cfCD6E476584bc4FCc2b59e390De56E"
	addr3 = "0xfB128757Fa79b80B91B5F8014b8ca67E42155BFa"
	addr4 = "0x00139306c45174EB35f130c04345feb69B1D4657"
	addr5 = "0x3f00696c105f83de4a6DF5e1E6E188D6a6F9A287"
	addr6 = "0xe40F4D0Db3348a6F55462c48fAFE16eC617bD2D7"
	addr7 = "0x3243a71d3997247f33885369275461A70a14CC56"
	addr8 = "0x4bDBD9195d9cbeb1057C795DcbD36133775C2C91"
	addr9 = "0x1aD440DCC1df84Fd42B21EFbFDe62043774e6e18"
)

func main() {
	client, err := ethclient.Dial("http://127.0.0.1:7545")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	wallet, err := ethereum.NewSoftwareWalletFromMnemonic(mnemonic)
	if err != nil {
		panic(err)
	}

	path, err := accounts.ParseDerivationPath(derivpath0)
	if err != nil {
		panic(err)
	}

	wallet.SelfDerive([]accounts.DerivationPath{path}, client)
	accounts := wallet.Accounts()

	fmt.Println("Tracked addresses: ")
	for _, account := range accounts {
		fmt.Println(account.Address.Hex())
	}
	fmt.Println("")

	account, err := wallet.Derive(path, true)
	if err != nil {
		panic(err)
	}

	balance, err := wallet.AccountBalanceEth(
		context.Background(),
		client,
		account,
		nil,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(balance)

	tx, err := wallet.CreateTransaction(
		context.Background(),
		client,
		account,
		common.HexToAddress(addr1),
		utils.Eth2Wei(big.NewFloat(1.0)),
		uint64(21000),
	)
	if err != nil {
		panic(err)
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	tx, err = wallet.SignTx(account, tx, chainId)
	if err != nil {
		panic(err)
	}

	err = client.SendTransaction(context.Background(), tx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Transaction send")
}
