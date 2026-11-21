package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
)

const (
	version5    = 0x05
	authVersion = 0x01

	methodNoAuth       = 0x00
	methodUserPass     = 0x02
	methodNoAcceptable = 0xFF

	cmdConnect      = 0x01
	cmdBind         = 0x02
	cmdUDPAssociate = 0x03

	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	repSuccess             = 0x00
	repGeneralFailure      = 0x01
	repNotAllowed          = 0x02
	repNetworkUnreachable  = 0x03
	repHostUnreachable     = 0x04
	repConnectionRefused   = 0x05
	repTTLExpired          = 0x06
	repCommandNotSupported = 0x07
	repAddressNotSupported = 0x08
)

var (
	errUnsupportedVersion = errors.New("socks5: unsupported version")
	errNoMethods          = errors.New("socks5: no authentication methods offered")
	errMalformedRequest   = errors.New("socks5: malformed request")
	errUnsupportedAddress = errors.New("socks5: unsupported address type")
)

type request struct {
	cmd  byte
	host string
	port uint16
}

func readGreeting(r io.Reader) ([]byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	if header[0] != version5 {
		return nil, errUnsupportedVersion
	}
	count := int(header[1])
	if count == 0 {
		return nil, errNoMethods
	}
	methods := make([]byte, count)
	if _, err := io.ReadFull(r, methods); err != nil {
		return nil, err
	}
	return methods, nil
}

func selectMethod(methods []byte, requireAuth bool) byte {
	want := byte(methodNoAuth)
	if requireAuth {
		want = methodUserPass
	}
	for _, method := range methods {
		if method == want {
			return want
		}
	}
	return methodNoAcceptable
}

func writeMethod(w io.Writer, method byte) error {
	_, err := w.Write([]byte{version5, method})
	return err
}

func readAuth(r io.Reader) (username, password string, err error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return "", "", err
	}
	if header[0] != authVersion {
		return "", "", fmt.Errorf("%w: auth version %#x", errMalformedRequest, header[0])
	}
	userLen := int(header[1])
	if userLen == 0 {
		return "", "", fmt.Errorf("%w: empty username", errMalformedRequest)
	}
	user := make([]byte, userLen)
	if _, err := io.ReadFull(r, user); err != nil {
		return "", "", err
	}
	var passLen [1]byte
	if _, err := io.ReadFull(r, passLen[:]); err != nil {
		return "", "", err
	}
	pass := make([]byte, int(passLen[0]))
	if len(pass) > 0 {
		if _, err := io.ReadFull(r, pass); err != nil {
			return "", "", err
		}
	}
	return string(user), string(pass), nil
}

func writeAuthStatus(w io.Writer, ok bool) error {
	status := byte(0x01)
	if ok {
		status = 0x00
	}
	_, err := w.Write([]byte{authVersion, status})
	return err
}

func readRequest(r io.Reader) (request, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return request{}, err
	}
	if header[0] != version5 {
		return request{}, errUnsupportedVersion
	}
	if header[2] != 0x00 {
		return request{}, fmt.Errorf("%w: reserved byte %#x", errMalformedRequest, header[2])
	}
	host, err := readAddress(r, header[3])
	if err != nil {
		return request{}, err
	}
	var portBytes [2]byte
	if _, err := io.ReadFull(r, portBytes[:]); err != nil {
		return request{}, err
	}
	return request{
		cmd:  header[1],
		host: host,
		port: binary.BigEndian.Uint16(portBytes[:]),
	}, nil
}

func readAddress(r io.Reader, atyp byte) (string, error) {
	switch atyp {
	case atypIPv4:
		var raw [4]byte
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return "", err
		}
		return netip.AddrFrom4(raw).String(), nil
	case atypIPv6:
		var raw [16]byte
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return "", err
		}
		return netip.AddrFrom16(raw).String(), nil
	case atypDomain:
		var length [1]byte
		if _, err := io.ReadFull(r, length[:]); err != nil {
			return "", err
		}
		if length[0] == 0 {
			return "", fmt.Errorf("%w: empty domain", errMalformedRequest)
		}
		raw := make([]byte, int(length[0]))
		if _, err := io.ReadFull(r, raw); err != nil {
			return "", err
		}
		return string(raw), nil
	default:
		return "", fmt.Errorf("%w: atyp %#x", errUnsupportedAddress, atyp)
	}
}

func writeReply(w io.Writer, reply byte, addr net.Addr) error {
	atyp := byte(atypIPv4)
	var address []byte
	var port uint16

	if addr != nil {
		if parsed, err := netip.ParseAddrPort(addr.String()); err == nil {
			ip := parsed.Addr().Unmap()
			port = parsed.Port()
			if ip.Is4() {
				octets := ip.As4()
				address = octets[:]
			} else {
				octets := ip.As16()
				atyp = atypIPv6
				address = octets[:]
			}
		}
	}
	if address == nil {
		address = []byte{0, 0, 0, 0}
	}

	response := make([]byte, 0, 4+len(address)+2)
	response = append(response, version5, reply, 0x00, atyp)
	response = append(response, address...)
	response = binary.BigEndian.AppendUint16(response, port)
	_, err := w.Write(response)
	return err
}

func remoteHost(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
