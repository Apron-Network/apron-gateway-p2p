package internal

import (
	"bytes"
	"strings"
)

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

func ServiceHostnameToId(hostname string) string {
	return strings.Replace(hostname, ".", "_", -1)
}

func ServiceHostnameToIdByte(hostname []byte) []byte {
	return bytes.Replace(hostname, []byte("."), []byte("_"), -1)
}