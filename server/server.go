package server

import (
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"sync"
)

var wO_sync = &opt.WriteOptions{Sync: true}

type Server struct {
	database *leveldb.DB
	shutdown chan struct{}
	listener net.Listener
	notifier Notifier
	wg       sync.WaitGroup
	pk       *[32]byte
	sk       *[32]byte
	keyMutex sync.Mutex
}

func StartServer(db *leveldb.DB, shutdown chan struct{}, pk *[32]byte, sk *[32]byte, listenAddr string) (*Server, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	server := &Server{
		database: db,
		shutdown: shutdown,
		listener: listener,
		notifier: Notifier{waiters: make(map[[32]byte][]chan []byte)},
		pk:       pk,
		sk:       sk,
	}
	server.wg.Add(1)
	go server.RunServer()
	return server, nil
}

func (server *Server) StopServer() {
	close(server.shutdown)
	server.listener.Close()
	server.wg.Wait()
}

func (server *Server) RunServer() error {
	defer server.wg.Done()
	for {
		select {
		case <-server.shutdown:
			return nil
		default: //
		}
		conn, err := server.listener.Accept()
		if err != nil {
			return err
		}

		server.wg.Add(1)
		go server.handleClient(conn)
	}
}

func (server *Server) handleClientShutdown(connection *transport.Conn) {
	defer server.wg.Done()
	<-server.shutdown
	(*connection).Close()
	return
}

// readClientCommands reads client commands from a connnection and sends them
// to channel commands. On error, the error is sent to channel disconnect and
// both channels (but not the connection are closed).
// commands is a TWO-WAY channel! the reader must reach return each cmd after
// interpreting it, readClientCommands will call cmd.Reset() and reuse it.
func (server *Server) readClientCommands(conn *transport.Conn,
	commands chan *proto.ClientToServer, disconnected chan error) {
	defer server.wg.Done()
	defer close(commands)
	defer close(disconnected)
	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	cmd := new(proto.ClientToServer)
	for {
		num, err := conn.ReadFrame(inBuf)
		if err != nil {
			disconnected <- err
			return
		}
		unpadMsg := proto.Unpad(inBuf[:num])
		if err := cmd.Unmarshal(unpadMsg); err != nil {
			disconnected <- err
			return
		}
		commands <- cmd
		cmd = <-commands
		cmd.Reset()
	}
}

// readClientNotifications is a for loop of blocking reads on notificationsIn
// and non-blocking sends on notificationsOut. If the input channel is closed
// or a send would block, the output channel is closed.
func (server *Server) readClientNotifications(notificationsIn chan []byte, notificationsOut chan []byte) {
	var hasOverflowed bool
	defer server.wg.Done()
	for n := range notificationsIn {
		if !hasOverflowed {
			select {
			case notificationsOut <- n:
			default:
				hasOverflowed = true
				close(notificationsOut)
			}
		}
	}
}

//for each client, listen for commands
func (server *Server) handleClient(connection net.Conn) error {
	defer server.wg.Done()
	newConnection, uid, err := transport.Handshake(connection, server.pk, server.sk, nil, proto.SERVER_MESSAGE_SIZE) //TODO: Decide on this bound
	if err != nil {
		return err
	}

	commands := make(chan *proto.ClientToServer)
	disconnected := make(chan error)
	server.wg.Add(2)
	go server.readClientCommands(newConnection, commands, disconnected)
	go server.handleClientShutdown(newConnection)

	var notificationsUnbuffered, notifications chan []byte
	var notifyEnabled bool
	defer func() {
		if notifyEnabled {
			server.notifier.StopWaitingSync(uid, notificationsUnbuffered)
		}
	}()

	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	response := new(proto.ServerToClient)
	for {
		select {
		case err := <-disconnected:
			return err
		case cmd := <-commands:
			if cmd.CreateAccount != nil && *cmd.CreateAccount {
				err = server.newUser(uid)
			} else if cmd.DeliverEnvelope != nil {
				err = server.newMessage((*[32]byte)(cmd.DeliverEnvelope.User),
					cmd.DeliverEnvelope.Envelope)
			} else if cmd.ListMessages != nil && *cmd.ListMessages {
				var messageList [][32]byte
				messageList, err = server.getMessageList(uid)
				response.MessageList = proto.ToProtoByte32List(messageList)
			} else if cmd.DownloadEnvelope != nil {
				response.Envelope, err = server.getEnvelope(uid, (*[32]byte)(cmd.DownloadEnvelope))
			} else if cmd.DeleteMessages != nil {
				messageList := cmd.DeleteMessages
				err = server.deleteMessages(uid, proto.To32ByteList(messageList))
			} else if cmd.UploadSignedKeys != nil {
				err = server.newKeys(uid, cmd.UploadSignedKeys)
			} else if cmd.GetSignedKey != nil {
				response.SignedKey, err = server.getKey((*[32]byte)(cmd.GetSignedKey))
			} else if cmd.GetNumKeys != nil {
				response.NumKeys, err = server.getNumKeys(uid)
			} else if cmd.ReceiveEnvelopes != nil {
				if *cmd.ReceiveEnvelopes && !notifyEnabled {
					notifyEnabled = true
					notificationsUnbuffered = server.notifier.StartWaiting(uid)
					notifications = make(chan []byte)
					server.wg.Add(1)
					go server.readClientNotifications(notificationsUnbuffered, notifications)
				} else if !*cmd.ReceiveEnvelopes && notifyEnabled {
					server.notifier.StopWaitingSync(uid, notificationsUnbuffered)
					notifyEnabled = false
				}
			}
			if err != nil {
				fmt.Printf("Server error: %v\n", err)
				response.Status = proto.ServerToClient_PARSE_ERROR.Enum()
			} else {
				response.Status = proto.ServerToClient_OK.Enum()
			}
			if err = server.writeProtobuf(newConnection, outBuf, response); err != nil {
				return err
			}
			commands <- cmd
		case notification, ok := <-notifications:
			if !notifyEnabled {
				continue
			}
			if !ok {
				notifyEnabled = false
				go server.notifier.StopWaitingSync(uid, notificationsUnbuffered)
				continue
			}
			response.Envelope = notification
			response.Status = proto.ServerToClient_OK.Enum()
			if err = server.writeProtobuf(newConnection, outBuf, response); err != nil {
				return err
			}
		}
		response.Reset()
	}
}

