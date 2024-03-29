// Code generated by protoc-gen-go.
// source: protobuf/request.proto
// DO NOT EDIT!

package protocol

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Request struct {
	ID                string `protobuf:"bytes,1,opt,name=ID" json:"ID,omitempty"`
	TransactionID     string `protobuf:"bytes,2,opt,name=TransactionID" json:"TransactionID,omitempty"`
	SuccessReplyTopic string `protobuf:"bytes,3,opt,name=SuccessReplyTopic" json:"SuccessReplyTopic,omitempty"`
	FailureReplyTopic string `protobuf:"bytes,4,opt,name=FailureReplyTopic" json:"FailureReplyTopic,omitempty"`
	Data              []byte `protobuf:"bytes,5,opt,name=Data,proto3" json:"Data,omitempty"`
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func init() {
	proto.RegisterType((*Request)(nil), "protocol.Request")
}

var fileDescriptor1 = []byte{
	// 164 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x4f, 0x2a, 0x4d, 0xd3, 0x2f, 0x4a, 0x2d, 0x2c, 0x4d, 0x2d, 0x2e, 0xd1, 0x03, 0x0b, 0x08,
	0x71, 0x80, 0xa9, 0xe4, 0xfc, 0x1c, 0xa5, 0xf5, 0x8c, 0x5c, 0xec, 0x41, 0x10, 0x39, 0x21, 0x3e,
	0x2e, 0x26, 0x4f, 0x17, 0x09, 0x46, 0x05, 0x46, 0x0d, 0xce, 0x20, 0x26, 0x4f, 0x17, 0x21, 0x15,
	0x2e, 0xde, 0x90, 0xa2, 0xc4, 0xbc, 0xe2, 0xc4, 0xe4, 0x92, 0xcc, 0xfc, 0x3c, 0x4f, 0x17, 0x09,
	0x26, 0xb0, 0x14, 0xaa, 0xa0, 0x90, 0x0e, 0x97, 0x60, 0x70, 0x69, 0x72, 0x72, 0x6a, 0x71, 0x71,
	0x50, 0x6a, 0x41, 0x4e, 0x65, 0x48, 0x7e, 0x41, 0x66, 0xb2, 0x04, 0x33, 0x58, 0x25, 0xa6, 0x04,
	0x48, 0xb5, 0x5b, 0x62, 0x66, 0x4e, 0x69, 0x51, 0x2a, 0x92, 0x6a, 0x16, 0x88, 0x6a, 0x0c, 0x09,
	0x21, 0x21, 0x2e, 0x16, 0x97, 0xc4, 0x92, 0x44, 0x09, 0x56, 0x05, 0x46, 0x0d, 0x9e, 0x20, 0x30,
	0x3b, 0x89, 0x0d, 0xec, 0x76, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x1a, 0x7c, 0xf7, 0x10,
	0xdc, 0x00, 0x00, 0x00,
}
