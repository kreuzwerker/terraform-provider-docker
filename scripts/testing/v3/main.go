package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const listenAddr = ":8085" // changed here on purpose

type config struct {
	Prefix string `json:"prefix"`
}

func main() {
	configsContent, err := os.ReadFile("configs.json")
	if err != nil {
		log.Fatalf("cannot open 'configs.json': %s", err)
	}

	var configs config
	err = json.Unmarshal(configsContent, &configs)
	if err != nil {
		log.Fatalf("cannot unmarshal 'configs.json': %s", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(fmt.Sprintf("%s - Hello World!", configs.Prefix)))
		if err != nil {
			log.Fatalln("failed to write for path '/'")
		}
	})

	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalf("failed to listen and server on port '%s'", listenAddr)
	}
}
