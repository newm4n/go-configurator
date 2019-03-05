package yamlasprop

import (
	"os"
	"testing"
)

var (
	data = `
one:
  oneone:
    oneoneone:
      oneoneoneone: Hurrah One
    oneonetwo:
      oneonetwoone: Hurrah Two
  onetwo:
    onetwoone: Hurrah Three
    onetwotwo: Hurrah Four
    onetwothree:
      - Hurrah Array One
      - Hurrah Array Two
  onethree: And I Say ${one.oneone.oneoneone.oneoneoneone} To the One
  onefour: Who say ${one.onethree} is cool
`

	data2 = `
grandMom:
  mom:
    kid:
      name: Brian
      age: 24
  dad:
    kid:
      name: Joel
      age: 25
grandDad:
  mom:
    kid:
      name: James
      age: 12
  dad:
    kid:
      name: Francis
      age: 13
  
`
)

type TheRoot struct {
	GrandMom TheGrandParent `yaml:"grandMom"`
	GrandDad TheGrandParent `yaml:"grandDad"`
}

type TheGrandParent struct {
	Mom TheParent `yaml:"mom"`
	Dad TheParent `yaml:"dad"`
}

type TheParent struct {
	Kid TheChild `yaml:"kid"`
}

type TheChild struct {
	Name string `yaml:"name"`
	Age  int    `yaml:"age"`
}

func TestYaml_Unmarshal(t *testing.T) {
	bytes := []byte(data2)

	yaml, err := NewYaml(bytes, nil)
	if err != nil {
		t.FailNow()
	}
	root := TheRoot{}
	err = yaml.Unmarshal(&root)
	if err != nil {
		t.FailNow()
	}
	if root.GrandDad.Dad.Kid.Name != "Francis" {
		t.FailNow()
	}
	if root.GrandDad.Dad.Kid.Age != 13 {
		t.FailNow()
	}
}

func TestNewYaml(t *testing.T) {

	bytes := []byte(data)
	if bytes == nil {
		t.FailNow()
	}

	err := os.Setenv("ENV_ONE_ONETWO_ONETWOTWO", "OVERIDE")
	if err != nil {
		t.FailNow()
	}

	yaml, err := NewYaml(bytes, &EnvVarOverride{
		EnvVarOverride: true,
		WithPrefix:     "ENV_",
		WithReplacer:   map[string]string{".": "_"},
	})

	if err != nil {
		t.FailNow()
	}

	//t.Logf("%s\n", yaml.String())

	equals := func(expect, actual string) {
		if expect != actual {
			t.Logf("\"%s\" != \"%s\"", expect, actual)
			t.Fail()
		}
	}

	equals("Hurrah One", yaml.Get("one.oneone.oneoneone.oneoneoneone"))
	equals("Hurrah Two", yaml.Get("one.oneone.oneonetwo.oneonetwoone"))
	equals("Hurrah Three", yaml.Get("one.onetwo.onetwoone"))
	equals("OVERIDE", yaml.Get("one.onetwo.onetwotwo"))
	equals("Hurrah Array One", yaml.Get("one.onetwo.onetwothree.[0]"))
	equals("Hurrah Array Two", yaml.Get("one.onetwo.onetwothree.[1]"))
	equals("", yaml.Get("one.onetwo.onetwothree.[3]"))
	equals("", yaml.Get("one.onetwo.onetwothree"))
	equals("", yaml.Get("one"))


	_, err =  yaml.GetRequired("nonexistentkey")
	if err == nil {
		t.FailNow()
	}
	val, err :=  yaml.GetRequired("one.onetwo.onetwotwo")
	if err != nil || val != "OVERIDE" {
		t.FailNow()
	}

	if yaml.GetDefaulted("one.onetwo.onetwotwo", "DEFAULT") == "DEFAULT" {
		t.FailNow()
	}

	if yaml.GetDefaulted("nonexistent", "DEFAULT") != "DEFAULT" {
		t.FailNow()
	}
}
