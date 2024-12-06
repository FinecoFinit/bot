package wg

import (
	"bot/concierge"
	"bytes"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type WireGuard struct {
	Config concierge.Config
}

type WireGuardKeys struct {
	Private   string
	PreShared string
	Public    string
}

func (w WireGuard) WgStartSession(user *concierge.User) error {
	preK := path.Join(w.Config.WgPreKeysDir, strconv.FormatInt(user.ID, 10))
	if _, err := os.Stat(preK); os.IsNotExist(err) {
		err = os.WriteFile(preK, []byte(user.PeerPre), 0644)
		if err != nil {
			return err
		}
	}

	wgCommand := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.PeerPub,
		"preshared-key", preK,
		"allowed-ips", w.Config.WgSubNet+strconv.Itoa(user.IP)+"/32")
	err := wgCommand.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to start session: %w", err)
	}

	err = os.Remove(preK)
	if err != nil {
		return fmt.Errorf("wgmng: failed to delete pre-shared key from directory: %w", err)
	}

	return nil
}

func (w WireGuard) WgStopSession(user *concierge.User) error {
	wgCommand := exec.Command(
		"wg",
		"set",
		"wg0-server",
		"peer", user.PeerPub,
		"remove")
	err := wgCommand.Run()
	if err != nil {
		return fmt.Errorf("wgmng: failed to stop session: %w", err)
	}
	return nil
}

func (w WireGuard) GenKeys() (WireGuardKeys, error) {
	var keys WireGuardKeys
	wgCom := exec.Command("wg", "genkey")
	wgKeyB, err := wgCom.Output()
	if err != nil {
		return keys, fmt.Errorf("RegisterQueue: failed to get generate peer key: %w", err)
	}
	wgKey := strings.TrimSuffix(string(wgKeyB[:]), "\n")

	wgCom = exec.Command("wg", "genpsk")
	wgKeyPreB, err := wgCom.Output()
	if err != nil {
		return keys, fmt.Errorf("RegisterQueue: failed to get generate peer preshared key: %w", err)
	}
	wgKeyPre := strings.TrimSuffix(string(wgKeyPreB[:]), "\n")

	wgCom = exec.Command("wg", "pubkey")
	wgCom.Stdin = bytes.NewBuffer(wgKeyB)
	wgKeyPubB, err := wgCom.Output()
	if err != nil {
		return keys, fmt.Errorf("RegisterQueue: failed to get generate pub key: %w", err)
	}
	wgKeyPub := strings.TrimSuffix(string(wgKeyPubB[:]), "\n")
	keys = WireGuardKeys{
		Private:   wgKey,
		PreShared: wgKeyPre,
		Public:    wgKeyPub,
	}
	return keys, nil
}
