package reghive

import (
	"io/ioutil"
	"strings"

	"github.com/JoshuaDoes/json"
)

var (
	nodeKeys map[string]string
)

func NodeKeyVal(input string) string {
	if input == "" {
		return ""
	}
	if nodeKeys == nil {
		nodesBytes, err := ioutil.ReadFile("bcdnodes.json")
		if err != nil {
			return input
		}
		testKeys := make(map[string]string)
		err = json.Unmarshal(nodesBytes, &testKeys)
		if err != nil {
			panic(err)
		}
		nodeKeys = make(map[string]string)
		for key, val := range testKeys {
			nodeKeys[strings.ToLower(key)] = strings.ToLower(val)
		}
	}
	input = strings.ToLower(input)

	for key, val := range nodeKeys {
		if input == key {
			return val
		}
	}

	return input
}

func NodeValKey(input string) string {
	if input == "" {
		return ""
	}
	if nodeKeys == nil {
		nodesBytes, err := ioutil.ReadFile("bcdnodes.json")
		if err != nil {
			return input
		}
		testKeys := make(map[string]string)
		err = json.Unmarshal(nodesBytes, &testKeys)
		if err != nil {
			panic(err)
		}
		nodeKeys = make(map[string]string)
		for key, val := range testKeys {
			nodeKeys[strings.ToLower(key)] = strings.ToLower(val)
		}
	}
	input = strings.ToLower(input)

	for key, val := range nodeKeys {
		if input == val {
			return key
		}
	}

	return input
}
