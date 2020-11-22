package app

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

func uid() (string, error) {
	var err error

	var r = make([]byte, 16)
	_, err = rand.Read(r)
	if err != nil {
		return "", err
	}

	var n = make([]byte, binary.MaxVarintLen64)
	var t = time.Now().UnixNano()
	binary.PutVarint(n, t)

	return hex.EncodeToString(append(r, n...)), nil
}
