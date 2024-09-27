package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)


func main() {
	ln , err := net.Listen("tcp", ":2121")
	if err!= nil {
        fmt.Println("Error listening:", err.Error())
        return
    }
	defer ln.Close()


	for {
		conn, err := ln.Accept()
        if err!= nil {
            fmt.Println("Error accepting:", err.Error())
            return
        }

        fmt.Println("Connected to:", conn.RemoteAddr())

        // Handle connections in a new goroutine
        go handleConnection(conn)
	}
}


func handleConnection(conn net.Conn) {
	defer conn.Close()

    for {
        message := make([]byte, 1024)
        length, err := conn.Read(message)
        if err!= nil {
            fmt.Println("Error reading:", err.Error())
            break
        }

        fmt.Printf("Received message: %s\n", message[:length])
        conn.Write([]byte("Server received your message \n"))

		// Use bufio to handle spaces in user input
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your reply: ")
		userInput, _ := reader.ReadString('\n')
		userInput = userInput + "\n"

		// Send the user's input back to the client
		conn.Write([]byte(userInput))
    }
}
