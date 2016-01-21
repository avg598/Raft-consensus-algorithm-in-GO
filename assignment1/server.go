package main

import (
	"fmt"
	"net"
	"strings"
	"bufio"
	"log"
	//"bytes"
	"strconv"
	"math/rand"
	"io"
	"time"
    "sync"
    //"sync/atomic"
)

type filestore struct {
	name string
	numbytes int
	version int64
	exptime int64
	contents []byte
}

var (
	m = make(map[string]filestore)

	lock sync.RWMutex
)

const PORT = ":8080"




/**
*	Command : write <filename> <numbytes> [<exptime>]\r\n<content bytes>\r\n
*	Returns : OK <version>\r\n
*/
func handleWrite(command []string, conn net.Conn, reader *bufio.Reader){

	// Invalid number of arguments
	if len(command) != 3 && len(command) != 4 {
		log.Printf("[SERVER]:[ERR]:invalid command arguments")
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	filename := strings.TrimSpace(command[1])
	numbytes, err := strconv.ParseUint(command[2], 10, 64)
	if err != nil {
		log.Printf("[SERVER]:[ERR]:unable to parse <numbytes> field: %v",command[2])
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	buff := make([]byte, numbytes+2)
	log.Printf("[SERVER]:[INFO]:Going to read %v bytes",numbytes+2)
	n, err := io.ReadFull(reader, buff)
	log.Printf("[SERVER]:[INFO]:Content received:%v",string(buff[:n]))
	// If unable to read numbytes bytes or if "\r\n" is not at the end of the buff
	if err != nil || (buff[n-2]!='\r' || buff[n-1]!='\n'){
		log.Printf("[SERVER]:[ERR]:insufficient data received or data doesn't end with \\r\\n")
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}


	rand.Seed(time.Now().Unix())
	version := rand.Int63()
	var exptime uint64 = 0
	if len(command) == 4 {
		exptime, err = strconv.ParseUint(command[3],0,64)
		if err != nil {
			log.Printf("[SERVER]:[ERR]:unable to parse <exptime> field: %v",command[3])
			conn.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}
	}

	log.Printf("[SERVER]:[INFO]:Writing to file %v",filename)

	// TODO:: Trim last "\r\n" from buff

	lock.Lock()
	m[filename]=filestore{filename,int(numbytes),version,int64(exptime),buff}		
	lock.Unlock()
	successMsg:="OK "+strconv.FormatInt(version,10)+"\r\n"	
	conn.Write([]byte(successMsg))
	
	return
}

/**
*	Command : read <filename>\r\n
* 	Returns : CONTENTS <version> <numbytes> <exptime>\r\n<content bytes>\r\n
*/
func handleRead(cmd []string, conn net.Conn){
	if len(cmd) != 2 {
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	filename:= cmd[1]

	lock.RLock()
	file, ok := m[filename]
	lock.RUnlock()
	if  ok {
		reply := fmt.Sprintf("CONTENTS %v %v %v\r\n%s\r\n", file.version, file.numbytes, file.exptime, file.contents)
		fmt.Fprintf(conn, reply)
		log.Printf("[SERVER]:[INFO]:Reply:%v", reply)
	} else {
		log.Printf("[SERVER]:[ERR]:file not found: %v",filename)
		conn.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
	}
	return
}

/**
*	Command : delete <filename>\r\n
*	Returns : OK\r\n  		 
*/
func handleDelete(cmd []string, conn net.Conn){

	// Invalid number of arguments
	if len(cmd) != 2 {
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	filename:= cmd[1]

	lock.Lock()
	_, ok := m[filename]
	if ok {
		delete(m, filename)
		lock.Unlock()
		log.Printf("[SERVER]:[INFO]:File deleted %v",filename)
		fmt.Fprintf(conn, "OK\r\n")   			
	} else {
		lock.Unlock()
		conn.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
	}
	return
}

/**
*	Command : cas <filename> <version> <numbytes> [<exptime>]\r\n<content bytes>\r\n
*	Returns : OK <version>\r\n
*/
func handleCas(cmd []string, conn net.Conn,  reader *bufio.Reader){

	// Invalid number of arguments
	if len(cmd) != 4 && len(cmd) != 5 {
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	filename:= cmd[1]
	//numbytes, err := strconv.Atoi(cmd[3])	
	numbytes, err := strconv.ParseUint(cmd[3], 10, 64)
	if err != nil {
		log.Printf("[SERVER]:[ERR]:Unable to parse <numbytes> field")
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	// parse version as number
	version, err := strconv.ParseInt(cmd[2], 10, 64)
	//ver, err := strconv.Atoi(cmd[2]) 
	if err != nil {
		log.Printf("[SERVER]:[ERR]:Unable to parse <version> field")
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	var exptime int64 = -1
	if len(cmd) == 5 {
		// Parse <exptime>
		exptime,err = strconv.ParseInt(cmd[4],0,64)
		if err != nil {
			log.Printf("[SERVER]:[ERR]:Unable to parse <exptime> field")
			conn.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}
	}

	buff := make([]byte, numbytes+2)
	log.Printf("[SERVER]:[INFO]:Going to read %v bytes",numbytes+2)
	n, err := io.ReadFull(reader, buff)
	log.Printf("[SERVER]:[INFO]:Content received:%v",string(buff[:n]))
	// If unable to read numbytes bytes or if "\r\n" is not at the end of the buff
	if err != nil || (buff[n-2]!='\r' || buff[n-1]!='\n'){
		log.Printf("[SERVER]:[ERR]:insufficient data received or data doesn't end with \\r\\n")
		conn.Write([]byte("ERR_CMD_ERR\r\n"))
		return
	}

	lock.Lock()
	if _, ok := m[filename]; ok {
		if m[filename].version == version {
			rand.Seed(version)
			newVersion:= rand.Int63()
			m[filename]=filestore{filename,int(numbytes),newVersion,exptime,buff}
			lock.Unlock()
			fmt.Fprintf(conn, "OK %v\r\n", newVersion)					
		} else {
			lock.Unlock()
			log.Printf("[SERVER]:[ERR]:version missmatch field")
			fmt.Fprintf(conn, "ERR_VERSION\r\n")
		}
	} else {
		lock.Unlock()
		log.Printf("[SERVER]:[ERR]:file not found")
		conn.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
	}
	return
}

func serverMain() {
	log.Printf("[SERVER]:[INFO]:Launching server...")
	listener,err := net.Listen("tcp", PORT)   

	if err != nil {
		log.Fatalf("[SERVER]:[ERR]:Error in listener: %v", err.Error())		
	}

	log.Printf("[SERVER]:[INFO]:Server started on %v", PORT)
	// infinite loop until ctr-c or exit
	for {
		conn,err := listener.Accept()

		if err != nil {
			log.Printf("[SERVER]:[ERR]:Error in accepting a connection: %v", err.Error())
		} else {
			go processClient(conn)
		}
	}
}


func processClient(conn net.Conn) {

	reader := bufio.NewReader(conn)
	for{
		line, isPrefix, err := reader.ReadLine()	// Read until \n or \r\n
		if err != nil || isPrefix {
			log.Printf("[SERVER]:[ERR]:Error in reading from client: %v", err.Error())
			errMsg := "ERR_CMD_ERR\r\n"
			conn.Write([]byte(errMsg))
			return
		}

		command := strings.Fields(string(line))
		if len(command) == 0 {
			log.Printf("[SERVER]:[ERR]:null command string received")
			conn.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}

		log.Printf("[SERVER]:[INFO]:Command : %v",strings.TrimSpace(string(line)))
		switch command[0] {
			case "read":
				handleRead(command, conn)

			case "write":
				handleWrite(command, conn, reader)

			case "cas":
				handleCas(command, conn, reader)

			case "delete":
				handleDelete(command, conn)

			default:
				log.Printf("[SERVER]:[ERR]:invalid command: %v",command[0])
				conn.Write([]byte("ERR_CMD_ERR\r\n"))
		}
		log.Printf("[SERVER]:[INFO]:Served : %v",strings.TrimSpace(string(line)))
	}
}


func main() {
	serverMain()
}
