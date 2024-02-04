package seeder

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"torrent-dsp/standard"
	"torrent-dsp/tools"
	"torrent-dsp/entities"
)

// Handle peer requests for pieces
func SeederMain() {
	// start a server listening on port 6881
	torrent, err := standard.ParseTorrentFile("./torrent-files/debian-11.6.0-amd64-netinst.iso.torrent")
	if err != nil {
		log.Fatal(err)
	}

    ln, err := net.Listen("tcp", ":6881")
	// why tcp?
	// TCP is a connection-oriented protocol, it requires the client and server to establish a connection before sending data
	// TCP is reliable, it guarantees that the data will be delivered in the same order in which it was sent
	// TCP is slower than UDP because it requires the establishment of a connection between the sender and the receiver

    if err != nil {
        log.Fatalf("Failed to listen: %s", err)
    }

    for {
        if conn, err := ln.Accept(); err == nil {
			fmt.Println("Accepted connection")
            go handleConnection(conn, torrent)
        }
    }
}


func handleConnection(conn net.Conn, torrent entities.Torrent) {
	conn.SetDeadline(time.Now().Add(tools.PIECE_UPLOAD_TIMEOUT))
	defer conn.Close()
	fmt.Println("Handling connection")
	fmt.Println()

	fmt.Println("------------------- Waiting for handshake -------------------")
	_, err := ReceiveHandShake(conn)
	fmt.Println("Received handshake")
	fmt.Println()
	if err != nil {
		fmt.Println("Error receiving handshake")
		return
	}

	// TODO: change byte to actual bit field of the data
	if err != nil {
		log.Fatal(err)
	}

	// send the handshake request
	SendHandShake(conn, torrent)
	fmt.Println("------------------- Sending bit field -------------------")
	SendBitField(conn)
	fmt.Println("------------------- Sent bit field -------------------")
	fmt.Println()
	
	// listen to unchoke message
	ReceiveUnchoke(conn)
	fmt.Println("Received unchoke message")
	// listen to interested message
	ReceiveInterested(conn)
	fmt.Println("Received interested message")
	SendUnchoke(conn)

	// listen to other request messages
	for {
		requestMsg, err := ReceiveRequest(conn)
		if err != nil {
			log.Fatal("found an error while receiving a request seeder")
			return
		}
		go handleRequest(*requestMsg, conn)
	}
}


func ReceiveHandShake(conn net.Conn) (*entities.HandShake, error) {
	// read the handshake response
	// handshake response is 68 bytes long <length of Protocol> + <Protocol> + <Reserved> + <InfoHash> + <PeerID>
	// TODO: increase time out
	buffer := make([]byte, 68)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading handshake response")
		return &entities.HandShake{}, err
	}

	// deserialize the handshake response
	handShake, err := entities.DeserializeHandShake(buffer)
	if err != nil {
		fmt.Println("Error deserializing handshake response")
		return &entities.HandShake{}, err
	}

	fmt.Println("Handshake received successfully")
	return handShake, nil
}


func SendHandShake(conn net.Conn, torrent entities.Torrent) error {
	// convert the client id to a byte array
	clientIDByte := [20]byte{}
	copy(clientIDByte[:], []byte(tools.CLIENT_ID))

	// create a handshake request
	handshakeRequest := entities.HandShake{
		Pstr:     "BitTorrent protocol",
		InfoHash: torrent.InfoHash,
		PeerID:   clientIDByte,
	}

	// send the handshake request
	// serialize the handshake request
	buffer := handshakeRequest.Serialize()

	// send the handshake request
	_, err := conn.Write(buffer)
	if err != nil {
		fmt.Println("Error sending handshake request SEEDER")
		return err
	}
	return nil
}


func SendBitField(conn net.Conn) error {
	// send the bitfield
	bitField := make([]byte, 255)
	for i := 0; i < len(bitField); i++ {
		bitField[i] = 255
	}

	msg := entities.Message{MessageID: tools.BIT_FIELD, Payload: bitField}
	_, err := conn.Write(msg.Serialize())
	if err != nil {
		return err
	}
	return nil
}


func ReceiveUnchoke(conn net.Conn) error {
	buffer := make([]byte, 5)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading unchoke message")
		return err
	}

	return nil
}


func ReceiveInterested(conn net.Conn) error {
	buffer := make([]byte, 5)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading interested message")
		return err
	}

	return nil
}


func ReceiveRequest(conn net.Conn) (*entities.Message, error) {
	// buffer := make([]byte, 17)
	conn.SetDeadline(time.Now().Add(tools.PIECE_UPLOAD_TIMEOUT))
    defer conn.SetDeadline(time.Time{})
	// time.Sleep(1 * time.Second)

	requestMsg, err := entities.DeserializeMessage(conn)
	if err != nil {
		fmt.Println("Error reading request message")
		return &entities.Message{}, err
	}

	if err != nil {
		fmt.Println("Error opening file")
		return &entities.Message{}, err
	}

	return requestMsg, nil
	
}


func handleRequest(requestMsg entities.Message, conn net.Conn) error {

	file, err := os.Open("downloads/debian-11.6.0-amd64-netinst.iso")
	defer file.Close()

	// parse payload
	if requestMsg.MessageID != tools.REQUEST {
		fmt.Println("Error: received message is not a request")
		return nil
	}
	index, begin, size, blockStart := ParseRequestPayload(requestMsg.Payload)

	// read the piece from the file
	piece := make([]byte, int64(size))
	_, err = file.ReadAt(piece, int64(begin))
	if err != nil {
		fmt.Println("Error reading piece from file")
		return err
	}
	fmt.Println("Piece read successfully")
	// send the piece
	err = SendPiece(conn, piece, index, blockStart)
	if err != nil {
		fmt.Println("Error sending piece")
		return err
	}

	// fmt.Println("Piece sent successfully")

	return nil
}


func SendPiece(conn net.Conn, piece []byte, index int, blockStart int) error {
	payload := make([]byte, 8+len(piece))
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(blockStart))
	copy(payload[8:], piece[:])

	msg := entities.Message{MessageID: tools.PIECE, Payload: payload}
	_, err := conn.Write(msg.Serialize())
	if err != nil {
		fmt.Println("Error sending piece")
		return err
	}

	return nil
}


func ParseRequestPayload(payload []byte) (int, int, int, int) {
	index := int(binary.BigEndian.Uint32(payload[0:4]))
	blockStart := int(binary.BigEndian.Uint32(payload[4:8]))
	blockSize := int(binary.BigEndian.Uint32(payload[8:12]))
	pieceSize := 262144
	fileSize := 471859200
	begin := index * pieceSize + blockStart
	// end := begin + blockSize
	end := tools.CalcMin(fileSize, begin + blockSize)

	if blockSize == 0 {
		fmt.Println("Error: block size is 0")
	}
	
	if end > fileSize {
		end = fileSize
		blockSize = end - begin
	}

	return index, begin, blockSize, blockStart
}


func SendUnchoke(conn net.Conn) {
	msg := entities.Message{MessageID: tools.UN_CHOKE, Payload: []byte{}}
	_, err := conn.Write(msg.Serialize())
	if err != nil {
		log.Fatalf("Error sending unchoke message to peer: %s", err)
	}

}