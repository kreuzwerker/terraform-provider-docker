package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type config struct {
	Prefix string `json:"prefix"`
}

func main() {
	configsContent, err := ioutil.ReadFile("configs.json")
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

	http.HandleFunc("/newroute", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("new Route!"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s - Hello World!", configs.Prefix)))
	})

	http.ListenAndServe(":8080", nil)
}
