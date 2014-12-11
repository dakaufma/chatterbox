// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	"log"
	"os"
	"time"
)

const (
	MAX_PREKEYS = 100 //TODO make this configurable
	MIN_PREKEYS = 50
)

func Start(rootDir string) (*Config, error) {
	conf := LoadConfig(&Config{
		RootDir:    rootDir,
		Now:        time.Now,
		TempPrefix: "daemon",
	})

	// ensure that we have a correct directory structure
	// including a correctly-populated outbox
	if err := InitFs(conf); err != nil {
		return nil, err
	}
	inBuf := make([]byte, util.MAX_MESSAGE_SIZE)
	outBuf := make([]byte, util.MAX_MESSAGE_SIZE)

	ourDename := conf.Dename

	denameClient, err := client.NewClient(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	conf.denameClient = denameClient
	conf.ourDename = ourDename
	conf.inBuf = inBuf
	conf.outBuf = outBuf

	return conf, nil
}

func (conf *Config) encryptFirstMessage(msg []byte, connToServer *util.ConnectionToServer, theirPk *[32]byte, encMsg []byte, theirDename []byte) error {
	ourSkAuth := (*[32]byte)(&conf.MessageAuthSecretKey)

	theirInBuf := make([]byte, util.MAX_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(theirDename, conf.denameClient)
	if err != nil {
		return err
	}
	theirKey, err := util.GetKey(theirConn, theirInBuf, conf.outBuf, theirPk, theirDename, conf.denameClient)
	if err != nil {
		return err
	}
	encMsg, ratch, err := util.EncryptAuthFirst(theirDename, msg, ourSkAuth, theirKey, conf.denameClient)
	StoreRatchet(conf, string(theirDename), ratch)
	if err != nil {
		return err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
	if err != nil {
		return err
	}
	return nil
}

func (conf *Config) encryptMessage(msg []byte, connToServer *util.ConnectionToServer, theirPk *[32]byte, encMsg []byte, theirDename []byte) error {
	msgRatch, err := LoadRatchet(conf, string(theirDename))
	if err != nil {
		return err
	}

	theirInBuf := make([]byte, util.MAX_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(theirDename, conf.denameClient)
	if err != nil {
		return err
	}

	encMsg, ratch, err := util.EncryptAuth(theirDename, msg, msgRatch)
	StoreRatchet(conf, string(theirDename), ratch)

	if err != nil {
		return err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
	if err != nil {
		return err
	}
	return nil
}

func (conf *Config) decryptFirstMessage(envelope []byte) ([]byte, error) {
	pkList, skList, err := LoadPrekeys(conf)
	if err != nil {
		return nil, err
	}
	skAuth := (*[32]byte)(&conf.MessageAuthSecretKey)
	ratch, msg, index, err := util.DecryptAuthFirst(envelope, skList, skAuth, conf.denameClient)
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, err
	}
	newPrekeys := append(skList[:index], skList[index+1:]...)
	StorePrekeys(conf, newPrekeys)
	StoreRatchet(conf, string(message.Dename), ratch)

	return msg, nil
}

func (conf *Config) decryptMessage(envelope []byte) ([]byte, error) {
	ratchets, err := AllRatchets(conf)
	if err != nil {
		return nil, err
	}
	var ratch *ratchet.Ratchet
	var msg []byte
	for _, msgRatch := range ratchets {
		ratch, msg, err = util.DecryptAuth(envelope, msgRatch)
		if err == nil {
			break
		}
	}
	if msg == nil {
		return nil, errors.New("Invalid message received.")
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, err
	}
	StoreRatchet(conf, string(message.Dename), ratch)
	return msg, nil
}

func Run(conf *Config, shutdown <-chan struct{}) error {

	initFn := func(path string, f os.FileInfo, err error) error {
		log.Printf("init path: %s\n", path)
		return err
	}

	profile, err := LoadPublicProfile(conf)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = WatchDir(watcher, conf.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	ourConn, err := util.CreateHomeServerConn(
		conf.ServerAddressTCP, (*[32]byte)(&profile.UserIDAtServer),
		(*[32]byte)(&conf.TransportSecretKeyForServer),
		(*[32]byte)(&conf.ServerTransportPK))
	if err != nil {
		return err
	}

	notifies := make(chan []byte)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		Buf:          conf.inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
	}

	go connToServer.ReceiveMessages()

	// load prekeys and ensure that we have enough of them
	prekeyPublics, prekeySecrets, err := LoadPrekeys(conf)
	if err != nil {
		return err
	}
	numKeys, err := util.GetNumKeys(ourConn, connToServer, conf.outBuf)
	if err != nil {
		return err
	}
	if numKeys < MIN_PREKEYS {
		newPublicPrekeys, newSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS - int(numKeys))
		prekeySecrets = append(prekeys, newSecretPrekeys...)
		prekeyPublics = append(prekeyPublics, newPublicPrekeys...)
		StorePrekeys(conf, prekeyPublics, prekeySecrets)
		var signingKey [64]byte
		copy(signingKey[:], conf.KeySigningSecretKey[:64])
		err = util.UploadKeys(ourConn, connToServer, conf.outBuf, util.SignKeys(newPublicPrekeys, &signingKey))
		if err != nil {
			return err // TODO handle this nicely
		}
	}

	for {
		select {
		case <-shutdown:
			return nil
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			if !(ev.IsDelete() || ev.IsRename()) {
				err = WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					return err
				}
				if true { //TODO: Fill in something's changed in the outbox
					if true { //TODO: First message in this conversation
						msg := []byte("Message") //TODO: msg is metadata + conversation

						var theirPk *[32]byte          //TODO: Load from file
						var encMsg, theirDename []byte //TODO: Load from file

						if err := conf.encryptFirstMessage(msg, connToServer, theirPk, encMsg, theirDename); err != nil {
							return err
						}
					} else { //TODO: Not-first message in this conversation
						msg := []byte("Message") //TODO: msg is metadata + conversation

						var theirPk *[32]byte          //TODO: Load from file
						var encMsg, theirDename []byte //TODO: Load from file

						if err := conf.encryptMessage(msg, connToServer, theirPk, encMsg, theirDename); err != nil {
							return err
						}
					}
				}
			}
		case envelope := <-connToServer.ReadEnvelope:
			if true { //TODO: is the first message we're receiving from the person
				msg, err := conf.decryptFirstMessage(envelope)
				if err != nil {
					return err
				}

				msg = msg //Take out
				//TODO: Take out metadata + converastion from msg, Store the decrypted message
			} else {
				msg, err := conf.decryptMessage(envelope)
				if err != nil {
					return err
				}

				msg = msg //Take out
				//TODO: Take out metadata + conversation from msg, then store

			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
