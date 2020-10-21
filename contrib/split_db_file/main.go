package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type (
	aircraft struct {
		Registration string `json:"r"`
		TypeCode     string `json:"t"`
		F            string `json:"f"`
		Description  string `json:"d"`
	}

	acJSONAsSlice aircraft

	dbJSON map[string]*acJSONAsSlice
)

// MarshalJSON - custom json marshalling - returns array of strings
func (t acJSONAsSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{t.Registration, t.TypeCode, t.F, t.Description})
}

var useShards = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"A", "B", "C", "D", "E", "F", "3D", "40", "48",
	"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7",
	"A8", "A9", "AA", "AB", "AC", "C0", "A00",
	"A19", "C00", "C04", "C05",
}

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

	db := dbJSON{}
	err = json.Unmarshal(dat, &db)
	if err != nil {
		panic(err)
	}

	shardDb := buildDb(useShards, db)

	// Write each shard file
	for fileChar, shardDb := range shardDb {
		err = writeDbShardFile(outDirectory, fileChar, shardDb)
		if err != nil {
			panic(err)
		}
	}

	// write list of useShards to files.json
	file, err := json.Marshal(useShards)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(outDirectory+"/files.json", file, 0644)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}

// buildDb splits db into shards using the list of shards provided
func buildDb(shards []string, db dbJSON) map[string]dbJSON {
	shardDb := map[string]dbJSON{}
	for _, shard := range shards {
		shardDb[shard] = dbJSON{}
	}

	// Split out DB into separate useShards
	for icao := range db {
		// Look for longest matching shard
		var bestShard string
		for _, try := range shards {
			if icao[:len(try)] == try {
				if len(try) > len(bestShard) {
					bestShard = try
				}
			}
		}
		// Trim off shard from key in dbJSON
		shardDb[bestShard][icao[len(bestShard):]] = db[icao]
	}
	return shardDb
}
// writeDbShardFile writes shardDb to a file named using fileChar, in outDirectory
func writeDbShardFile(outDirectory string, fileChar string, shardDb dbJSON) error {
	// find children for this shard
	var children []string
	for _, shard := range useShards {
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
		return err
	}
	err = ioutil.WriteFile(outDirectory+"/"+fileChar+".json", file, 0644)
	if err != nil {
		return err
	}
	return nil
}