package config

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var expected = `{
  "key": "value",
  "sub": {
    "sub": {
      "key": 1
    }
  }
}`

func TestBasic(t *testing.T) {
	new()
	SetConfigFileName("basic")
	AddSearchPath("./test")
	err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	val := Get("hello")
	if val.(string) != "world" {
		t.Errorf("Unexpected value: %v", val)
	}
}

func TestNested(t *testing.T) {
	new()
	AddSearchPath("./test")
	SetConfigFileName("nested")
	err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	val := Get("first.second")
	if val.(string) != "arrived" {
		t.Errorf("Unexpected value: %v", val)
	}
}

func TestNewConfig(t *testing.T) {
	new()
	AddSearchPath("./test")
	SetConfigFileName("new")
	err := ParseConfig()
	if err == nil {
		t.Error("Expected an error")
	}

	if _, ok := err.(*FileNotFoundError); ok {
		Sync()
	}
	_, err = os.Stat("./test/new")
	if err != nil {
		t.Error(err)
	}

	if err = Set("key", "value"); err != nil {
		t.Error(err)
	}
	if err = Set("sub.sub.key", 1); err != nil {
		t.Error(err)
	}

	Sync()

	content, err := ioutil.ReadFile("./test/new")
	if err != nil {
		t.Error(err)
	}
	if strings.Compare(string(content), expected) != 0 {
		t.Error("Compared config files do not match: expecting ", expected, " got ", string(content))
	}

	os.Remove("./test/new")
}
