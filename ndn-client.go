/*
 * NDN-Client, a NDN swiss-knife.
 * Copyright (C) 2016  Antonio Cardace, Davide Aguiari.
 *
 * NDN-Client is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 *
 * Wavetrack is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 */

package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	broadcastAddress        = "255.255.255.255"
	ndnPort                 = ":8888"
	ndnPacketTypeInterest   = 1
	ndnPacketTypeData       = 2
	ndnInterestPacketHeader = 7
	ndnDataPacketHeader     = 7
	maxUDPPacketSize        = 65507
)

func hexDump(buf []byte, n int) {
	fmt.Println("HEXDUMP:")
	for index := 0; index < n; index++ {
		fmt.Printf("0x%X ", buf[index])
	}
	fmt.Println()
}

func dumpInterestPacket(packet []byte) {
	buf := bytes.NewBuffer(packet)
	var packetType uint8
	var nonce, ipAddr uint32
	var nameLength uint16
	packetLength := len(buf.Bytes())

	if *usingGalileo {
		binary.Read(buf, binary.BigEndian, &ipAddr)
	}
	binary.Read(buf, binary.BigEndian, &packetType)
	if packetType == ndnPacketTypeInterest {
		binary.Read(buf, binary.BigEndian, &nonce)
		binary.Read(buf, binary.BigEndian, &nameLength)
		name := make([]byte, nameLength)
		binary.Read(buf, binary.BigEndian, &name)

		fmt.Printf("Interest Packet of Length %d\n", packetLength)
		fmt.Printf("\tNonce: 0x%X\n", nonce)
		fmt.Printf("\tNameLength: %d\n", nameLength)
		fmt.Printf("\tName: %s\n", name)
	} else {
		fmt.Println("Invalid or corrupted Interest Packet")
	}
}

func dumpDataPacket(packet []byte) {
	buf := bytes.NewBuffer(packet)
	var packetType uint8
	var contentLength, ipAddr uint32
	var nameLength uint16

	if *usingGalileo {
		binary.Read(buf, binary.BigEndian, &ipAddr)
	}
	binary.Read(buf, binary.BigEndian, &packetType)
	if packetType == ndnPacketTypeData {
		binary.Read(buf, binary.BigEndian, &nameLength)
		binary.Read(buf, binary.BigEndian, &contentLength)
		name := make([]byte, nameLength)
		binary.Read(buf, binary.BigEndian, &name)
		content := make([]byte, contentLength)
		binary.Read(buf, binary.BigEndian, &content)

		fmt.Println("Received Data Packet")
		fmt.Printf("\tNameLength: %d\n", nameLength)
		fmt.Printf("\tContentLength: %d\n", contentLength)
		fmt.Printf("\tName: %s\n", name)
		fmt.Printf("\tContent: %s\n", content)
	} else {
		fmt.Println("Invalid or corrupted Data Packet")
	}
}

func sendInterestPacket(conn net.Conn, interest string) (n int) {
	buf := new(bytes.Buffer)

	// randomize nonce field
	rand.Seed(time.Now().UnixNano())
	/* prepare the UDP packet */
	if *usingGalileo {
		ipAddr := []uint8(net.ParseIP(
			strings.Split(conn.LocalAddr().String(), ":")[0]).To4())
		binary.Write(buf, binary.BigEndian, ipAddr[0])
		binary.Write(buf, binary.BigEndian, ipAddr[1])
		binary.Write(buf, binary.BigEndian, ipAddr[2])
		binary.Write(buf, binary.BigEndian, ipAddr[3])
	}
	binary.Write(buf, binary.BigEndian, byte(ndnPacketTypeInterest))
	binary.Write(buf, binary.BigEndian, rand.Uint32())
	binary.Write(buf, binary.BigEndian, uint16(len(interest)))
	binary.Write(buf, binary.BigEndian, []byte(interest))
	/* send the UDP packet */
	n, err := conn.Write(buf.Bytes())
	checkError(err)

	// dump packet if requested
	if *dumpInterest {
		if *hexD {
			hexDump(buf.Bytes(), buf.Len())
		} else {
			dumpInterestPacket(buf.Bytes())
		}
	}
	return n
}

