// Code generated by protoc-gen-gogo.
// source: ClientClient.proto
// DO NOT EDIT!

/*
	Package proto is a generated protocol buffer package.

	It is generated from these files:
		ClientClient.proto
		ClientServer.proto
		DenameChatProfile.proto
		LocalAccountConfig.proto
		LocalConversationMetadata.proto
		Prekeys.proto

	It has these top-level messages:
		Message
*/
package proto

import proto1 "github.com/gogo/protobuf/proto"
import math "math"

// discarding unused import gogoproto "github.com/gogo/protobuf/gogoproto/gogo.pb"

import github_com_andres_erbsen_dename_protocol "github.com/andres-erbsen/dename/protocol"

import io "io"
import fmt "fmt"
import github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"

import bytes "bytes"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = math.Inf

type Message struct {
	Contents         []byte                                                `protobuf:"bytes,1,req,name=contents" json:"contents"`
	Subject          string                                                `protobuf:"bytes,2,req,name=subject" json:"subject"`
	Participants     []string                                              `protobuf:"bytes,3,rep,name=participants" json:"participants"`
	Date             int64                                                 `protobuf:"varint,4,req,name=date" json:"date"`
	Dename           string                                                `protobuf:"bytes,5,req,name=dename" json:"dename"`
	DenameLookup     *github_com_andres_erbsen_dename_protocol.ClientReply `protobuf:"bytes,6,req,name=dename_lookup,customtype=github.com/andres-erbsen/dename/protocol.ClientReply" json:"dename_lookup,omitempty"`
	XXX_unrecognized []byte                                                `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto1.CompactTextString(m) }
func (*Message) ProtoMessage()    {}

func init() {
}
func (m *Message) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Contents", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Contents = append(m.Contents, data[index:postIndex]...)
			index = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Subject", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Subject = string(data[index:postIndex])
			index = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Participants", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Participants = append(m.Participants, string(data[index:postIndex]))
			index = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Date", wireType)
			}
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				m.Date |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Dename", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + int(stringLen)
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Dename = string(data[index:postIndex])
			index = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DenameLookup", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DenameLookup = &github_com_andres_erbsen_dename_protocol.ClientReply{}
			if err := m.DenameLookup.Unmarshal(data[index:postIndex]); err != nil {
				return err
			}
			index = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			index -= sizeOfWire
			skippy, err := github_com_gogo_protobuf_proto.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *Message) Size() (n int) {
	var l int
	_ = l
	l = len(m.Contents)
	n += 1 + l + sovClientClient(uint64(l))
	l = len(m.Subject)
	n += 1 + l + sovClientClient(uint64(l))
	if len(m.Participants) > 0 {
		for _, s := range m.Participants {
			l = len(s)
			n += 1 + l + sovClientClient(uint64(l))
		}
	}
	n += 1 + sovClientClient(uint64(m.Date))
	l = len(m.Dename)
	n += 1 + l + sovClientClient(uint64(l))
	if m.DenameLookup != nil {
		l = m.DenameLookup.Size()
		n += 1 + l + sovClientClient(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovClientClient(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozClientClient(x uint64) (n int) {
	return sovClientClient(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Message) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Message) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintClientClient(data, i, uint64(len(m.Contents)))
	i += copy(data[i:], m.Contents)
	data[i] = 0x12
	i++
	i = encodeVarintClientClient(data, i, uint64(len(m.Subject)))
	i += copy(data[i:], m.Subject)
	if len(m.Participants) > 0 {
		for _, s := range m.Participants {
			data[i] = 0x1a
			i++
			l = len(s)
			for l >= 1<<7 {
				data[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			data[i] = uint8(l)
			i++
			i += copy(data[i:], s)
		}
	}
	data[i] = 0x20
	i++
	i = encodeVarintClientClient(data, i, uint64(m.Date))
	data[i] = 0x2a
	i++
	i = encodeVarintClientClient(data, i, uint64(len(m.Dename)))
	i += copy(data[i:], m.Dename)
	if m.DenameLookup != nil {
		data[i] = 0x32
		i++
		i = encodeVarintClientClient(data, i, uint64(m.DenameLookup.Size()))
		n1, err := m.DenameLookup.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func encodeFixed64ClientClient(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32ClientClient(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintClientClient(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (this *Message) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*Message)
	if !ok {
		return false
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if !bytes.Equal(this.Contents, that1.Contents) {
		return false
	}
	if this.Subject != that1.Subject {
		return false
	}
	if len(this.Participants) != len(that1.Participants) {
		return false
	}
	for i := range this.Participants {
		if this.Participants[i] != that1.Participants[i] {
			return false
		}
	}
	if this.Date != that1.Date {
		return false
	}
	if this.Dename != that1.Dename {
		return false
	}
	if that1.DenameLookup == nil {
		if this.DenameLookup != nil {
			return false
		}
	} else if !this.DenameLookup.Equal(*that1.DenameLookup) {
		return false
	}
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return false
	}
	return true
}
