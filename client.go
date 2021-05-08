package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"io"
)

//Connection details
const (
	connHost = "localhost"
	connPort = "8081"
	connType = "tcp"
)

func main() {
	//Start a TCP session with the server
	conn, err := net.Dial(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}

	//Get the user id by which client wants to be identified in the Messenger
	fmt.Print("Enter your name: ")
	name, err := bufio.NewReader(os.Stdin).ReadBytes('\n')

	if err != nil {
		fmt.Println("Error reading the input")
		return
	}

	//Get the user id of the user with whom the client wants to connect with
	fmt.Print("Whom do you want to talk to? ")
	to, err := bufio.NewReader(os.Stdin).ReadBytes('\n')

	if err != nil {
		fmt.Println("Error reading the message")
		return
	}

	//Create a map which would be sent to server
	var data = map[string]string{}
	data["name"] = string(name)
	data["to"] = string(to)

	//Marshal the map into a string
	val, _ := json.Marshal(data)

	//Send the data to the server
	conn.Write(val)

	//Read the response from the server
	tmp := make([]byte, 8192)

	n, err := conn.Read(tmp)
	if err != nil {
		if err != io.EOF {
			fmt.Println("read error:", err)
		}
	}

	//Response will be the status of the other user: online or offline
	response := string(tmp[:n])

	//If status is offline, wait till the user comes online
	//Client may abort if not ready to wait
	if response == "offline" {
		fmt.Println("waiting till the user comes online. Press Ctrl + C to exit.")
		for {
			tmp := make([]byte, 8192)

			n, err := conn.Read(tmp)
			if err != nil {
				if err != io.EOF {
					fmt.Println("read error", err)
					continue
				}
			}

			response := string(tmp[:n])

			if response == "online" {
				break
			}
		}
	}

	//Print the message to indicate the start of the Conversation
	fmt.Println("======================= Conversation =======================")

	//A wait group is initiated to wait till
	//following goroutines are done executing
	var wg sync.WaitGroup
  wg.Add(1)

	//The send and receive functions must be concurrent

	//Initiate a goroutine to send message
	go sendMessage(conn)

	//Inittiate a goroutine to receive message
	go receiveMessage(conn, &wg)

	//Wait till the WaitGroup count is zero before the main exits
	wg.Wait()
}

func sendMessage(conn net.Conn) {
	//Read the stdin for any input message to be sent
	for {
		buffer, err := bufio.NewReader(os.Stdin).ReadBytes('\n')

		if err != nil {
			fmt.Println("Error reading the message")
			return
		}

		//Send the message to the server which then redirects it to receiver
		conn.Write(buffer)
	}

}

func receiveMessage(conn net.Conn, wg *sync.WaitGroup) {
	//Read the connection for any incoming messages
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println("Unable to reach server. Timed out :(")
			conn.Close()
			wg.Done()
			return
		}

		//Print the messages received
		fmt.Print("\t\t\t\t\t", message)
	}
}
