package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"reflect"
	"strconv"

	"github.com/jackpal/bencode-go"
	"github.com/yinfredyue/bittorrent-go/torrent"
)

type Client struct {
	torrent torrent.Torrent
}

func loadPeers(t torrent.Torrent) ([]peer, error) {
	// send GET request to tracker
	infoHash, err := t.InfoHash()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", t.Tracker, nil)
	if err != nil {
		return nil, err
	}

	params := req.URL.Query()
	params.Add("info_hash", string(infoHash))
	params.Add("peer_id", string(NewPeerId()))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", strconv.Itoa(t.Info.Length))
	params.Add("compact", "1")
	req.URL.RawQuery = params.Encode()

	cli := http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decodedObj, err := bencode.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	decodedDict, ok := decodedObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("fail to decode tracker response")
	}

	var peers []peer
	switch peersRaw := decodedDict["peers"].(type) {
	case string:
		// Compact
		// Each peer is represented by 6 bytes.
		// First 4 bytes is IP, where each byte is a number in the IP.
		// Last 2 bytes is port, in big-endian order.
		for i := 0; i < len(peersRaw); i += 6 {
			addr, ok := netip.AddrFromSlice([]byte(peersRaw)[i : i+4])
			if !ok {
				return nil, fmt.Errorf("fail to parse peer addr")
			}
			port := binary.BigEndian.Uint16([]byte(peersRaw[i+4 : i+6]))
			addrPort := netip.AddrPortFrom(addr, port)
			peer := peer{addrPort: addrPort}
			peers = append(peers, peer)
		}
	case [](interface{}):
		// Not compact
		// Each peer is represented as a dict.
		for _, peerRaw := range peersRaw {
			peerRawDict := peerRaw.(map[string]interface{})
			ipStr := peerRawDict["ip"].(string)
			addr, err := netip.ParseAddr(ipStr)
			if err != nil {
				return nil, err
			}
			port := peerRawDict["port"].(int64)
			addrPort := netip.AddrPortFrom(addr, uint16(port))
			peer := peer{addrPort: addrPort}
			peers = append(peers, peer)
		}
	default:
		log.Fatalf("Unexpected case: %v", reflect.TypeOf(peersRaw))
	}

	return peers, nil
}

func NewClient(torrent torrent.Torrent) Client {
	return Client{torrent: torrent}
}

func (cli *Client) ConnectToPeers() error {
	peers, err := loadPeers(cli.torrent)
	if err != nil {
		return err
	}

	// TODO: connect to peers!
	for _, p := range peers {
		fmt.Printf("%v\n", p.addrPort)
	}

	return nil
}

func Download() error {
	return nil
}
