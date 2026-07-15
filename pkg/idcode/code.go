package idcode

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Device 生成设备内码：D + 16 hex（共 17 字符）。
func Device() (string, error) {
	return generate("D")
}

// Channel 生成通道内码：C + 16 hex（共 17 字符）。
func Channel() (string, error) {
	return generate("C")
}

// MustDevice 生成设备内码；失败时 panic（仅用于不可能失败的启动路径慎用）。
func MustDevice() string {
	s, err := Device()
	if err != nil {
		panic(err)
	}
	return s
}

// MustChannel 生成通道内码。
func MustChannel() string {
	s, err := Channel()
	if err != nil {
		panic(err)
	}
	return s
}

func generate(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate internal code: %w", err)
	}
	return prefix + hex.EncodeToString(b[:]), nil
}
