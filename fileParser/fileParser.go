package fileParser

import (
	"BittorrentClient/download"
	"BittorrentClient/peers"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/jackpal/bencode-go"
)

// flat bencode struct
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

type bencodeTracker struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func ParseFile(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	benTorr := bencodeTorrent{}
	err = bencode.Unmarshal(file, &benTorr)
	if err != nil {
		return TorrentFile{}, err
	}
	return benTorr.ToTorrentFile()
}

func (benTorr bencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	var infoBuffer bytes.Buffer

	err := bencode.Marshal(&infoBuffer, benTorr.Info)
	if err != nil {
		return TorrentFile{}, err
	}

	infoHash := sha1.Sum(infoBuffer.Bytes())

	buf := []byte(benTorr.Info.Pieces)
	numHashes := len(buf) / 20
	pieceHashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(pieceHashes[i][:], buf[i*20:(i+1)*20])
	}

	result := TorrentFile{
		Announce:    benTorr.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: benTorr.Info.PieceLength,
		Length:      benTorr.Info.Length,
		Name:        benTorr.Info.Name,
	}
	return result, nil
}

func (t *TorrentFile) BuildTrackerUrl(peerID [20]byte, port uint16) (string, error) {
	finalUrl, err := url.Parse(t.Announce)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}

	finalUrl.RawQuery = params.Encode()
	return finalUrl.String(), nil

}

func (t *TorrentFile) RequestPeers(peerID [20]byte, port uint16) ([]peers.Peer, error) {
	trackerUrl, err := t.BuildTrackerUrl(peerID, port)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(trackerUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	trackerInfo := bencodeTracker{}
	err = bencode.Unmarshal(resp.Body, &trackerInfo)
	if err != nil {
		return nil, err
	}

	return peers.UnMarshal([]byte(trackerInfo.Peers))
}

func (t *TorrentFile) DownloadFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.RequestPeers(peerID, 6881)
	if err != nil {
		return err
	}

	torrent := download.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}

	buf, err := torrent.Download()
	if err != nil {
		return err
	}
	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}
