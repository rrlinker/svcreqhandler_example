package courier

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"reflect"
)

type Type uint64

// Message types
const (
	MTVersion       Type = 0x01
	MTAuthorization Type = 0x0A
	MTLinkLibrary   Type = 0x111B

	MTOK Type = 0x0C << 56
)

var typeMapMsg2Go = map[Type]reflect.Type{
	MTVersion:       reflect.TypeOf(Version{}),
	MTAuthorization: reflect.TypeOf(Authorization{}),
	MTLinkLibrary:   reflect.TypeOf(LinkLibrary{}),

	MTOK: reflect.TypeOf(OK{}),
}

var typeMapGo2Msg = map[reflect.Type]Type{
	reflect.TypeOf(Version{}):       MTVersion,
	reflect.TypeOf(Authorization{}): MTAuthorization,
	reflect.TypeOf(LinkLibrary{}):   MTLinkLibrary,

	reflect.TypeOf(OK{}): MTOK,
}

var (
	ErrUnknownMessage = errors.New("unknown message")
	ErrInvalidSize    = errors.New("unexpected message size")
)

type SizeError struct {
	error
	Type         Type
	ExpectedSize uint64
	ActualSize   uint64
}

func newSizeError(t Type, expected uint64, actual uint64) *SizeError {
	return &SizeError{
		error:        ErrInvalidSize,
		Type:         t,
		ExpectedSize: expected,
		ActualSize:   actual,
	}
}

type Header struct {
	Size uint64
	Type Type
}

type Message interface {
}

type Token [128]byte

type Version struct{ Value uint64 }
type Authorization struct{ Token Token }
type LinkLibrary struct{ ByteName [64]byte }

type OK struct{}

func getTotalMessageSize(msg interface{}) uint64 {
	return uint64(binary.Size(msg)) + 16
}

func (ll *LinkLibrary) Name() string {
	n := bytes.IndexByte(ll.ByteName[:], 0)
	if n < 0 {
		return string(ll.ByteName[:])
	}
	return string(ll.ByteName[:n])

}

type Courier struct {
	conn net.Conn
}

func New(conn net.Conn) *Courier {
	return &Courier{
		conn: conn,
	}
}

func (c *Courier) readHeader() (header Header, err error) {
	err = binary.Read(c.conn, binary.LittleEndian, &header)
	return
}

func (c *Courier) Receive() (Message, error) {
	header, err := c.readHeader()
	if err != nil {
		return nil, err
	}
	if t, ok := typeMapMsg2Go[header.Type]; ok {
		msg := reflect.New(t)
		err = binary.Read(c.conn, binary.LittleEndian, msg.Interface())
		return msg.Elem().Interface(), err
	}
	return header, ErrUnknownMessage
}

func (c *Courier) Send(msg Message) {
}
