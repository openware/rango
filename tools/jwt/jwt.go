package main

import (
	"flag"
	"fmt"

	"github.com/openware/rango/pkg/auth"
)

var (
	port  = flag.String("port", "7070", "Port to bind")
	uid   = flag.String("uid", "IDABC0000001", "UID")
	email = flag.String("email", "admin@barong.io", "Email")
	role  = flag.String("role", "admin", "Role")
	level = flag.Int("level", 3, "Level")
)

func main() {
	flag.Parse()

	ks, _ := auth.LoadOrGenerateKeys("config/rsa-key", "config/rsa-key.pub")
	t, _ := auth.ForgeToken(*uid, *email, *role, *level, ks.PrivateKey, nil)
	fmt.Print(t)
}
