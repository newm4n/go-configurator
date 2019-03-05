# go-configurator

Another library that it sole purpose is to read application configuration from `YAML` file. You can
then access the onfiguration in a "properties" styled configuration or map them directly
to your struct model. 

## Getting go-configurator

As easy as :

```text
$ go get github.com/newm4n/go-configurator
```

## Parsing YML file using go-configurator

Assuming the following YAML file :

```yaml
app:
  name: AWESOME
  logLevel: Debug
server:
  host: 10.123.123.23
  port: 12345
db:
  host: ${server.host}
  port: 321
  user:
    name: dbuser
    pass: dbsecret
some:
  array:
    - "One"
    - "Two"

```

Parse the yaml bytes as the following.
```go
// load your bytes here.
yaml, err := NewYaml(bytes, nil)
if err != nil {
    panic(fmt.sprintf("failed loading configuration data. got %v", err))
}
```

Now you can access your configuration with "Properties" style.

```go
fmt.println(yaml.Get("app.name"))       // prints AWESOME
fmt.println(yaml.Get("app.logLevel"))   // prints Debug
fmt.println(yaml.Get("server.host"))    // prints 10.123.123.23
fmt.println(yaml.Get("server.port"))    // prints 12345
fmt.println(yaml.Get("db.user.name"))   // prints dbuser
fmt.println(yaml.Get("some.array.[0]"))   // prints One
fmt.println(yaml.Get("some.array.[1]"))   // prints Two
```

## Load configuration into a struct

Assuming we want to load the above YML data into a struct.
So we define our structs.

```go
type Configuration struct {
	App    Application `yaml:"app"`
	Server Server      `yaml:"server"`
	Db     Database    `yaml:"db"`
}

type Application struct {
	Name     string `yaml:"name"`
	LogLevel string `yaml:"logLevel"`
}

type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Database struct {
    Host string         `yaml:"host"`
    Port int            `yaml:"port"`
    User UserCredential `yaml:"user"`
}

type UserCredential struct {
	Name string     `yaml:"name"`
	Password string `yaml:"pass"`
}

```

Now, we can map them into our struct :

```go
// first we parse the configuration bytes.
yaml, err := NewYaml(bytes, nil)
if err != nil {
    panic(fmt.sprintf("failed loading configuration data. got %v", err))
}

// then we unmarshal them into our struct.
config := Configuration{}
err = yaml.Unmarshal(&config)
if err != nil {
    panic(fmt.sprintf("failed marshaling configuration data. got %v", err))
}

fmt.println(config.App.Name) // prints AWESOME
```

## Automatically overriding configuration values with environment variables.

First we define our environment variable overiding rule.

```go
override := &EnvVarOverride{
    EnvVarOverride: true,
    WithPrefix:     "ENV_",   // this will look for env variable with prefix of "ENV_"
    WithReplacer:   map[string]string{".": "_"}, // this will replace the property key sub-string with another string.
}
```

Then we set some overiding environment variables:

```text
$ export ENV_APP_NAME="EVEN MORE AWESOME" 
```

Now, lets load our YAML data

```go
yaml, err := NewYaml(bytes, override)
if err != nil {
    panic(fmt.sprintf("failed loading configuration data. got %v", err))
}

fmt.println(yaml.Get("app.name"))       // prints EVEN MORE AWESOME
fmt.println(yaml.Get("app.logLevel"))   // prints Debug
```

## Known Issue

* Yaml mapper to struct are not yet support mapping to array
    ```yaml
    some:
      value:
        to:
          array:
            - "value 1"
            - "value 2"
    ```