func sendDataPacket(conn net.Conn, interest string) (n int) {
	buf := new(bytes.Buffer)
	/* prepare the UDP packet */
	if *usingGalileo {
		ipAddr := []uint8(net.ParseIP(
			strings.Split(conn.LocalAddr().String(), ":")[0]).To4())
		binary.Write(buf, binary.BigEndian, ipAddr[0])
		binary.Write(buf, binary.BigEndian, ipAddr[1])
		binary.Write(buf, binary.BigEndian, ipAddr[2])
		binary.Write(buf, binary.BigEndian, ipAddr[3])
	}
	binary.Write(buf, binary.BigEndian, byte(ndnPacketTypeData))
	binary.Write(buf, binary.BigEndian, uint16(len(interest)))
	binary.Write(buf, binary.BigEndian, uint32(len(*content)))
	binary.Write(buf, binary.BigEndian, []byte(interest))
	binary.Write(buf, binary.BigEndian, []byte(*content))
	/* send the UDP packet */
	n, err := conn.Write(buf.Bytes())
	checkError(err)

	// dump packet if requested
	if *dumpData {
		if *hexD {
			hexDump(buf.Bytes(), buf.Len())
		} else {
			dumpDataPacket(buf.Bytes())
		}
	}
	return n
}

func recvDataPacket(conn *net.UDPConn, wg *sync.WaitGroup) {
	var readBytes int
	content := ""
	//make room for the packet
	recvBuffer := make([]byte, maxUDPPacketSize)
	for content == "" {
		/* wait for data packet */
		n, _, err := conn.ReadFromUDP(recvBuffer)
		readBytes = n
		checkError(err)
		content = parseDataContent(recvBuffer)
	}
	// dump packet if requested
	if *dumpData {
		if *hexD {
			hexDump(recvBuffer, readBytes)
		} else {
			dumpDataPacket(recvBuffer)
		}
	}
	fmt.Println(content)
	wg.Done()
}

func parseDataContent(buf []byte) string {
	dataBuffer := bytes.NewBuffer(buf)
	var contentLength uint32
	var packetType uint8
	var nameLength uint16
	var ipAddr uint32 //only for Galileo

	if *usingGalileo {
		binary.Read(dataBuffer, binary.BigEndian, &ipAddr)
	}
	binary.Read(dataBuffer, binary.BigEndian, &packetType)
	if packetType == ndnPacketTypeData {
		binary.Read(dataBuffer, binary.BigEndian, &nameLength)
		binary.Read(dataBuffer, binary.BigEndian, &contentLength)
		name := make([]byte, nameLength)
		binary.Read(dataBuffer, binary.BigEndian, &name)
		content := make([]byte, contentLength)
		binary.Read(dataBuffer, binary.BigEndian, &content)
		return string(content)
	}
	return ""
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

// command line args
var ndnGateway = flag.String("gw", "", "IP Address of the NDN Gateway")
var usingGalileo = flag.Bool("intel", false, "if using Intel Galilo")
var hexD = flag.Bool("x", false, "Hex dump")
var dumpInterest = flag.Bool("di", false, "Dump sent Interest packet")
var dumpData = flag.Bool("dd", false, "Dump received Data packet")
var sendDataPkt = flag.Bool("sd", false, "Send a Data packet")
var content = flag.String("c", "", "content of the Data packet")
var sendOnly = flag.Bool("nl", false, "Do not wait for a response packet")

func main() {
	interest := ""

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Printf("Usage: %s [OPTION]... INTEREST\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		return
	}
	// if there's no NDN gateway use broadcast Address
	if *ndnGateway == "" {
		*ndnGateway = broadcastAddress
	}
	//interest is the last element on the cmdline
	interest = os.Args[len(os.Args)-1]
	/* Init NDN Client socket */
	client, err := net.Dial("udp4", *ndnGateway+ndnPort)
	checkError(err)
	defer client.Close()

	if *sendDataPkt {
		sendDataPacket(client, interest)
	} else {
		var wg sync.WaitGroup
		/*Init NDN Server socket */
		if !*sendOnly {
			wg.Add(1)
			serverAddr, err := net.ResolveUDPAddr("udp4", ndnPort)
			checkError(err)
			server, err := net.ListenUDP("udp", serverAddr)
			checkError(err)
			defer server.Close()
			go recvDataPacket(server, &wg)
		}

		sendInterestPacket(client, interest)
		wg.Wait()
	}
}
