package config

import (
	"os"
)

type Config struct {
	listenAddr 		string
	FTPRoot			string
	DBpath 			string	
}



func Load() (*Config, error) {
	config := &Config{
        listenAddr:     getEnv("LISTEN_ADDR" , ":2121"), 
        FTPRoot:        getEnv("FTP_ROOT", "./"),
        DBpath:         getEnv("DB_PATH", "./ftp.db"),
    }

    return config, nil
}



func getEnv(key,fallback string) string {
	if value , ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

