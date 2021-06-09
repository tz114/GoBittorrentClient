package peers

import (
	"encoding/binary"
	"net"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func UnMarshal(peersBin []byte) ([]Peer, error) {
	totalPeers := len(peersBin) / 6
	peers := make([]Peer, totalPeers)
	for i := 0; i < totalPeers; i++ {
		offset := i * 6
		peer := Peer{
			IP:   net.IP(peersBin[offset : offset+4]),
			Port: binary.BigEndian.Uint16(([]byte(peersBin[offset+4 : offset+6]))),
		}
		peers[i] = peer
	}
	return peers, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}
