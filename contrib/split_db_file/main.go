package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Aircraft struct {
	Registration string `json:"r"`
	TypeCode     string `json:"t"`
	F            string `json:"f"`
	Description  string `json:"d"`
}

type AcJsonAsSlice Aircraft

func (t AcJsonAsSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{t.Registration, t.TypeCode, t.F, t.Description})
}

type dbJson map[string]*AcJsonAsSlice

func main() {
	if len(os.Args) < 2 {
		fmt.Println("missing db file path")
		fmt.Println("usage: <db_file> <output_directory>")
		os.Exit(1)
	} else if len(os.Args) < 3 {
		fmt.Println("missing output directory")
		fmt.Println("usage: <db_file> <output_directory>")
		os.Exit(1)
	}
	dbFile := os.Args[1]
	outDirectory := os.Args[2]

	// Parse DB file
	dat, err := ioutil.ReadFile(dbFile)
	if err != nil {
		panic(err)
	}

	db := dbJson{}
	err = json.Unmarshal(dat, &db)
	if err != nil {
		panic(err)
	}

	// List of shards
	shards := []string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"A", "B", "C", "D", "E", "F", "3D", "40", "48",
		"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7",
		"A8", "A9", "AA", "AB", "AC", "C0", "A00",
		"A19", "C00", "C04", "C05",
	}

	// Init each shard
	shardDb := map[string]dbJson{}
	for _, shard := range shards {
		shardDb[shard] = dbJson{}
	}

	// Split out DB into separate shards
	for icao, ac := range db {
		// Look for longest matching shard
		var bestShard string
		for _, try := range shards {
			if icao[:len(try)] == try {
				if len(try) > len(bestShard) {
					bestShard = try
				}
			}
		}
		// Trim off shard from key in dbJson
		shardDb[bestShard][icao[len(bestShard):]] = ac
	}

	// Write each shard file
	for fileChar, shardDb := range shardDb {
		// find children for this shard
		var children []string
		for _, shard := range shards {
			if len(shard) == len(fileChar)+1 && fileChar == shard[0:len(fileChar)] {
				children = append(children, shard)
			}
		}

		// build up file with shard + children
		shardFile := map[string]interface{}{}
		for k, v := range shardDb {
			shardFile[k] = v
		}
		if len(children) > 0 {
			shardFile["children"] = children
		}

		file, err := json.Marshal(shardFile)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(outDirectory+"/"+fileChar+".json", file, 0644)
		if err != nil {
			panic(err)
		}
	}

	// write list of shards to files.json
	file, err := json.Marshal(shards)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(outDirectory+"/files.json", file, 0644)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}
