package connection

import (
	"BittorrentClient/bitfield"
	"BittorrentClient/message"
	"BittorrentClient/peers"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield bitfield.Bitfield
	peer     peers.Peer
	infoHash [20]byte
	peerID   [20]byte
}

func recvBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := message.DeSerialize(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		err := fmt.Errorf("Expected bitfield but got %s", msg)
		return nil, err
	}
	if msg.ID != message.MsgBitfield {
		err := fmt.Errorf("expected bitfield but got ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

func Start(peer peers.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	handShake := Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
	_, err = handShake.DoHandshake(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	log.Print("handshake done")
	bf, err := recvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

func (h *Handshake) DoHandshake(conn net.Conn) (*Handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	_, err := conn.Write(h.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := ReadHandshake(conn)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(res.InfoHash[:], h.InfoHash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", res.InfoHash, h.InfoHash)
	}

	return res, nil
}

func (h *Handshake) Serialize() []byte {
	pstrlen := len(h.Pstr)
	bufLen := 49 + pstrlen
	buf := make([]byte, bufLen)
	buf[0] = byte(pstrlen)
	copy(buf[1:], h.Pstr)
	// Leave 8 reserved bytes
	copy(buf[1+pstrlen+8:], h.InfoHash[:])
	copy(buf[1+pstrlen+8+20:], h.PeerID[:])

	return buf
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}

func (c *Client) Read() (*message.Message, error) {
	msg, err := message.DeSerialize(c.Conn)
	return msg, err
}

func (c *Client) SendUnchoke() {
	msg := message.Message{
		ID: message.MsgUnchoke,
	}
	c.Conn.Write(msg.Serialize())
}

func (c *Client) SendInterested() {
	msg := message.Message{
		ID: message.MsgInterested,
	}
	c.Conn.Write(msg.Serialize())
}

func (c *Client) SendHave(index int) error {
	msg := message.FormatHave(index)
	log.Println("sendhave")
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Client) SendRequest(index, begin, length int) error {
	req := message.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}
