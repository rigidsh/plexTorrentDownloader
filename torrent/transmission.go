package torrent

import (
	"context"
	"encoding/base64"
	"github.com/hekmon/transmissionrpc/v3"
	"net/url"
)

type TransmissionRemoteTorrent struct {
	client *transmissionrpc.Client
}

func NewTransmissionRemoteTorrent(endpoint *url.URL) (*TransmissionRemoteTorrent, error) {
	client, err := transmissionrpc.New(endpoint, nil)

	if err != nil {
		return nil, err
	}

	return &TransmissionRemoteTorrent{client}, nil
}

func (t *TransmissionRemoteTorrent) AddTorrent(torrentFile []byte, downloadPath string) error {
	metainfo := base64.StdEncoding.EncodeToString(torrentFile)

	_, err := t.client.TorrentAdd(context.TODO(), transmissionrpc.TorrentAddPayload{
		MetaInfo:    &metainfo,
		DownloadDir: &downloadPath,
	})
	if err != nil {
		return err
	}

	return nil
}
