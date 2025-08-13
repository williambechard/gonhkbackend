package main

import (
	"fmt"
	"net/http"
	"nhknewseasybkend/internal/util"
	"os"

	"github.com/joho/godotenv"
)

// Import RegisterRoutes from links.go

func ClearLogFile() {
	logPath := util.GetLogFilePath()
	if _, err := os.Stat(logPath); err == nil {
		f, err := os.OpenFile(logPath, os.O_TRUNC|os.O_WRONLY, 0666)
		if err == nil {
			f.Close()
		}
	}
}

func SetupEnv() (string, string, error) {
	// Always load .env from project root
	rootEnvPath := "c:/Users/ury2o/Documents/nhknewseasybkend/.env"
	err := godotenv.Load(rootEnvPath)
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "4200"
	}
	return host, port, err
}

func SetupLogger(envErr error) {
	util.InitLogger()
	if envErr != nil {
		util.LogToFile(fmt.Sprintf("Warning: .env file not loaded: %v", envErr))
	}
}

// ListenAndServeFunc is an interface for testable server start
type ListenAndServeFunc interface {
	ListenAndServe() error
}

// StartServer can accept a custom ListenAndServeFunc for testability
func StartServer(host, port string, srv ListenAndServeFunc) error {
	addr := fmt.Sprintf("%s:%s", host, port)
	mux := http.NewServeMux()
	RegisterRoutes(mux)
	util.LogToFile(fmt.Sprintf("API server running on http://%s/ ...", addr))
	var err error
	if srv != nil {
		err = srv.ListenAndServe()
	} else {
		err = http.ListenAndServe(addr, mux)
	}
	if err != nil {
		util.LogToFile(fmt.Sprintf("Server error: %v", err))
	}
	util.CloseLogger()
	return err
}

// Run contains the main logic and is testable
// Run can accept a custom ListenAndServeFunc for testability
func Run(srv ListenAndServeFunc) int {
	ClearLogFile()
	host, port, envErr := SetupEnv()
	SetupLogger(envErr)
	err := StartServer(host, port, srv)
	if err != nil {
		return 1
	}
	return 0
}

func main() {
	code := Run(nil)
	if code != 0 {
		fmt.Println("Server failed to start. Check logs/log.txt for details. Is port 4200 already in use?")
		os.Exit(code)
	}
}
