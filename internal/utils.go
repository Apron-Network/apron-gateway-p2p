package internal

import (
	"bytes"
)

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

func ServiceHostnameToIdByte(hostname []byte) []byte {
	return bytes.Replace(hostname, []byte("."), []byte("_"), -1)
}
