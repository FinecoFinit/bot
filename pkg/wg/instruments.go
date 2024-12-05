package wg

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func (h HighWay) Gen() (string, string, string, error) {
	wgCom := exec.Command("wg", "genkey")
	wgKeyB, err := wgCom.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("RegisterQueue: failed to get generate peer key: %w", err)
	}
	wgKey := strings.TrimSuffix(string(wgKeyB[:]), "\n")

	wgCom = exec.Command("wg", "genpsk")
	wgKeyPreB, err := wgCom.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("RegisterQueue: failed to get generate peer preshared key: %w", err)
	}
	wgKeyPre := strings.TrimSuffix(string(wgKeyPreB[:]), "\n")

	wgCom = exec.Command("wg", "pubkey")
	wgCom.Stdin = bytes.NewBuffer(wgKeyB)
	wgKeyPubB, err := wgCom.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("RegisterQueue: failed to get generate pub key: %w", err)
	}
	wgKeyPub := strings.TrimSuffix(string(wgKeyPubB[:]), "\n")
	return wgKey, wgKeyPre, wgKeyPub, nil
}
