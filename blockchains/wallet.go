package blockchains

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/solsticewallet/solstice-core/blockchains/ethereum"
	"github.com/solsticewallet/solstice-core/blockchains/networks"
	"github.com/solsticewallet/solstice-core/crypt"
)

var pwdcheck = []byte("aine-midsummer@proton.me")

type Wallet interface {
	Network() string
	Save(path string, password string) error
}

type walletImp interface {
	Network() string
}

type wallet struct {
	imp       walletImp
	encrypted bool
	pwdCheck  []byte
}

type WalletOpts struct {
	Network    string
	Mnemonic   string
	Passphrase string
	Encrypted  bool
}

func NewWallet(opts WalletOpts) (wlt Wallet, err error) {
	var imp walletImp

	switch opts.Network {
	case networks.Ethereum:
		imp, err = ethereum.NewSoftwareWalletFromMnemonic(
			opts.Mnemonic,
			opts.Passphrase,
		)
	default:
		imp, err = nil, errors.New("unsupported network")
	}

	if err != nil {
		return nil, err
	}

	wlt = &wallet{
		imp:       imp,
		encrypted: opts.Encrypted,
	}
	return
}

func LoadWallet(path string, password string) (wlt Wallet, err error) {
	data, pwdCheck, err := readWalletData(path, password)
	if err != nil {
		return
	}

	tmpMap := make(map[string]interface{})
	err = json.Unmarshal(data, &tmpMap)
	if err != nil {
		return
	}

	network := tmpMap["network"].(string)
	walletType := tmpMap["wallet_type"].(string)

	wallet := constructWallet(network, walletType)
	err = json.Unmarshal(data, &wallet.imp)
	if err != nil {
		return
	}

	if pwdCheck != nil {
		wallet.encrypted = true
		wallet.pwdCheck = pwdCheck
	}
	wlt = wallet

	return
}

func (w wallet) Network() string {
	return w.imp.Network()
}

func (w *wallet) Save(path string, password string) error {
	data, err := json.Marshal(w.imp)
	if err != nil {
		return err
	}

	keySize := crypt.AESNone
	if w.encrypted {
		keySize = crypt.AES256
	}

	newPwdCheck, err := writeWalletData(path, data, keySize, password, w.pwdCheck)
	if err != nil {
		return err
	}
	if newPwdCheck != nil {
		w.pwdCheck = newPwdCheck
	}
	return nil
}

func writeWalletData(path string, data []byte, keySize crypt.AESKeySize, password string, pwdCheck []byte) (newPwdCheck []byte, err error) {
	var bts []byte
	if keySize != crypt.AESNone {
		pwdHash := keySize.Hash([]byte(password))

		if len(pwdCheck) > 0 {
			var checkRes []byte
			checkRes, err = crypt.AESDecrypt(pwdHash, pwdCheck)
			if err != nil {
				return
			}

			if !bytes.Equal(checkRes, pwdcheck) {
				err = errors.New("invalid password")
				return
			}

		}

		newPwdCheck, err = crypt.AESEncrypt(pwdHash, pwdcheck)
		if err != nil {
			return
		}

		bts, err = crypt.AESEncrypt(pwdHash, data)
		if err != nil {
			newPwdCheck = nil
			return
		}

	} else {
		bts = make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(bts, data)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		newPwdCheck = nil
		return
	}
	defer f.Close()

	_, err = f.Write([]byte{byte(keySize)}) // Key size
	if err != nil {
		newPwdCheck = nil
		return
	}

	b8 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b8, uint64(len(bts)))
	_, err = f.Write(b8) // Data size
	if err != nil {
		newPwdCheck = nil
		return
	}

	_, err = f.Write(bts) // Data
	if err != nil {
		newPwdCheck = nil
		return
	}
	return
}

func readWalletData(path string, password string) (data []byte, pwdCheck []byte, err error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	b1 := make([]byte, 1)
	_, err = f.Read(b1) // Key size
	if err != nil {
		return
	}
	keySize := crypt.AESKeySize(b1[0])

	b8 := make([]byte, 8)
	_, err = f.Read(b8) // Data size
	if err != nil {
		return
	}
	dataSize := binary.LittleEndian.Uint64(b8)

	walletData := make([]byte, int(dataSize))
	_, err = f.Read(walletData)
	if err != nil && err != io.EOF {
		return
	}

	switch keySize {
	case crypt.AESNone:
		data = make([]byte, base64.StdEncoding.DecodedLen(len(walletData)))
		_, err = base64.StdEncoding.Decode(data, walletData)
		if err != nil {
			data = nil
			return
		}
	case crypt.AES256:
		pwdHash := keySize.Hash([]byte(password))

		data, err = crypt.AESDecrypt(pwdHash, walletData)
		if err != nil {
			return
		}

		pwdCheck, err = crypt.AESEncrypt(pwdHash, pwdcheck)
		if err != nil {
			data = nil
			return
		}
	}
	return
}

func constructWallet(network string, walletType string) *wallet {
	var imp walletImp

	switch network {
	case networks.Ethereum:
		imp = ethereum.ConstructWallet(walletType)
	}

	return &wallet{
		imp: imp,
	}
}
