// Code generated by protoc-gen-gogo.
// source: LocalConversationMetadata.proto
// DO NOT EDIT!

package proto

import proto1 "github.com/gogo/protobuf/proto"
import math "math"

// discarding unused import gogoproto "github.com/gogo/protobuf/gogoproto/gogo.pb"

import io4 "io"
import fmt4 "fmt"
import github_com_gogo_protobuf_proto4 "github.com/gogo/protobuf/proto"

import bytes4 "bytes"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = math.Inf

type ConversationMetadata struct {
	Participants     []string `protobuf:"bytes,1,rep" json:"Participants"`
	Subject          string   `protobuf:"bytes,2,req" json:"Subject"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *ConversationMetadata) Reset()         { *m = ConversationMetadata{} }
func (m *ConversationMetadata) String() string { return proto1.CompactTextString(m) }
func (*ConversationMetadata) ProtoMessage()    {}

func init() {
}
func (m *ConversationMetadata) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io4.ErrUnexpectedEOF
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
				return fmt4.Errorf("proto: wrong wireType = %d for field Participants", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io4.ErrUnexpectedEOF
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
				return io4.ErrUnexpectedEOF
			}
			m.Participants = append(m.Participants, string(data[index:postIndex]))
			index = postIndex
		case 2:
			if wireType != 2 {
				return fmt4.Errorf("proto: wrong wireType = %d for field Subject", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io4.ErrUnexpectedEOF
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
				return io4.ErrUnexpectedEOF
			}
			m.Subject = string(data[index:postIndex])
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
			skippy, err := github_com_gogo_protobuf_proto4.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io4.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *ConversationMetadata) Size() (n int) {
	var l int
	_ = l
	if len(m.Participants) > 0 {
		for _, s := range m.Participants {
			l = len(s)
			n += 1 + l + sovLocalConversationMetadata(uint64(l))
		}
	}
	l = len(m.Subject)
	n += 1 + l + sovLocalConversationMetadata(uint64(l))
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovLocalConversationMetadata(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozLocalConversationMetadata(x uint64) (n int) {
	return sovLocalConversationMetadata(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func NewPopulatedConversationMetadata(r randyLocalConversationMetadata, easy bool) *ConversationMetadata {
	this := &ConversationMetadata{}
	if r.Intn(10) != 0 {
		v1 := r.Intn(10)
		this.Participants = make([]string, v1)
		for i := 0; i < v1; i++ {
			this.Participants[i] = randStringLocalConversationMetadata(r)
		}
	}
	this.Subject = randStringLocalConversationMetadata(r)
	if !easy && r.Intn(10) != 0 {
		this.XXX_unrecognized = randUnrecognizedLocalConversationMetadata(r, 3)
	}
	return this
}

type randyLocalConversationMetadata interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneLocalConversationMetadata(r randyLocalConversationMetadata) rune {
	res := rune(r.Uint32() % 1112064)
	if 55296 <= res {
		res += 2047
	}
	return res
}
func randStringLocalConversationMetadata(r randyLocalConversationMetadata) string {
	v2 := r.Intn(100)
	tmps := make([]rune, v2)
	for i := 0; i < v2; i++ {
		tmps[i] = randUTF8RuneLocalConversationMetadata(r)
	}
	return string(tmps)
}
func randUnrecognizedLocalConversationMetadata(r randyLocalConversationMetadata, maxFieldNumber int) (data []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		data = randFieldLocalConversationMetadata(data, r, fieldNumber, wire)
	}
	return data
}
func randFieldLocalConversationMetadata(data []byte, r randyLocalConversationMetadata, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(key))
		v3 := r.Int63()
		if r.Intn(2) == 0 {
			v3 *= -1
		}
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(v3))
	case 1:
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(key))
		ll := r.Intn(100)
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(ll))
		for j := 0; j < ll; j++ {
			data = append(data, byte(r.Intn(256)))
		}
	default:
		data = encodeVarintPopulateLocalConversationMetadata(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return data
}
func encodeVarintPopulateLocalConversationMetadata(data []byte, v uint64) []byte {
	for v >= 1<<7 {
		data = append(data, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	data = append(data, uint8(v))
	return data
}
func (m *ConversationMetadata) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ConversationMetadata) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Participants) > 0 {
		for _, s := range m.Participants {
			data[i] = 0xa
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
	data[i] = 0x12
	i++
	i = encodeVarintLocalConversationMetadata(data, i, uint64(len(m.Subject)))
	i += copy(data[i:], m.Subject)
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func encodeFixed64LocalConversationMetadata(data []byte, offset int, v uint64) int {
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
func encodeFixed32LocalConversationMetadata(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintLocalConversationMetadata(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (this *ConversationMetadata) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*ConversationMetadata)
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
	if len(this.Participants) != len(that1.Participants) {
		return false
	}
	for i := range this.Participants {
		if this.Participants[i] != that1.Participants[i] {
			return false
		}
	}
	if this.Subject != that1.Subject {
		return false
	}
	if !bytes4.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return false
	}
	return true
}
