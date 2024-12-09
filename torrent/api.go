package torrent

type Client interface {
	AddTorrent(torrentFile []byte, downloadPath string) error
}
