package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/vizn3r/cloud/lib/logger"
	"gopkg.in/yaml.v3"
)

var (
	log    = logger.New("CONF", logger.Yellow)
	global any
	mu     sync.RWMutex
)

func FindAndLoadConfig[T any](conf string) error {
	var err error
	var configPath string

	// Check if config file exists
	possiblePaths := []string{
		os.Getenv("CONFIG_PATH"),
		conf,
		"./" + conf,
		"/etc/vizn3r-cloud/" + conf,
		"/usr/local/etc/vizn3r-cloud/" + conf,
		"$HOME/.config/vizn3r-cloud/" + conf,
		"$HOME/.config/cloud/" + conf,
	}

	for _, path := range possiblePaths {
		if _, err = os.Stat(path); err == nil && conf != "" {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return fmt.Errorf("couldn't find config file")
	}

	return LoadConfig[T](configPath)
}

func decodeBytes[T any](data []byte, ftype string) (*T, error) {
	var conf T
	switch ftype {
	case "json":
		parser := json.NewDecoder(strings.NewReader(string(data)))
		parser.DisallowUnknownFields()
		if err := parser.Decode(&conf); err != nil {
			return nil, fmt.Errorf("couldn't decode config file %s", err)
		}
		return &conf, nil
	case "yaml":
		parser := yaml.NewDecoder(strings.NewReader(string(data)))
		parser.KnownFields(true)
		if err := parser.Decode(&conf); err != nil {
			return nil, fmt.Errorf("couldn't decode config file %s", err)
		}
		return &conf, nil
	default:
		return nil, fmt.Errorf("unknown config file type")
	}
}

func LoadFromBytes[T any](data []byte, ftype string) error {
	conf, err := decodeBytes[T](data, ftype)
	if err != nil {
		return err
	}

	mu.Lock()
	global = conf
	mu.Unlock()

	return nil
}

func LoadConfig[T any](path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("couldn't open '%s' config file", path)
	}

	if data == nil {
		return fmt.Errorf("couldn't read '%s' config file", path)
	}

	parts := strings.Split(path, ".")
	ftype := strings.ToLower(parts[len(parts)-1])
	err = LoadFromBytes[T](data, ftype)
	if err != nil {
		return err
	}

	return nil
}

func Get[T any]() *T {
	mu.RLock()
	defer mu.RUnlock()

	if global == nil {
		log.Fatal("Global config is not initialized")
		return nil
	}

	conf, ok := global.(*T)
	if !ok {
		log.Fatal("Global config type assertion failed")
		return nil
	}

	return conf
}
