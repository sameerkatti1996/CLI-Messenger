package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

//Connection details
const (
	connHost = "localhost"
	connPort = "8081"
	connType = "tcp"
)

//Data type to the user details
type user_detail struct {
	ip_port net.Addr
	conn    net.Conn
}

//Map of user to user details(ip:port, conn)
var ip_user_map = map[string]user_detail{}

//Mutex which will be used before map access.
//As there is concurrency, mutex is needed to avoid race conditions
var mapMutex = sync.RWMutex{}

func main() {
	fmt.Println("Starting " + connType + " server on " + connHost + ":" + connPort)

	//Start the server
	l, err := net.Listen(connType, connHost+":"+connPort)

	//If there are any errors
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	//Do not close listen till all connections(goroutines) are closed
	defer l.Close()

	for {
		//If there is any incoming connection, accept
		c, err := l.Accept()

		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}

		//Get IP Addr of and data(name, to) from the user
		ip := c.RemoteAddr()
		data := getData(c)

		//Unmarshal []byte to map
		m := make(map[string]string)
		err = json.Unmarshal(data, &m)

		if err != nil {
			fmt.Println("Error communicating with client")
			c.Close()
			continue
		}

		//Lock the mutex as we are about to do map operations
		mapMutex.Lock()

		//Create a new mapping with ip and conn object for incoming user
		ip_user_map[m["name"]] = user_detail{ip, c}

		//Check if the user with whom incoming session wants to connect to, exists
		to_detail, exists := ip_user_map[m["to"]]

		//Unlock mutex as map operations are done
		mapMutex.Unlock()

		//If the receiver exists, notify the sender & receiver
		if exists {
			//Send message to incoming session that receiver is online
			c.Write([]byte("online"))

			//Get connection data for the intended receiver
			//and send a message that receiver is online
			co := to_detail.conn
			co.Write([]byte("online"))

		} else {
			//Send a message that receiver is offline
			c.Write([]byte("offline"))
		}

		//Initialize goroutines for messages to be exchanged between those users
		go exchangeMessages(c, m["to"])
	}
}

//Helper function to get the data{name, to}
func getData(conn net.Conn) []byte {
	tmp := make([]byte, 8192)

	n, err := conn.Read(tmp)
	if err != nil {
		if err != io.EOF {
			fmt.Println("read error:", err)
			return []byte{}
		}
	}

	return tmp[:n]
}

//Function that facilitates message exchanges between users
//For the conn that is passed, the function read message from receiver
//and sends to user with connection conn
func exchangeMessages(conn net.Conn, user string) {
	//Loop for multiple message exchange
	for {

		//Wait till the receiver exixts
		for {

			mapMutex.Lock()
			_, exists := ip_user_map[user]
			mapMutex.Unlock()

			if exists {
				break
			}

		}

		mapMutex.Lock()
		user_conn := ip_user_map[user].conn
		mapMutex.Unlock()

		//Read any message from the other user
		buffer, err := bufio.NewReader(user_conn).ReadBytes('\n')

		if err != nil {
			//Close the connection
			conn.Close()

			//Delete the map entry which makes user offline
			mapMutex.Lock()
			delete(ip_user_map, user)
			mapMutex.Unlock()

			return
		}

		//send the received message
		conn.Write(buffer)

	}
}
