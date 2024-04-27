package fitserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"

	// Blank import to embed config.json
	_ "embed"

	gateway "github.com/ekotlikoff/gofit/internal/frontend"
	"github.com/ekotlikoff/gofit/internal/server"
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
		ServerPort  int
		LogFile     string
		Quiet       bool
	}
)

// RunServer runs the gofit server
func RunServer() {
	c := loadConfig()
	RunServerWithConfig(c)
}

// RunServerWithConfig runs the gofit server with a custom config
func RunServerWithConfig(config Configuration) {
	configureLogging(config)
	fitserver := server.FitServer{
		BasePath: config.BasePath,
		Port:     config.ServerPort,
	}
	go fitserver.Serve()
	serverURL, _ := url.Parse("http://localhost:" + strconv.Itoa(config.ServerPort))
	gw := gateway.Gateway{
		BasePath: config.BasePath,
		Port:     config.GatewayPort,
		Backend:  serverURL,
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
