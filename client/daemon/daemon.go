// Package daemon long-running client-side chatterbox functionality
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.google.com/p/go.exp/fsnotify"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/client/profilesyncd"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	dename "github.com/andres-erbsen/dename/protocol"
)

const (
	// How many prekeys should the daemon try to keep at the server?
	maxPrekeys  = 100 //TODO make this configurable
	minPrekeys  = 50
	daemonAppID = "daemon"
)

// Daemon encapsulates long-running client-side chatterbox functionality
type Daemon struct {
	persistence.Paths

	// Gets the current time
	Now func() time.Time

	proto.LocalAccountConfig

	foreignDenameClient  *client.Client
	timelessDenameClient *client.Client

	inBuf  []byte
	outBuf []byte

	stop chan struct{}
	wg   sync.WaitGroup
	psd  *profilesyncd.ProfileSyncd

	ourDenameLookup *dename.ClientReply
}

// New initializes a chatterbox daemon by loading condiguration from rootDir
func New(rootDir string) (*Daemon, error) {
	d := &Daemon{
		Paths: persistence.Paths{
			RootDir:     rootDir,
			Application: "daemon",
		},
		Now: time.Now,
	}
	if err := persistence.UnmarshalFromFile(d.configPath(), &d.LocalAccountConfig); err != nil {
		return nil, err
	}
	persistence.UnmarshalFromFile(d.ourDenameLookupReplyPath(), d.ourDenameLookup)

	// ensure that we have a correct directory structure
	// including a correctly-populated outbox
	if err := InitFs(d); err != nil {
		return nil, err
	}
	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	ourDenameClient, err := client.NewClient(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	// TODO: randomized per-connection TOR dialer
	d.foreignDenameClient, err = client.NewClient(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	timelessCfg := client.DefaultConfig
	timelessCfg.Freshness.Threshold = fmt.Sprintf("%dy", 365*100)
	d.timelessDenameClient, err = client.NewClient(&timelessCfg, nil, nil)
	if err != nil {
		return nil, err
	}
	d.inBuf = inBuf
	d.outBuf = outBuf
	d.psd, err = profilesyncd.New(ourDenameClient, 10*time.Minute, d.Dename, d.onOurDenameProfileDownload, nil)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Start activates the already initialized chatterbox daemon
func (d *Daemon) Start() {
	d.stop = make(chan struct{})
	d.psd.Start()
	if d.ourDenameLookup == nil {
		d.psd.Force()
	}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.run()
	}()
}

// Stop stops the daemon and returns when it has completely shut down
func (d *Daemon) Stop() {
	close(d.stop)
	d.psd.Stop()
	d.wg.Wait()
}

// run executes the main loop of the chatterbox daemon
func (d *Daemon) run() error {
	profile := new(proto.Profile)
	if err := persistence.UnmarshalFromFile(d.ourChatterboxProfilePath(), profile); err != nil {
		return err
	}

	ourConn, err := util.CreateHomeServerConn(
		d.ServerAddressTCP, (*[32]byte)(&profile.UserIDAtServer),
		(*[32]byte)(&d.TransportSecretKeyForServer),
		(*[32]byte)(&d.ServerTransportPK))
	if err != nil {
		return err
	}
	defer ourConn.Close()

	notifies := make(chan []byte)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		Buf:          d.inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
	}

	go connToServer.ReceiveMessages()

	// load prekeys and ensure that we have enough of them
	prekeyPublics, prekeySecrets, err := LoadPrekeys(d)
	if err != nil {
		return err
	}
	numKeys, err := util.GetNumKeys(ourConn, connToServer, d.outBuf)
	if err != nil {
		return err
	}
	if numKeys < minPrekeys {
		newPublicPrekeys, newSecretPrekeys, err := GeneratePrekeys(maxPrekeys - int(numKeys))
		prekeySecrets = append(prekeySecrets, newSecretPrekeys...)
		prekeyPublics = append(prekeyPublics, newPublicPrekeys...)
		if err = StorePrekeys(d, prekeyPublics, prekeySecrets); err != nil {
			return err
		}
		var signingKey [64]byte
		copy(signingKey[:], d.KeySigningSecretKey[:64])
		err = util.UploadKeys(ourConn, connToServer, d.outBuf, util.SignKeys(newPublicPrekeys, &signingKey))
		if err != nil {
			return err // TODO handle this nicely
		}
	}

	initFn := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return d.processOutboxDir(path)
		}
		return d.processOutboxDir(filepath.Dir(path))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = WatchDir(watcher, d.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	if err = util.EnablePush(ourConn, connToServer, d.outBuf); err != nil {
		return err
	}

	for {
		select {
		case <-d.stop:
			return nil
		case ev := <-watcher.Event:
			fmt.Printf("event: %v\n", ev)
			// event in the directory structure; watch any new directories
			if _, err = os.Stat(ev.Name); err == nil {
				err = WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					return err
				}

				d.processOutboxDir(ev.Name)
			}
		case envelope := <-connToServer.ReadEnvelope:
			// assume it's the first message we're receiving from the person; try to decrypt
			message, ratch, index, err := d.decryptFirstMessage(envelope, prekeyPublics, prekeySecrets)
			if err == nil {
				// assumption was correct, found a prekey that matched
				StoreRatchet(d, message.Dename, ratch)

				//TODO: Update prekeys by removing index, store
				if err = d.receiveMessage(message); err != nil {
					return err
				}
				newPrekeyPublics := append(prekeyPublics[:index], prekeyPublics[index+1:]...)
				newPrekeySecrets := append(prekeySecrets[:index], prekeySecrets[index+1:]...)
				if err = StorePrekeys(d, newPrekeyPublics, newPrekeySecrets); err != nil {
					return err
				}
			} else { // try decrypting with a ratchet
				fillAuth := util.FillAuthWith((*[32]byte)(&d.MessageAuthSecretKey))
				checkAuth := util.CheckAuthWith(d.ProfileRatchet)
				ratchets, err := AllRatchets(d, fillAuth, checkAuth)
				if err != nil {
					return err
				}

				message, ratch, err := d.decryptMessage(envelope, ratchets)
				if err != nil {
					return err
				}
				if err = d.receiveMessage(message); err != nil {
					return err
				}

				StoreRatchet(d, message.Dename, ratch)
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}

func (d *Daemon) ProfileRatchet(name string, reply *dename.ClientReply) (*dename.Profile, error) {
	if reply != nil {
		if profile, err := d.foreignDenameClient.LookupFromReply(name, reply); err == nil {
			// case 1: a fresh lookup is provided by the sender: remember and use it
			return d.LatestProfile(name, profile)
		}
	}
	stored, err := d.LatestProfile(name, nil)
	if err == nil && stored != nil {
		// case 2: if we already have a profile, we don't care about absolute freshness.
		// This is okay assuming that 1) the original profile we got was fresh at
		// some point and 2) any changes after that would be pushed to us by the
		// profile owner sing a case 1 message. We still ignore received profiles
		// that are older than the one currently stored.
		if reply != nil {
			if profile, err := d.timelessDenameClient.LookupFromReply(name, reply); err == nil {
				return d.LatestProfile(name, profile)
			}
		}
		return stored, nil
	}

	// case 3: look up the profile ourselves and remember it.  This should only
	// happen if somebody sends us a message and we receive it when its bundled
	// lookup is no longer fresh.
	profile, err := d.foreignDenameClient.Lookup(name)
	if err != nil {
		return nil, err
	}
	return d.LatestProfile(name, profile)
}

func (d *Daemon) onOurDenameProfileDownload(p *dename.Profile, r *dename.ClientReply, e error) {
	d.ourDenameLookup = r
	d.MarshalToFile(d.ourDenameLookupReplyPath(), r)
}

func (d *Daemon) sendFirstMessage(msg []byte, theirDename string) (*ratchet.Ratchet, error) {
	profile, err := d.foreignDenameClient.Lookup(theirDename)
	if err != nil {
		return nil, err
	}
	if err := d.MarshalToFile(d.profilePath(theirDename), profile); err != nil {
		return nil, err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return nil, err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return nil, err
	}

	addr := chatProfile.ServerAddressTCP
	pkSig := (*[32]byte)(&chatProfile.KeySigningKey)
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	ourSkAuth := (*[32]byte)(&d.MessageAuthSecretKey)

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(addr, port, pkTransport)
	if err != nil {
		return nil, err
	}
	defer theirConn.Close()

	theirKey, err := util.GetKey(theirConn, theirInBuf, d.outBuf, theirPk, theirDename, pkSig)
	if err != nil {
		return nil, err
	}
	encMsg, ratch, err := util.EncryptAuthFirst(msg, ourSkAuth, theirKey, d.ProfileRatchet)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, d.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (d *Daemon) sendMessage(msg []byte, theirDename string, msgRatch *ratchet.Ratchet) (*ratchet.Ratchet, error) {
	profile := new(dename.Profile)
	err := persistence.UnmarshalFromFile(d.profilePath(theirDename), profile)
	if err != nil {
		return nil, err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return nil, err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return nil, err
	}

	addr := chatProfile.ServerAddressTCP
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(addr, port, pkTransport)
	if err != nil {
		return nil, err
	}
	defer theirConn.Close()

	encMsg, ratch, err := util.EncryptAuth(msg, msgRatch)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, d.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (d *Daemon) decryptFirstMessage(envelope []byte, pkList []*[32]byte, skList []*[32]byte) (*proto.Message, *ratchet.Ratchet, int, error) {
	skAuth := (*[32]byte)(&d.MessageAuthSecretKey)
	ratch, msg, index, err := util.DecryptAuthFirst(envelope, pkList, skList, skAuth, d.ProfileRatchet)

	if err != nil {
		return nil, nil, -1, err
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, -1, err
	}

	return message, ratch, index, nil
}

func (d *Daemon) decryptMessage(envelope []byte, ratchets []*ratchet.Ratchet) (*proto.Message, *ratchet.Ratchet, error) {
	var ratch *ratchet.Ratchet
	var msg []byte
	for _, msgRatch := range ratchets {
		var err error
		ratch, msg, err = util.DecryptAuth(envelope, msgRatch)
		if err == nil {
			break // found the right ratchet
		}
	}
	if msg == nil {
		return nil, nil, errors.New("Invalid message received.")
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, err
	}
	return message, ratch, nil
}

func (d *Daemon) processOutboxDir(dirname string) error {
	// TODO: refactor: separate message assembly and filesystem access?
	fmt.Printf("processing outbox dir: %s\n", dirname)
	// parse metadata
	metadataFile := filepath.Join(dirname, persistence.MetadataFileName)
	if _, err := os.Stat(metadataFile); err != nil {
		return nil // no metadata --> not an outgoing message
	}

	metadata := proto.ConversationMetadata{}
	err := persistence.UnmarshalFromFile(metadataFile, &metadata)
	if err != nil {
		return err
	}

	// load messages
	potentialMessages, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}
	messages := make([][]byte, 0, len(potentialMessages))
	sendTime := d.Now().UTC()
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != persistence.MetadataFileName {
			msg, err := ioutil.ReadFile(filepath.Join(dirname, finfo.Name()))
			if err != nil {
				return err
			}

			// make protobuf for message; append it
			payload := proto.Message{
				Dename:       d.Dename,
				DenameLookup: d.ourDenameLookup,
				Contents:     msg,
				Subject:      metadata.Subject,
				Participants: metadata.Participants,
				Date:         sendTime.UnixNano(),
			}
			payloadBytes, err := payload.Marshal()
			if err != nil {
				return err
			}
			messages = append(messages, payloadBytes)
		}
	}
	if len(messages) == 0 {
		return nil // no messages to send, just the metadata file
	}

	// ensure the conversation directory exists
	convName, err := filepath.Rel(d.OutboxDir(), dirname)
	if err != nil {
		return err
	}
	convPath := filepath.Join(d.ConversationDir(), convName)
	if os.Mkdir(convPath, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// copy the metadata file to the conversation directory if it isn't already there
	convMetadataFile := filepath.Join(convPath, persistence.MetadataFileName)
	if _, err = os.Stat(convMetadataFile); err != nil {
		if os.IsNotExist(err) {
			if err = d.MarshalToFile(convMetadataFile, &metadata); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, recipient := range metadata.Participants {
		if recipient == d.Dename {
			continue
		}
		for _, msg := range messages {
			fillAuth := util.FillAuthWith((*[32]byte)(&d.MessageAuthSecretKey))
			checkAuth := util.CheckAuthWith(d.ProfileRatchet)
			if err != nil {
				return err
			}
			if msgRatch, err := LoadRatchet(d, recipient, fillAuth, checkAuth); err != nil { //First message in this conversation
				ratch, err := d.sendFirstMessage(msg, recipient)
				if err != nil {
					return err
				}
				StoreRatchet(d, recipient, ratch)
			} else { // Not-first message in this conversation
				ratch, err := d.sendMessage(msg, recipient, msgRatch)
				if err != nil {
					return err
				}
				StoreRatchet(d, recipient, ratch)
			}
		}
	}

	// move the sent messages to the conversation folder
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != persistence.MetadataFileName {
			messageName := persistence.MessageName(sendTime, string(d.Dename))
			if err = os.Rename(filepath.Join(dirname, finfo.Name()), filepath.Join(convPath, messageName)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Daemon) receiveMessage(message *proto.Message) error {
	// generate metadata file
	metadata := proto.ConversationMetadata{
		Participants: message.Participants,
		Subject:      message.Subject,
	}

	// generate conversation name
	convName := persistence.ConversationName(&metadata)

	// create conversation directory if it doesn't already exist
	convDir := filepath.Join(d.ConversationDir(), convName)
	if err := os.Mkdir(convDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// create outbox directory if it doesn't already exist
	outDir := filepath.Join(d.OutboxDir(), convName)
	if err := os.Mkdir(outDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// create conversation metadata file if it doesn't already exist
	convMetadataFile := filepath.Join(convDir, persistence.MetadataFileName)
	if _, err := os.Stat(convMetadataFile); err != nil {
		if os.IsNotExist(err) {
			d.MarshalToFile(convMetadataFile, &metadata)
		} else {
			return err
		}
	}

	// create outbox metadata file if it doesn't already exist
	outMetadataFile := filepath.Join(outDir, persistence.MetadataFileName)
	if _, err := os.Stat(outMetadataFile); err != nil {
		if os.IsNotExist(err) {
			d.MarshalToFile(outMetadataFile, &metadata)
		} else {
			return err
		}
	}

	// generate the message name: date-sender
	messageName := persistence.MessageName(time.Unix(0, message.Date), string(message.Dename))
	fmt.Printf("new message name: %s\n", messageName)

	// write the message to the conversation folder
	if err := ioutil.WriteFile(filepath.Join(convDir, messageName), message.Contents, 0600); err != nil {
		return err
	}

	return nil
}
