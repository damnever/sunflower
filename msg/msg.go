package msg

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/bufpool"
	"github.com/pkg/errors"
)

var (
	ErrBadRequestType     = fmt.Errorf("bad request type")
	ErrBadResponseType    = fmt.Errorf("bad response type")
	ErrUnknownMessageType = fmt.Errorf("unknown message type")

	errCodeMap = map[msgpb.ErrCode]error{
		msgpb.ErrCodeNull:                nil,
		msgpb.ErrCodeBadClient:           fmt.Errorf("bad client, try download a new client from control panel"),
		msgpb.ErrCodeBadVersion:          fmt.Errorf("bad version, try download a new client from control panel"),
		msgpb.ErrCodeBadProtoOrAddr:      fmt.Errorf("bad porotocol or address"),
		msgpb.ErrCodeBadRegistryAddr:     fmt.Errorf("bad registry address"),
		msgpb.ErrCodeNoSuchTunnel:        fmt.Errorf("no such tunnel"),
		msgpb.ErrCodeDuplicateAgent:      fmt.Errorf("duplicate agent"),
		msgpb.ErrCodeInternalServerError: fmt.Errorf("internal server error"),
	}
)

func CodeToError(errCode msgpb.ErrCode) error {
	return errCodeMap[errCode]
}

func Write(w net.Conn, v interface{}) error {
	m, err := toMessage(v)
	if err != nil {
		return err
	}

	sz := m.Size()
	buf := bufpool.GrowGet(sz)
	defer bufpool.Put(buf)
	p := buf.Bytes()[:sz]

	if sz, err = m.MarshalTo(p); err != nil {
		return errors.WithStack(err)
	}

	if err := binary.Write(w, binary.BigEndian, uint16(sz)); err != nil {
		return errors.WithStack(err)
	}

	for nw := 0; nw < sz; {
		n, err := w.Write(p[nw:])
		if err != nil {
			return errors.WithStack(err)
		}
		nw += n
	}
	return nil
}

func toMessage(v interface{}) (*msgpb.Message, error) {
	m := msgpb.Message{}
	switch x := v.(type) {
	case *msgpb.PingRequest:
		m.Body = &msgpb.Message_PingRequest{PingRequest: x}
	case *msgpb.PingResponse:
		m.Body = &msgpb.Message_PingResponse{PingResponse: x}
	case msgpb.PingRequest:
		m.Body = &msgpb.Message_PingRequest{PingRequest: &x}
	case msgpb.PingResponse:
		m.Body = &msgpb.Message_PingResponse{PingResponse: &x}

	case *msgpb.HandshakeRequest:
		m.Body = &msgpb.Message_HandshakeRequest{HandshakeRequest: x}
	case *msgpb.HandshakeResponse:
		m.Body = &msgpb.Message_HandshakeResponse{HandshakeResponse: x}
	case msgpb.HandshakeRequest:
		m.Body = &msgpb.Message_HandshakeRequest{HandshakeRequest: &x}
	case msgpb.HandshakeResponse:
		m.Body = &msgpb.Message_HandshakeResponse{HandshakeResponse: &x}

	case *msgpb.TunnelHandshakeRequest:
		m.Body = &msgpb.Message_TunnelHandshakeRequest{TunnelHandshakeRequest: x}
	case *msgpb.TunnelHandshakeResponse:
		m.Body = &msgpb.Message_TunnelHandshakeResponse{TunnelHandshakeResponse: x}
	case msgpb.TunnelHandshakeRequest:
		m.Body = &msgpb.Message_TunnelHandshakeRequest{TunnelHandshakeRequest: &x}
	case msgpb.TunnelHandshakeResponse:
		m.Body = &msgpb.Message_TunnelHandshakeResponse{TunnelHandshakeResponse: &x}

	case *msgpb.NewTunnelRequest:
		m.Body = &msgpb.Message_NewTunnelRequest{NewTunnelRequest: x}
	case *msgpb.NewTunnelResponse:
		m.Body = &msgpb.Message_NewTunnelResponse{NewTunnelResponse: x}
	case msgpb.NewTunnelRequest:
		m.Body = &msgpb.Message_NewTunnelRequest{NewTunnelRequest: &x}
	case msgpb.NewTunnelResponse:
		m.Body = &msgpb.Message_NewTunnelResponse{NewTunnelResponse: &x}

	case *msgpb.CloseTunnelRequest:
		m.Body = &msgpb.Message_CloseTunnelRequest{CloseTunnelRequest: x}
	case *msgpb.CloseTunnelResponse:
		m.Body = &msgpb.Message_CloseTunnelResponse{CloseTunnelResponse: x}
	case msgpb.CloseTunnelRequest:
		m.Body = &msgpb.Message_CloseTunnelRequest{CloseTunnelRequest: &x}
	case msgpb.CloseTunnelResponse:
		m.Body = &msgpb.Message_CloseTunnelResponse{CloseTunnelResponse: &x}

	case *msgpb.ShutdownRequest:
		m.Body = &msgpb.Message_ShutdownRequest{ShutdownRequest: x}
	case msgpb.ShutdownRequest:
		m.Body = &msgpb.Message_ShutdownRequest{ShutdownRequest: &x}

	default:
		return nil, errors.WithStack(ErrUnknownMessageType)
	}
	return &m, nil
}

func ReadTo(r net.Conn, v interface{}) error {
	m, err := Read(r)
	if err != nil {
		return errors.WithStack(err)
	}
	vv, vm := reflect.ValueOf(v).Elem(), reflect.ValueOf(m).Elem()
	if vv.Type() != vm.Type() {
		return errors.WithStack(ErrBadResponseType)
	}
	vv.Set(vm)
	return nil
}

func Read(r net.Conn) (interface{}, error) {
	var sz uint16
	if err := binary.Read(r, binary.BigEndian, &sz); err != nil {
		return nil, errors.WithStack(err)
	}

	buf := bufpool.Get()
	defer bufpool.Put(buf)

	// bytes.Buffer has WriteTo, no need additional buffer
	if _, err := io.CopyN(buf, r, int64(sz)); err != nil {
		return nil, errors.WithStack(err)
	}

	m := &msgpb.Message{}
	if err := m.Unmarshal(buf.Bytes()); err != nil {
		return nil, errors.WithStack(err)
	}
	return fromMessage(m)
}

func fromMessage(m *msgpb.Message) (interface{}, error) {
	if v := m.GetPingRequest(); v != nil {
		return v, nil
	}
	if v := m.GetPingResponse(); v != nil {
		return v, nil
	}

	if v := m.GetHandshakeRequest(); v != nil {
		return v, nil
	}
	if v := m.GetHandshakeResponse(); v != nil {
		return v, nil
	}

	if v := m.GetTunnelHandshakeRequest(); v != nil {
		return v, nil
	}
	if v := m.GetTunnelHandshakeResponse(); v != nil {
		return v, nil
	}

	if v := m.GetNewTunnelRequest(); v != nil {
		return v, nil
	}
	if v := m.GetNewTunnelResponse(); v != nil {
		return v, nil
	}

	if v := m.GetCloseTunnelRequest(); v != nil {
		return v, nil
	}
	if v := m.GetCloseTunnelResponse(); v != nil {
		return v, nil
	}

	if v := m.GetShutdownRequest(); v != nil {
		return v, nil
	}
	return nil, errors.WithStack(ErrUnknownMessageType)
}
