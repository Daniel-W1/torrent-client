package standard

import (
	"fmt"
	"os"
	bencode "github.com/zeebo/bencode"

	"torrent-dsp/entities"
)

// parse a torrent file and return a Torrent struct
func ParseTorrentFile(filename string) (entities.Torrent, error) {
	// open the file
	file, err := os.Open(filename)
	if err != nil {
		return entities.Torrent{}, err
	}
	defer file.Close()

	// decode the file
	var torrent = entities.Torrent{}
	err = bencode.NewDecoder(file).Decode(&torrent)
	if err != nil {
		fmt.Println("Encountered error while decoding")
		return entities.Torrent{}, err
	}

	// generate the info hash
	torrent.GenerateInfoHash()
	fmt.Println("Torrent file parsed successfully")
	return torrent, nil
}