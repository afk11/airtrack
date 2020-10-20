package main

import (
	"bytes"
	"fmt"
	asset "github.com/afk11/airtrack/pkg/assets"
	"github.com/afk11/airtrack/pkg/iso3166"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func main() {
	countryCodesData, err := asset.Asset("assets/iso3166_country_codes.txt")
	if err != nil {
		panic(err)
	}
	countryCodeRows, err := iso3166.ParseColumnFormat(bytes.NewBuffer(countryCodesData))
	if err != nil {
		panic(err)
	}
	countryCodeStore, err := iso3166.New(countryCodeRows)
	if err != nil {
		panic(err)
	}

	codes := countryCodeStore.GetAlphaTwoCodes()
	for _, code := range codes {
		codeLwr := strings.ToLower(code.String())
		res, err := http.Get(fmt.Sprintf(
			"https://www.openaip.net/customer_export_akfshb9237tgwiuvb4tgiwbf/%s_wpt.aip",
			codeLwr))
		if err != nil {
			panic(err)
		}
		if res.StatusCode != 200 {
			_ = res.Body.Close()
			continue
		}
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		_ = res.Body.Close()
		if len(b) == 0 {
			continue
		}
		err = ioutil.WriteFile(fmt.Sprintf(
			"resources/airports/%s.aip",
			codeLwr), b, 0644)
		if err != nil {
			panic(err)
		}
	}
	os.Exit(0)
}
