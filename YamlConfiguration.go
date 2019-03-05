package yamlasprop

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/newm4n/go-utility"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	TABSIZE           = 4
	REFERENCE_PATTERN = "\\${[a-zA-Z0-9.]+}"
)

type yamlindice struct {
	index int
	text  string
}

// Yaml struct holds the parsing instance object.
type Yaml struct {
	tabSize       int
	prevIsArr     bool
	arrIndex      int
	stack         go_utility.Stack
	envVarOveride *EnvVarOverride
	Properties    map[string]string
}

// EnvVarOverride holds information about how to overide configuration values with environment variables.
type EnvVarOverride struct {
	EnvVarOverride bool
	WithReplacer   map[string]string
	WithPrefix     string
}

// String get the content of this parser
func (yml *Yaml) String() string {
	buf := bytes.Buffer{}
	for k, v := range yml.Properties {
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	return buf.String()
}

// NewYaml function create and parse the configuration data,
// if override is provided, it will also try to look for matchine environment variable and use it.
// if override is nil, no environment variable lookup will be done.
func NewYaml(p []byte, override *EnvVarOverride) (*Yaml, error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	y := &Yaml{
		prevIsArr:     false,
		arrIndex:      0,
		Properties:    make(map[string]string),
		envVarOveride: override,
	}
	for scanner.Scan() {
		if err := y.addLine(scanner.Text()); err != nil {
			return nil, err
		}
	}
	y.referenceLink()
	y.doEnvVarOverride()
	return y, nil
}

func (yml *Yaml) doEnvVarOverride() {
	if yml.envVarOveride != nil {
		for key, _ := range yml.Properties {
			var varName = key
			if yml.envVarOveride.WithReplacer != nil {
				for old, new := range yml.envVarOveride.WithReplacer {
					varName = strings.Replace(varName, old, new, -1)
				}
			}
			varName = fmt.Sprintf("%s%s", yml.envVarOveride.WithPrefix, varName)
			varName = strings.ToUpper(varName)

			envValue := os.Getenv(varName)
			if len(envValue) > 0 {
				yml.Properties[key] = envValue
			}
		}
	}
}

func (yml *Yaml) referenceLink() {
	var refFound = true
	re := regexp.MustCompile(REFERENCE_PATTERN)
	for refFound {
		refFound = false
		for k, v := range yml.Properties {
			if re.MatchString(v) {
				str := re.FindString(v)
				ref := str[2 : len(str)-1]
				//log.Print(ref)
				nv := strings.Replace(v, str, yml.Properties[ref], -1)
				yml.Properties[k] = nv
				refFound = true
			}
		}
	}
}

func (yml *Yaml) addLine(line string) error {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	trimed := strings.TrimSpace(line)

	if trimed[:1] == "-" {
		val := strings.TrimSpace(trimed[1:])
		if !yml.prevIsArr {
			yml.arrIndex = 0
		} else {
			yml.arrIndex++
		}
		yml.Properties[fmt.Sprintf("%s.[%d]", yml.getPathString(), yml.arrIndex)] = val
		yml.prevIsArr = true
		return nil
	}
	yml.prevIsArr = false
	yml.arrIndex = 0
	var index = 0
	for i := 0; i < len(line); i++ {
		c := line[i : i+1]
		if c == " " {
			index++
		} else if c == "\t" {
			if yml.tabSize == 0 {
				index += TABSIZE
			} else {
				index += yml.tabSize
			}
		} else {
			break
		}
	}
	if strings.Index(trimed, ":") == -1 {
		return errors.New(fmt.Sprintf("malformed yaml file. near \"%s\"", line))
	}
	key := trimed[:strings.Index(trimed, ":")]
	if !yml.stack.IsEmpty() {
		if yml.stack.Peek().(yamlindice).index == index {
			yml.stack.Pop()
		} else if yml.stack.Peek().(yamlindice).index > index {
			for yml.stack.Peek().(yamlindice).index >= index {
				yml.stack.Pop()
				if yml.stack.IsEmpty() {
					break
				}
			}
		}
	}
	yml.stack.Push(yamlindice{
		index: index,
		text:  key,
	})
	//log.Printf("Trimmed %s - %d - %d", trimed, index, yml.stack.Size())
	value := trimed[strings.Index(trimed, ":")+1:]
	if value != "" {
		yml.Properties[yml.getPathString()] = strings.TrimSpace(value)
	}
	return nil
}

func (yml *Yaml) getPathString() string {
	array := yml.stack.PeekStack()
	buf := bytes.Buffer{}
	for i, v := range array {
		if i > 0 {
			buf.WriteString(".")
		}
		yi := v.(yamlindice)
		buf.WriteString(yi.text)
	}
	return string(buf.Bytes())
}

// Clear will cleare up the configuration content.
func (yml *Yaml) Clear() {
	for k := range yml.Properties {
		delete(yml.Properties, k)
	}
}

// ListKeys List all configuration keys
func (yml *Yaml) ListKeys() []string {
	ret := make([]string,0,len(yml.Properties))
	for key,_ := range yml.Properties {
		ret = append(ret, key)
	}
	return ret
}

// Get will get the configuration value of a key.
func (yml *Yaml) Get(key string) string {
	if str, ok := yml.Properties[key]; !ok {
		return ""
	} else {
		return str
	}
}

// GetRequired is like Get, but if no configuration with specified key found, it will yield an error.
func (yml *Yaml) GetRequired(key string) (string, error) {
	if str, ok := yml.Properties[key]; !ok {
		return "", errors.New(fmt.Sprintf("Configuration with key %s not exist.", key))
	} else {
		return str, nil
	}
}

// GetDefaulted is like Get, but if no configuration with specified key found, the specified defaulted param will be returned.
func (yml *Yaml) GetDefaulted(key, defaulted string) string {
	if str, ok := yml.Properties[key]; !ok {
		return defaulted
	} else {
		return str
	}
}

// HaveKey will check if the key parameter is a valid configuration key.
func (yml *Yaml) HaveKey(key string) bool {
	_, ok := yml.Properties[key]
	return ok
}

// Unmarshal the loaded configuration into configuration struct.
func (yml *Yaml) Unmarshal(target interface{}) error {

	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("can not unmarshal %s", reflect.TypeOf(target))
	}

	return yml.unmarshal(rv, "")
}

