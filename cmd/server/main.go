package main

import (
	"github.com/BigBr41n/GFTP-server/internal/config"
	"github.com/BigBr41n/GFTP-server/internal/ftp"
)

func main() {
    //load the configurations
    cfg := config.Load();

    //create a server
    server := ftp.NewServer(cfg)

    //start the server
    server.ListenAndServe()
}

