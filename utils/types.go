package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func ListContains(slice []string, obj string) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == obj {
			return true
		}
	}
	return false
}

func ListDelete(slice []string, obj string) (result []string) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == obj {
			continue
		}
		result = append(result, slice[i])
	}
	return
}

func RandomString(len int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:len]
}
