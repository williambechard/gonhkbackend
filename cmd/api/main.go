package main

import (
	"fmt"
	"net/http"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/service"
	"nhknewseasybkend/internal/util"
	"nhknewseasybkend/internal/worker"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// RegisterRoutes is defined in routes.go

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
	rootEnvPath := ".env"
	err := godotenv.Load(rootEnvPath)
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return host, port, err
}

func SetupLogger(envErr error) {
	util.InitLogger()
	if envErr != nil {
		util.Log(fmt.Sprintf("Warning: .env file not loaded: %v", envErr), util.LogTypeWarn)
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
	util.Log(fmt.Sprintf("API server running on http://%s/ ...", addr), util.LogTypeLog)
	fmt.Printf("API server running on http://%s/api/ ...\n", addr)
	var err error
	if srv != nil {
		err = srv.ListenAndServe()
	} else {
		err = http.ListenAndServe(addr, mux)
	}
	if err != nil {
		util.Log(fmt.Sprintf("Server error: %v", err), util.LogTypeError)
	}

	util.CloseLogger()
	return err
}

// Run contains the main logic and is testable
// Run can accept a custom ListenAndServeFunc for testability
func Run(srv ListenAndServeFunc, host string, port string) int {
	//ClearLogFile()

	err := StartServer(host, port, srv)
	if err != nil {
		return 1
	}
	return 0
}

func printEnvMasked(keys []string) {
	for _, key := range keys {
		val := os.Getenv(key)
		masked := "XXXXXXX"
		if val == "" {
			masked = "(not set)"
		}
		fmt.Printf("%s: %s\n", key, masked)
	}
}

func main() {
	// Clear log file before any logging
	ClearLogFile()
	host, port, envErr := SetupEnv()
	SetupLogger(envErr)

	// Load .env at the very start so all packages get env vars
	_ = godotenv.Load(".env")
	util.Log("SUPABASE_URL: XXXXXXX", util.LogTypeLog)
	util.Log("SUPABASE_SERVICE_ROLE_KEY: XXXXXXX", util.LogTypeLog)
	server.InitSupabase()

	// Fetch parts of speech before starting workers
	posList, err := service.GetAllPartsOfSpeechGraphQL()
	if err != nil {
		util.Log(fmt.Sprintf("[Startup] Error fetching parts of speech: %v", err), util.LogTypeError)
		os.Exit(1)
	}
	util.Log(fmt.Sprintf("[Startup] Loaded %d parts of speech", len(posList)), util.LogTypeLog)

	service.InitWords()

	grammarList, err := service.GetAllGrammarGraphQL()
	if err != nil {
		util.Log(fmt.Sprintf("[Startup] Error fetching grammar: %v", err), util.LogTypeError)
		os.Exit(1)
	}
	util.Log(fmt.Sprintf("[Startup] Loaded %d grammar points", len(grammarList)), util.LogTypeLog)

	// Log parts of speech pull at the very start
	util.Log("[Startup] Pulling parts of speech from Supabase...", util.LogTypeLog)

	// Start the LinkTranslationWorker (runs every 5 minutes)
	linkWorker := worker.NewLinkTranslationWorker(5 * time.Minute)
	linkWorker.RunOnce = true // Only run once for testing
	linkWorker.Start()

	code := Run(nil, host, port)
	if code != 0 {
		util.Log("Server failed to start. Check logs/log.txt for details. Is port 4200 already in use?", util.LogTypeError)
		os.Exit(code)
	}
}
