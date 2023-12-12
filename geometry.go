package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

var Geometry = make(map[string]interface{})

var NChips = 100
var NChannels = 64

func LoadGeometry(geofile string) {
   
	yamlFile, err := ioutil.ReadFile(geofile)
	if err != nil {
	 fmt.Printf("yamlFile.Get err #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, obj)
	if err != nil {
	 fmt.Printf("Unmarshal: %v", err)
	}
	//fmt.Println(obj["chips"])
}

func ChipChannelToXY(Chip uint8, Channel uint8){
	// TODO: generalize for multiple tiles
	Pixel :=  
	index := (Chip - 11) * NChannels + Channel



}
   