func (yml *Yaml) unmarshal(v reflect.Value, keyPath string) error {

	var rv reflect.Value

	if v.Kind() == reflect.Ptr {
		rv = v.Elem()
	} else {
		rv = v
	}

	switch rv.Type().Kind() {
	case reflect.String:
		rv.SetString(yml.Properties[keyPath])
		return nil
	case reflect.Bool:
		str := strings.ToUpper(yml.Properties[keyPath])
		rv.SetBool(str == "TRUE" || str == "YES" || str == "ON")
		return nil
	case reflect.Int:
		val, err := strconv.ParseInt(yml.Properties[keyPath], 10, 64)
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Int8:
		val, err := strconv.ParseInt(yml.Properties[keyPath], 10, 8)
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Int16:
		val, err := strconv.ParseInt(yml.Properties[keyPath], 10, 16)
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Int32:
		val, err := strconv.ParseInt(yml.Properties[keyPath], 10, 32)
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Int64:
		val, err := strconv.ParseInt(yml.Properties[keyPath], 10, 64)
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Float32:
		val, err := strconv.ParseFloat(yml.Properties[keyPath], 32)
		if err != nil {
			return err
		}
		rv.SetFloat(val)
		return nil
	case reflect.Float64:
		val, err := strconv.ParseFloat(yml.Properties[keyPath], 64)
		if err != nil {
			return err
		}
		rv.SetFloat(val)
		return nil
	case reflect.Uint:
		val, err := strconv.ParseUint(yml.Properties[keyPath], 10, 32)
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	case reflect.Uint8:
		val, err := strconv.ParseUint(yml.Properties[keyPath], 10, 8)
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	case reflect.Uint16:
		val, err := strconv.ParseUint(yml.Properties[keyPath], 10, 16)
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	case reflect.Uint32:
		val, err := strconv.ParseUint(yml.Properties[keyPath], 10, 32)
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	case reflect.Uint64:
		val, err := strconv.ParseUint(yml.Properties[keyPath], 10, 64)
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		rv = v.Elem()
	} else {
		rv = v
	}

	t := rv.Type()

	for fi := 0; fi < t.NumField(); fi++ {
		field := t.Field(fi)
		var tn string
		var path string

		tagName, ok := field.Tag.Lookup("yaml")
		if ok {
			tn = tagName
		} else {
			tn = field.Name
		}
		if len(keyPath) == 0 {
			path = tn
		} else {
			path = fmt.Sprintf("%s.%s", keyPath, tn)
		}

		fieldValue := rv.FieldByName(field.Name)
		err := yml.unmarshal(fieldValue, path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error while unmarshalling type %s. got %v\n", field.Name, err)
		}
	}
	return nil
}


