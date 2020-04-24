package main

import (
	"crypto/rsa"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/openware/rango/pkg/auth"
	"github.com/openware/rango/pkg/upstream"
)

var (
	addr   = flag.String("addr", ":8080", "http service address")
	pubKey = flag.String("pubKey", "config/rsa-key.pub", "path to public key")
)

const prefix = "Bearer "

type httpHanlder func(w http.ResponseWriter, r *http.Request)

func token(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(string(authHeader), prefix) {
		return ""
	}

	return authHeader[len(prefix):]
}

func authHandler(h httpHanlder, key *rsa.PublicKey) httpHanlder {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := auth.ParseAndValidate(token(r), key)
		if err == nil {
			h(w, r)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	}
}

func main() {
	flag.Parse()
	hub := newHub()

	ks := auth.KeyStore{}
	ks.GenerateKeys()
	if err := ks.LoadPublicKey(*pubKey); err != nil {
		log.Panic(err)
	}

	go hub.run()
	go upstream.AMQPUpstream(hub.messages)

	wsHandler := func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}

	http.HandleFunc("/public", wsHandler)
	http.HandleFunc("/private", authHandler(wsHandler, ks.PublicKey))

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
