package main

import (
	"github.com/okx/okbchain/libs/tendermint/libs/log"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDownloadSnapshot(t *testing.T) {
	url := "https://download.redis.io/releases/redis-6.2.6.tar.gz"
	_, err := downloadSnapshot(url, "/tmp", log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
	assert.Nil(t, err)
}

func TestExtractTarGz(t *testing.T) {
	file := "/tmp/redis-6.2.6.tar.gz"
	err := extractTarGz(file, "/tmp")
	assert.Nil(t, err)
}
