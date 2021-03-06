package client

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"time"

	"code.google.com/p/go.crypto/curve25519"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/agl/ed25519"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	dename "github.com/andres-erbsen/dename/protocol"
	"golang.org/x/net/proxy"
)

const PROFILE_FIELD_ID = 1984
const ENCRYPT_ADDED_LEN = 168
const ENCRYPT_FIRST_ADDED_LEN = 200

type ProfileRatchet func(string, *dename.ClientReply) (*dename.Profile, error)

func ReceiveReply(connToServer *ConnectionToServer) (*proto.ServerToClient, error) {
	response := <-connToServer.ReadReply //TODO: Timeout
	return response, nil
}

func CreateAccount(conn *transport.Conn, inBuf []byte) error {
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	if err := WriteProtobuf(conn, command); err != nil {
		return err
	}

	_, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func ListUserMessages(connToServer *ConnectionToServer) ([][32]byte, error) {
	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	if err := WriteProtobuf(connToServer.Conn, listMessages); err != nil {
		return nil, err
	}

	response, err := ReceiveReply(connToServer)
	if err != nil {
		return nil, err
	}

	return proto.To32ByteList(response.MessageList), nil
}

func RequestMessage(connToServer *ConnectionToServer, messageHash *[32]byte) error {
	getEnvelope := &proto.ClientToServer{
		DownloadEnvelope: (*proto.Byte32)(messageHash),
	}
	if err := WriteProtobuf(connToServer.Conn, getEnvelope); err != nil {
		return err
	}
	return nil
}

func SignKeys(keys []*[32]byte, sk *[64]byte) [][]byte {

	pkList := make([][]byte, 0)
	for _, key := range keys {
		signature := ed25519.Sign(sk, key[:])
		pkList = append(pkList, append(append([]byte{}, key[:]...), signature[:]...))
	}
	return pkList
}

func TorAnon(addr string) proxy.Dialer {
	var identity [16]byte
	if _, err := rand.Read(identity[:]); err != nil {
		panic(err)
	}
	dialer, err := proxy.SOCKS5("tcp", addr, &proxy.Auth{
		User:     fmt.Sprintf("%x", identity[:8]),
		Password: fmt.Sprintf("%x", identity[8:]),
	}, proxy.Direct)
	if err != nil {
		panic(err)
	}
	return dialer
}

func EncryptAuthFirst(message []byte, skAuth *[32]byte, userKey *[32]byte, prt ProfileRatchet) ([]byte, *ratchet.Ratchet, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(prt),
	}

	out := append([]byte{}, (*userKey)[:]...)
	paddedMsg := proto.Pad(message, proto.MAX_MESSAGE_SIZE-ENCRYPT_FIRST_ADDED_LEN-len(out))
	out = ratch.EncryptFirst(out, paddedMsg, userKey)

	return out, ratch, nil
}

func EncryptAuth(message []byte, ratch *ratchet.Ratchet) ([]byte, *ratchet.Ratchet, error) {
	paddedMsg := proto.Pad(message, proto.MAX_MESSAGE_SIZE-ENCRYPT_ADDED_LEN)
	out := ratch.Encrypt(nil, paddedMsg)

	return out, ratch, nil
}

func DecryptAuthFirst(in []byte, pkList []*[32]byte, skList []*[32]byte, skAuth *[32]byte, prt ProfileRatchet) (*ratchet.Ratchet, []byte, int, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(prt),
	}

	if len(in) < 32 {
		return nil, nil, -1, errors.New("Message length incorrect.")
	}
	var pkAuth [32]byte
	copy(pkAuth[:], in[:32])

	envelope := in[32:]
	for i, pk := range pkList {
		if *pk == pkAuth {
			msg, err := ratch.DecryptFirst(envelope, skList[i])
			if err == nil {
				unpadMsg := proto.Unpad(msg)
				return ratch, unpadMsg, i, nil
			}
		}
	}
	return nil, nil, -1, errors.New("Invalid first message received.") //TODO: Should I make the error message something different?
}
func DecryptAuth(in []byte, ratch *ratchet.Ratchet) (*ratchet.Ratchet, []byte, error) {
	msg, err := ratch.Decrypt(in)
	if err != nil {
		return nil, nil, err
	}
	unpadMsg := proto.Unpad(msg)
	return ratch, unpadMsg, nil
}

func DeleteMessages(connToServer *ConnectionToServer, messageList [][32]byte) error {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: proto.ToProtoByte32List(messageList),
	}
	if err := WriteProtobuf(connToServer.Conn, deleteMessages); err != nil {
		return err
	}

	_, err := ReceiveReply(connToServer)
	return err
}

func UploadKeys(connToServer *ConnectionToServer, keyList [][]byte) error {
	uploadKeys := &proto.ClientToServer{
		UploadSignedKeys: keyList,
	}
	if err := WriteProtobuf(connToServer.Conn, uploadKeys); err != nil {
		return err
	}

	_, err := ReceiveReply(connToServer)
	return err
}

