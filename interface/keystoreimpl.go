package _interface

import (
	"github.com/elastos/Elastos.ELA.SPV/spvwallet"
)

type KeystoreImpl struct {
	keystore spvwallet.Keystore
}

// This method will open or create a keystore with the given password
func (impl *KeystoreImpl) Open(password string) (Keystore, error) {
	var err error
	// Try to open keystore first
	impl.keystore, err = spvwallet.OpenKeystore([]byte(password))
	if err == nil {
		return impl, nil
	}

	// Try to create a keystore
	impl.keystore, err = spvwallet.CreateKeystore([]byte(password))
	if err != nil {
		return nil, err
	}

	return impl, nil
}

func (impl *KeystoreImpl) ChangePassword(old, new string) error {
	return impl.keystore.ChangePassword([]byte(old), []byte(new))
}

func (impl *KeystoreImpl) MainAccount() Account {
	return impl.keystore.MainAccount()
}

func (impl *KeystoreImpl) NewAccount() Account {
	return impl.keystore.NewAccount()
}

func (impl *KeystoreImpl) GetAccounts() []Account {
	var accounts []Account
	for _, account := range impl.keystore.GetAccounts() {
		accounts = append(accounts, account)
	}
	return accounts
}

func (impl *KeystoreImpl) Json() (string, error) {
	return impl.keystore.Json()
}

func (impl *KeystoreImpl) FromJson(str string, password string) error {
	impl.keystore = new(spvwallet.KeystoreImpl)
	return impl.keystore.FromJson(str, password)
}
