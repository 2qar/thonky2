package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// cacheFilename returns a filename based on attr and sheetID
func cacheFilename(attr, sheetID string) string {
	return "cache/" + sheetID + "/" + attr + ".json"
}

// saveSheetAttr writes a sheet attribute to the disk
func saveSheetAttr(v interface{}, attr, sheetID string) error {
	log.Printf("saving %s for [%s]\n", attr, sheetID)
	filename := cacheFilename(attr, sheetID)
	m, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, m, 0644)
	if err != nil {
		return err
	}
	return nil
}

// loadSheetAttr loads a sheet attribute from the cache
func loadSheetAttr(v interface{}, attr, sheetID string) error {
	log.Printf("loading %s for [%s]\n", attr, sheetID)
	b, err := ioutil.ReadFile(cacheFilename(attr, sheetID))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &v)
}
