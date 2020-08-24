package airtrack

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
)

type GenerateKey struct{}

func (c *GenerateKey) Run(ctx *Context) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return errors.Wrapf(err, "reading random bytes for key")
	}
	fmt.Println(base64.StdEncoding.EncodeToString(key))
	return nil
}
