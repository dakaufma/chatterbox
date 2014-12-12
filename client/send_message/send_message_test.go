package main

import (
	"fmt"
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"github.com/andres-erbsen/chatterbox/proto"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func handleError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func TestSpawnConversationInOutbox(t *testing.T) {
	// init the file system + configuration structure
	rootDir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(rootDir)
	handleError(err, t)

	conf := &daemon.Config{
		RootDir:    rootDir,
		Now:        func() time.Time { return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC) },
		TempPrefix: "some_ui",
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: []byte("user_dename"),
		},
	}

	err = daemon.InitFs(conf)
	handleError(err, t)

	subject := "test subject"
	recipients := [][]byte{[]byte("recipient_dename_b"), []byte("recipient_dename_a")}
	messages := [][]byte{[]byte("message1"), []byte("message2")}
	err = SpawnConversationInOutbox(conf, subject, recipients, messages)
	handleError(err, t)

	// check that a conversation exists in the outbox with the correct name
	outboxDir := conf.OutboxDir()
	expectedName := "2009-11-10T23:00:00Z-user_dename-recipient_dename_a-recipient_dename_b"
	_, err = os.Stat(filepath.Join(outboxDir, expectedName))
	handleError(err, t)

	// check that it has a valid metadata file
	metadataBytes, err := ioutil.ReadFile(filepath.Join(outboxDir, expectedName, daemon.MetadataFileName))
	handleError(err, t)
	metadataProto := new(proto.ConversationMetadata)
	err = metadataProto.Unmarshal(metadataBytes)
	handleError(err, t)

	// check that it has all message files; for now assume they have the correct contents
	files, err := ioutil.ReadDir(filepath.Join(outboxDir, expectedName))
	handleError(err, t)
	if len(files) != 3 { // metadata file + 2 messages
		t.Error(fmt.Sprintf("Wrong number of files %d in outgoing conversation; should be 3", len(files)))
	}

	// check that the temp directory has been cleaned up
	files, err = ioutil.ReadDir(conf.TmpDir())
	handleError(err, t)
	if len(files) > 0 {
		t.Error("tmp directory not cleaned up")
	}
}