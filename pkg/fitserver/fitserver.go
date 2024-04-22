package fitserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	// Blank import to embed config.json
	_ "embed"

	gateway "github.com/ekotlikoff/gofit/internal/frontend"
)

//go:embed config.json
var config []byte

type (
	// Configuration is a struct that configures the chess server
	Configuration struct {
		ServiceName string
		Environment string
		BasePath    string
		GatewayPort int
		LogFile     string
		Quiet       bool
	}
)

// RunServer runs the gochess server
func RunServer() {
	c := loadConfig()
	RunServerWithConfig(c)
}

// RunServerWithConfig runs the gochess server with a custom config
func RunServerWithConfig(config Configuration) {
	configureLogging(config)
	gw := gateway.Gateway{
		BasePath: config.BasePath,
		Port:     config.GatewayPort,
	}
	gw.Serve()
}

func configureLogging(config Configuration) {
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
	}
	if config.Quiet {
		log.SetOutput(ioutil.Discard)
	}
}

func loadConfig() Configuration {
	configuration := Configuration{}
	err := json.Unmarshal(config, &configuration)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	return configuration
}
