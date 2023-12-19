package main

import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type Geometry struct {
	Chips  [][]interface{} `yaml:"chips"`
	Chips  [][]interface{} `yaml:"pixels"`
}

var NChips = 100
var NChannels = 64

func LoadGeometry(geofile string) {
   
	// Read YAML file
	yamlFile, err := ioutil.ReadFile("test.yaml")
	if err != nil {
		panic(err)
	}

	// Create an instance of the Config struct to store the unmarshaled data
	var config Config

	// Unmarshal YAML into the Config struct
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}


}

func ChipChannelToXY(Chip uint8, Channel uint8){
	// TODO: generalize for multiple tiles
	Pixel :=  
	index := (Chip - 11) * NChannels + Channel



}
   
