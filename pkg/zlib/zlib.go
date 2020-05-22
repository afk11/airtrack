package zlib

import (
	"bytes"
	"compress/zlib"
	"github.com/pkg/errors"
	"io/ioutil"
)

func Encode(in []byte) ([]byte, error) {
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	_, err := w.Write(in)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	c := compressed.Bytes()
	return c, nil
}
func Decode(in []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewBuffer(in))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading compressed data")
	}
	return raw, nil
}