func GetKey(conn *transport.Conn, inBuf []byte, pk *[32]byte, dename string, pkSig *[32]byte) (*[32]byte, error) {
	getKey := &proto.ClientToServer{
		GetSignedKey: (*proto.Byte32)(pk),
	}
	if err := WriteProtobuf(conn, getKey); err != nil {
		return nil, err
	}

	response, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}

	var userKey [32]byte
	copy(userKey[:], response.SignedKey[:32])

	var sig [64]byte
	copy(sig[:], response.SignedKey[32:(32+64)])

	if !ed25519.Verify(pkSig, userKey[:], &sig) {
		return nil, errors.New("Improperly signed key returned")
	}

	return &userKey, nil
}

func GetNumKeys(connToServer *ConnectionToServer) (int64, error) {
	getNumKeys := &proto.ClientToServer{
		GetNumKeys: protobuf.Bool(true),
	}
	if err := WriteProtobuf(connToServer.Conn, getNumKeys); err != nil {
		return 0, err
	}

	response, err := ReceiveReply(connToServer)
	if err != nil {
		return 0, err
	}
	return *response.NumKeys, nil
}

func EnablePush(connToServer *ConnectionToServer) error {
	true_ := true
	command := &proto.ClientToServer{
		ReceiveEnvelopes: &true_,
	}
	if err := WriteProtobuf(connToServer.Conn, command); err != nil {
		return err
	}
	_, err := ReceiveReply(connToServer)
	if err != nil {
		return err
	}
	return nil
}

func UploadMessageToUser(conn *transport.Conn, inBuf []byte, pk *[32]byte, envelope []byte) error {
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pk),
		Envelope: envelope,
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	if err := WriteProtobuf(conn, deliverCommand); err != nil {
		return err
	}

	_, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func WriteProtobuf(conn *transport.Conn, message *proto.ClientToServer) error {
	unpadMsg, err := protobuf.Marshal(message)
	if err != nil {
		return err
	}
	_, err = conn.WriteFrame(proto.Pad(unpadMsg, proto.SERVER_MESSAGE_SIZE))
	return err
}

func ReceiveProtobuf(conn *transport.Conn, inBuf []byte) (*proto.ServerToClient, error) {
	response := new(proto.ServerToClient)
	conn.SetDeadline(time.Now().Add(time.Hour))
	num, err := conn.ReadFrame(inBuf)
	if err != nil {
		return nil, err
	}
	unpadMsg := proto.Unpad(inBuf[:num])
	if err := response.Unmarshal(unpadMsg); err != nil {
		return nil, err
	}
	if response.Status == nil {
		return nil, errors.New("Server returned nil status.")
	}
	if *response.Status != proto.ServerToClient_OK {
		return nil, errors.New("Server did not return OK")
	}
	return response, nil
}

func GenerateLongTermKeys(secretConfig *proto.LocalAccountConfig, publicProfile *proto.Profile, rand io.Reader) error {
	if pk, sk, err := box.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.TransportSecretKeyForServer = (proto.Byte32)(*sk)
		publicProfile.UserIDAtServer = (proto.Byte32)(*pk)
	}
	if pk, sk, err := box.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.MessageAuthSecretKey = (proto.Byte32)(*sk)
		publicProfile.MessageAuthKey = (proto.Byte32)(*pk)
	}

	if pk, sk, err := ed25519.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.KeySigningSecretKey = sk[:]
		publicProfile.KeySigningKey = (proto.Byte32)(*pk)
	}
	return nil
}

func FillAuthWith(ourAuthPrivate *[32]byte) func([]byte, []byte, *[32]byte) {
	return func(tag, data []byte, theirAuthPublic *[32]byte) {
		var sharedAuthKey [32]byte
		curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)

		var ourAuthPublic [32]byte
		curve25519.ScalarBaseMult(&ourAuthPublic, ourAuthPrivate)

		h := hmac.New(sha256.New, sharedAuthKey[:])
		h.Write(data)
		copy(tag, h.Sum(nil))
	}
}

func CheckAuthWith(prt ProfileRatchet) func([]byte, []byte, []byte, *[32]byte) error {
	return func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error {
		var sharedAuthKey [32]byte
		message := new(proto.Message)
		unpadMsg := proto.Unpad(msg)
		err := message.Unmarshal(unpadMsg)
		if err != nil {
			return err
		}

		profile, err := prt(message.Dename, message.DenameLookup)
		if err != nil {
			return err
		}

		chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
		if err != nil {
			return err
		}

		chatProfile := new(proto.Profile)
		if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
			return err
		}

		theirAuthPublic := (*[32]byte)(&chatProfile.MessageAuthKey)

		curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
		h := hmac.New(sha256.New, sharedAuthKey[:])

		h.Write(data)
		if subtle.ConstantTimeCompare(tag, h.Sum(nil)[:len(tag)]) == 0 {
			return errors.New("Authentication failed: failed to reproduce envelope auth tag using the current auth pubkey from dename")
		}
		return nil
	}
}
