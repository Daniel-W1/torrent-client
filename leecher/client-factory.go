package leecher

import (
	"bytes"
	"net"
	"time"

	"torrent-dsp/entities"
	"torrent-dsp/tools"
)

func ClientFactory(peer entities.Peer, torrent entities.Torrent) (*entities.Client, error) {
	client, err := createClient(peer, torrent)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// to create a client we need to:
// 1. connect to the peer
// 2. shake hands with the peer
// 3. receive bit field message from the peer
func createClient(peer entities.Peer, torrent entities.Torrent) (*entities.Client, error) {
	conn, err := connectToPeer(peer, torrent)
	if err != nil {
		return nil, err
	}

	// shake hands with the peer
	err = ShakeHandWithPeer(torrent, peer, tools.CLIENT_ID, conn)
	if err != nil {
		return nil, err
	}

	// receive bit field message from the peer
	bitFieldMessage, err := ReceiveBitFieldMessage(conn)
	if err != nil {
		return &entities.Client{}, err
	}

	// create a new client
	client := &entities.Client{
		Peer:        peer,
		BitField:    bitFieldMessage.Payload,
		Conn:        conn,
		ChokedState: tools.CHOKE,
	}

	return client, nil
}

// connect to peer. If not possible then return error
func connectToPeer(peer entities.Peer, torrent entities.Torrent) (net.Conn, error) {

	conn, err := net.DialTimeout("tcp", peer.String(), tools.CONNECTION_TIMEOUT)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func ShakeHandWithPeer(torrent entities.Torrent, peer entities.Peer, clientID string, conn net.Conn) error {

	conn.SetDeadline(time.Now().Add(tools.CONNECTION_TIMEOUT))
	defer conn.SetDeadline(time.Time{})
	
	// convert the client id to a byte array
	clientIDByte := [20]byte{}
	copy(clientIDByte[:], []byte(clientID))

	// create a handshake request
	handshakeRequest := entities.HandShake{
		Pstr:     "BitTorrent protocol",
		InfoHash: torrent.InfoHash,
		PeerID:   clientIDByte,
	}

	// send the handshake request
	handshakeResponse, err := handshakeRequest.Send(conn)
	if err != nil {
		return err
	}

	// check that the infohash in the response matches the infohash of the torrent
	if !bytes.Equal(handshakeResponse.InfoHash[:], torrent.InfoHash[:]) {
		return err
	}

	// check that the peer id in the response is different from ours
	if bytes.Equal(handshakeResponse.PeerID[:], tools.ConvertStringToByteArray(tools.CLIENT_ID)[:]) {
		return err
	}

	return nil
}

func ReceiveBitFieldMessage(conn net.Conn) (*entities.Message, error) {
	conn.SetDeadline(time.Now().Add(tools.CONNECTION_TIMEOUT))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline


	// receive the bitField message
	// fmt.Println("Deserializing bit field message...")
	bitFieldMessageResponse, err := entities.DeserializeMessage(conn)
	if err != nil {
		// fmt.Println("Error receiving bit field message")
		return nil, err
	}

	// check that the message is a bit field message
	if bitFieldMessageResponse.MessageID != tools.BIT_FIELD {
		// fmt.Println("Expected bit field message")
		return &entities.Message{}, nil
		// log.Fatal("expected bit field message")
	}

	return bitFieldMessageResponse, nil
}
