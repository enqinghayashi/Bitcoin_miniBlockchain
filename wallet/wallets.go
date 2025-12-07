package wallet

import (
	"bytes"
	"encoding/gob"
	"os"
)

const walletFile = "wallets.dat"

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	ws := &Wallets{Wallets: make(map[string]*Wallet)}
	if _, err := os.Stat(walletFile); err == nil {
		if err := ws.LoadFromFile(); err != nil {
			return nil, err
		}
	}
	return ws, nil
}

func (ws *Wallets) CreateWallet() (string, error) {
	w := NewWallet()
	address := string(w.GetAddress())
	ws.Wallets[address] = w
	return address, ws.SaveToFile()
}

func (ws *Wallets) GetAddresses() []string {
	addresses := make([]string, 0, len(ws.Wallets))
	for addr := range ws.Wallets {
		addresses = append(addresses, addr)
	}
	return addresses
}

func (ws *Wallets) GetWallet(address string) (*Wallet, bool) {
	w, ok := ws.Wallets[address]
	return w, ok
}

func (ws *Wallets) LoadFromFile() error {
	content, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(bytes.NewReader(content))
	var loaded Wallets
	if err := decoder.Decode(&loaded); err != nil {
		return err
	}
	ws.Wallets = loaded.Wallets
	if ws.Wallets == nil {
		ws.Wallets = make(map[string]*Wallet)
	}
	return nil
}

func (ws *Wallets) SaveToFile() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(ws); err != nil {
		return err
	}
	return os.WriteFile(walletFile, buf.Bytes(), 0o600)
}