func (server *Server) getNumKeys(user *[32]byte) (*int64, error) { //TODO: Batch read of some kind?
	prefix := append([]byte{'k'}, (*user)[:]...)
	snapshot, err := server.database.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	keyRange := util.BytesPrefix(prefix)
	iter := snapshot.NewIterator(keyRange, nil)
	defer iter.Release()
	var numRecords int64
	for iter.Next() {
		numRecords = numRecords + 1
	}
	return &numRecords, iter.Error()
}

func (server *Server) deleteKey(uid *[32]byte, key []byte) error {
	keyHash := sha256.Sum256((key))
	dbKey := append(append([]byte{'k'}, uid[:]...), keyHash[:]...)
	return server.database.Delete(dbKey, wO_sync)
}

func (server *Server) getKey(user *[32]byte) ([]byte, error) {
	prefix := append([]byte{'k'}, user[:]...)
	server.keyMutex.Lock()
	defer server.keyMutex.Unlock()
	snapshot, err := server.database.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	keyRange := util.BytesPrefix(prefix)
	iter := snapshot.NewIterator(keyRange, nil)
	defer iter.Release()
	if iter.First() == false {
		return nil, errors.New("No keys left in database")
	}
	err = iter.Error()
	server.deleteKey(user, iter.Value())
	return append([]byte{}, iter.Value()...), err
}

func (server *Server) newKeys(uid *[32]byte, keyList [][]byte) error {
	batch := new(leveldb.Batch)
	for _, key := range keyList {
		keyHash := sha256.Sum256(key)
		dbKey := append(append([]byte{'k'}, uid[:]...), keyHash[:]...)
		batch.Put(dbKey, key)
	}
	return server.database.Write(batch, wO_sync)
}
func (server *Server) deleteMessages(uid *[32]byte, messageList [][32]byte) error {
	batch := new(leveldb.Batch)
	for _, messageHash := range messageList {
		key := append(append([]byte{'m'}, uid[:]...), messageHash[:]...)
		batch.Delete(key)
	}
	return server.database.Write(batch, wO_sync)
}

func (server *Server) getEnvelope(uid *[32]byte, messageHash *[32]byte) ([]byte, error) {
	key := append(append([]byte{'m'}, uid[:]...), (messageHash)[:]...)
	envelope, err := server.database.Get(key, nil)
	return envelope, err
}

func (server *Server) writeProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ServerToClient) error {
	unpadMsg, err := protobuf.Marshal(message)
	if err != nil {
		return err
	}
	padMsg := proto.Pad(unpadMsg, proto.SERVER_MESSAGE_SIZE)
	copy(outBuf, padMsg)
	conn.WriteFrame(outBuf[:proto.SERVER_MESSAGE_SIZE])
	return nil
}

func (server *Server) getMessageList(user *[32]byte) ([][32]byte, error) {
	messages := make([][32]byte, 0)
	prefix := append([]byte{'m'}, (*user)[:]...)
	messageRange := util.BytesPrefix(prefix)
	snapshot, err := server.database.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	iter := snapshot.NewIterator(messageRange, nil)
	defer iter.Release()
	for iter.Next() {
		var message [32]byte
		copy(message[:], iter.Key()[len(prefix):len(prefix)+32])
		messages = append(messages, message)
	}
	err = iter.Error()
	return messages, err
}

func (server *Server) newMessage(uid *[32]byte, envelope []byte) error {
	// TODO: check that user exists
	messageHash := sha256.Sum256(envelope)
	key := append(append([]byte{'m'}, uid[:]...), messageHash[:]...)
	err := server.database.Put(key, (envelope)[:], wO_sync)
	if err != nil {
		return err
	}
	server.notifier.Notify(uid, append([]byte{}, envelope...))
	return nil
}

func (server *Server) newUser(uid *[32]byte) error {
	return server.database.Put(append([]byte{'u'}, uid[:]...), []byte(""), wO_sync)
}
