package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"unsafe"

	"github.com/sequix/nbd/pkg/bytesutil"
	"github.com/sequix/nbd/pkg/nbd"
)

// todo tls
// todo meta context
// todo timeout
// todo oldstyle & newstlye negotiation
// todo reconnect if link is down

type Client struct {
	conn              net.Conn
	style             NegotiationStyle
	forceStyle        bool
	exportSizeInBytes uint64
	structuredReply   bool
}

type NegotiationStyle string

const (
	NegotiationStyleOld      NegotiationStyle = "oldstyle"
	NegotiationStyleNew                       = "newstyle"
	NegotiationStyleFixedNew                  = "fixed-newstyle"
)

type ExportInfo struct {
	TransmissionFlags
	BlockSizes

	Export      string
	SizeInBytes uint64
	Description string
}

type TransmissionFlags struct {
	RawTransmissionFlags      uint16
	ReadOnly                  bool
	SupportFlush              bool
	SupportFUA                bool
	Rotational                bool // ?
	SupportTrim               bool // ?
	SupportWriteZeroes        bool
	NoFragmentStructuredReply bool
	MultiConn                 bool
	SupportResize             bool // ?
	SupportCache              bool
	FastZeros                 bool
}

type BlockSizes struct {
	MinBlockSize       uint32
	MaxBlockSize       uint32
	PreferredBlockSize uint32
}

type optReply struct {
	opt     nbd.Option
	rep     nbd.OptionReply
	payload []byte
}

// Create a client based on fixed newstyle negotiation.
func New(network, addr string, opts ...Option) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn:       conn,
		style:      NegotiationStyleFixedNew,
		forceStyle: false,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if err := c.ping(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) read(dst []byte) (n int, err error) {
	return io.ReadFull(c.conn, dst)
}

func (c *Client) write(src []byte) (n int, err error) {
	return c.conn.Write(src)
}

func (c *Client) Close() error {
	panic("todo")
	// NBD_OPT_ABORT
}

var (
	bigEndian = binary.BigEndian
)

func (c *Client) ping() error {
	buf := make([]byte, 8)
	if n, err := c.read(buf[0:8]); err != nil {
		return fmt.Errorf("recving nbdmagic of %d bytes: %w", n, err)
	}
	nbdmagic := bigEndian.Uint64(buf[0:8])
	if nbdmagic != nbd.NBDMAGIC {
		return ErrNotValidNBDServer
	}

	if n, err := c.read(buf[0:8]); err != nil {
		return fmt.Errorf("recving magicnumber of %d bytes: %w", n, err)
	}
	magicnumber := bigEndian.Uint64(buf[0:8])
	switch magicnumber {
	case nbd.CliservMagic:
		if c.forceStyle && c.style != NegotiationStyleOld {
			return fmt.Errorf("%w: want %q, got %q", ErrNegotiationStyleMismatch, c.style, NegotiationStyleOld)
		}
		c.style = NegotiationStyleOld
		return c.pingOldStyle(buf)
	case nbd.IHAVEOPT:
		return c.pingNewStyleOrFixedNewStyle()
	default:
		return fmt.Errorf("%w: %x", ErrUnknownMagicNumber, magicnumber)
	}
}

func (c *Client) pingOldStyle(buf []byte) error {
	panic("todo")
}

func (c *Client) pingNewStyleOrFixedNewStyle() error {
	buf := make([]byte, 4)
	if _, err := c.read(buf[0:2]); err != nil {
		return fmt.Errorf("recving handshake flags: %w", err)
	}
	clientHandshakeFlags := uint32(0)
	handshakeFlags := bigEndian.Uint16(buf[0:2])
	if handshakeFlags&nbd.HandshakeFlagMaskFixedNewStyle != 0 {
		c.style = NegotiationStyleFixedNew
		clientHandshakeFlags |= nbd.HandshakeFlagMaskFixedNewStyle
		if c.forceStyle && c.style != NegotiationStyleFixedNew {
			return fmt.Errorf("%w: want %q, got %q", ErrNegotiationStyleMismatch, c.style,
				NegotiationStyleFixedNew)
		}
	} else {
		c.style = NegotiationStyleNew
		clientHandshakeFlags &= ^uint32(nbd.HandshakeFlagMaskFixedNewStyle)
		if c.forceStyle && c.style != NegotiationStyleNew {
			return fmt.Errorf("%w: want %q, got %q", ErrNegotiationStyleMismatch, c.style,
				NegotiationStyleNew)
		}
	}
	if handshakeFlags&nbd.HandshakeFlagMaskNoZeroes != 0 {
		clientHandshakeFlags |= nbd.HandshakeFlagMaskNoZeroes
	}
	bigEndian.PutUint32(buf[0:4], clientHandshakeFlags)
	if n, err := c.write(buf[0:4]); err != nil {
		return fmt.Errorf("writing client handshake flags of %d bytes: %w", n, err)
	}
	return nil
}

func (c *Client) sendOpt(opt nbd.Option, data []byte) error {
	buf := make([]byte, 16)
	bigEndian.PutUint64(buf[0:8], nbd.IHAVEOPT)
	bigEndian.PutUint32(buf[8:12], uint32(opt))
	bigEndian.PutUint32(buf[12:16], uint32(len(data)))
	if n, err := c.write(buf[0:16]); err != nil {
		return fmt.Errorf("writing opt of %d bytes: %w", n, err)
	}
	if len(data) > 0 {
		if n, err := c.write(data); err != nil {
			return fmt.Errorf("writing data of %d bytes: %w", n, err)
		}
	}
	return nil
}

func (c *Client) recvOptReply(opt nbd.Option) (nbd.OptionReply, []byte, error) {
	buf := make([]byte, 20)
	if n, err := c.read(buf[0:20]); err != nil {
		return 0, nil, fmt.Errorf("recving opt reply of %d bytes: %w", n, err)
	}
	if replyMagic := bigEndian.Uint64(buf[0:8]); replyMagic != nbd.ReplyMagic {
		return 0, nil, fmt.Errorf("invalid reply magic 0x%X", replyMagic)
	}
	if gotOpt := nbd.Option(bigEndian.Uint32(buf[8:12])); gotOpt != opt {
		return 0, nil, fmt.Errorf("%w: want %d, got %d", ErrOptReplyTypeMismatch, opt, gotOpt)
	}
	replyType := nbd.OptionReply(nbd.OptionReply(bigEndian.Uint32(buf[12:16])))
	replyLength := int(bigEndian.Uint32(buf[16:20]))
	var dst []byte
	if replyLength > 0 {
		dst = make([]byte, replyLength)
		if n, err := c.read(dst); err != nil {
			return 0, nil, fmt.Errorf("recving reply content of %d bytes: %w", n, err)
		}
	}
	return replyType, dst, nil
}

func (c *Client) List() ([]string, error) {
	panic("todo")
}

var allInfoTypes = []nbd.InfoType{
	nbd.InfoExport,
	nbd.InfoName,
	nbd.InfoDescription,
	nbd.InfoBlockSize,
}

func (c *Client) Info(dst *ExportInfo, export string, infoRequires ...nbd.InfoType) (info *ExportInfo, err error) {
	if len(infoRequires) == 0 {
		infoRequires = allInfoTypes
	}

	payload := c.buildPayloadForOptInfoAndOptGo(export, infoRequires)
	if err := c.sendOpt(nbd.OptInfo, payload); err != nil {
		return dst, fmt.Errorf("sending opt NBD_OPT_INFO: %w", err)
	}

	for {
		replyType, payload, err := c.recvOptReply(nbd.OptInfo)
		if err != nil {
			return dst, fmt.Errorf("recving opt reply: %w", err)
		}
		_, info2, err := c.parseInfoReply(replyType, payload)
		if err != nil {
			if err == ErrOptInfoRepAck {
				break
			}
			return dst, fmt.Errorf("parsing info reply: %w", err)
		}
		info = info2
	}
	return
}

func (c *Client) buildPayloadForOptInfoAndOptGo(export string, infoTypes []nbd.InfoType) []byte {
	var (
		ne  = len(export)
		nit = len(infoTypes)
		dst = make([]byte, 0, 4+ne+2+nit*2)
		buf = make([]byte, 40)
	)
	bigEndian.PutUint32(buf[0:4], uint32(len(export)))
	dst = append(dst, buf[0:4]...)
	dst = append(dst, bytesutil.ToUnsafeBytes(export)...)
	bigEndian.PutUint16(buf[0:2], uint16(len(infoTypes)))
	dst = append(dst, buf[0:2]...)
	for _, it := range infoTypes {
		bigEndian.PutUint16(buf[0:2], uint16(it))
		dst = append(dst, buf[0:2]...)
	}
	return dst
}

func (c *Client) parseInfoReply(replyType nbd.OptionReply, payload []byte) (infoType nbd.InfoType, info *ExportInfo, err error) {
	switch replyType {
	case nbd.OptRepInfo:
		if infoType, info, err = c.parseInfoPayload(payload); err != nil {
			return nbd.InfoNone, info, fmt.Errorf("parsing info payload: %w", err)
		}
		return infoType, info, nil
	case nbd.OptRepAck:
		return nbd.InfoNone, nil, ErrOptInfoRepAck
	case nbd.OptRepErrTLSReqd:
		return nbd.InfoNone, nil, fmt.Errorf("%w: payload: %v", ErrTLSRequired, payload)
	case nbd.OptRepErrBlockSizeReqd:
		return nbd.InfoNone, nil, fmt.Errorf("%w: payload: %v", ErrBlockSizeRequired, payload)
	case nbd.OptRepErrUnknown:
		return nbd.InfoNone, nil, fmt.Errorf("%w: payload: %v", ErrUnknown, payload)
	default:
		return nbd.InfoNone, nil, fmt.Errorf("%w: 0x%X", ErrUnknownOptReplyType, replyType)
	}
}

func (c *Client) parseInfoPayload(src []byte) (nbd.InfoType, *ExportInfo, error) {
	info := &ExportInfo{}
	infoType := nbd.InfoType(bigEndian.Uint16(src[0:2]))
	switch infoType {
	case nbd.InfoExport:
		info.SizeInBytes = bigEndian.Uint64(src[2:10])
		info.TransmissionFlags = *c.parseTransmissionFlags(bigEndian.Uint16(src[10:12]))
	case nbd.InfoName:
		info.Export = string(src[2:])
	case nbd.InfoDescription:
		info.Description = string(src[2:])
	case nbd.InfoBlockSize:
		info.MinBlockSize = bigEndian.Uint32(src[2:6])
		info.PreferredBlockSize = bigEndian.Uint32(src[6:10])
		info.MaxBlockSize = bigEndian.Uint32(src[10:14])
	default:
		return 0, nil, fmt.Errorf("%w: %d", ErrUnknownInfoType, infoType)
	}
	return infoType, info, nil
}

func (c *Client) parseTransmissionFlags(flags uint16) *TransmissionFlags {
	fs := &TransmissionFlags{}
	fs.RawTransmissionFlags = flags
	fs.ReadOnly = flags&nbd.TansmissionFlagMaskReadOnly != 0
	fs.SupportFlush = flags&nbd.TansmissionFlagMaskFlush != 0
	fs.SupportFUA = flags&nbd.TansmissionFlagMaskFUA != 0
	fs.Rotational = flags&nbd.TansmissionFlagMaskRotational != 0
	fs.SupportTrim = flags&nbd.TansmissionFlagMaskSendTrim != 0
	fs.SupportWriteZeroes = flags&nbd.TansmissionFlagMaskSendWriteZeroes != 0
	fs.NoFragmentStructuredReply = flags&nbd.TansmissionFlagMaskSendDF != 0
	fs.MultiConn = flags&nbd.TansmissionFlagMaskCanMultiConn != 0
	fs.SupportResize = flags&nbd.TansmissionFlagMaskSendResize != 0
	fs.SupportCache = flags&nbd.TansmissionFlagMaskSendCache != 0
	fs.FastZeros = flags&nbd.TansmissionFlagMaskFastZero != 0
	return fs
}

func (c *Client) Go(dev, export string) error {
	// DO NOT use O_EXCL, which will lead to
	// mount: /dev/nbd1 is already mounted or /mnt busy
	nbdFile, err := os.OpenFile(dev, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open nbd file %q: %w", dev, err)
	}
	defer nbdFile.Close()

	nbdFD := uintptr(nbdFile.Fd())
	if c.style == NegotiationStyleNew {
		return c.goNewStyle(nbdFD, export)
	}
	return c.goFixedNewStyle(nbdFD, export)
}

func (c *Client) goNewStyle(nbdFD uintptr, export string) error {
	panic("todo")
}

func (c *Client) goFixedNewStyle(nbdFD uintptr, export string) error {
	payload := c.buildPayloadForOptInfoAndOptGo(export, nil)
	if err := c.sendOpt(nbd.OptGo, payload); err != nil {
		return fmt.Errorf("sending opt NBD_OPT_GO: %w", err)
	}
	var info *ExportInfo

	for {
		replyType, payload, err := c.recvOptReply(nbd.OptGo)
		if err != nil {
			return fmt.Errorf("recving reply: %w", err)
		}
		_, info2, err := c.parseInfoReply(replyType, payload)
		if err != nil {
			if err == ErrOptInfoRepAck {
				break
			}
			return fmt.Errorf("parsing info reply: %w", err)
		}
		info = info2
	}
	return c.bindAndBlock(uintptr(nbdFD), info)
}

func (c *Client) bindAndBlock(nbdFD uintptr, info *ExportInfo) error {
	var (
		connFile *os.File
		err      error
	)
	switch c.conn.(type) {
	case *net.TCPConn:
		cc := c.conn.(*net.TCPConn)
		connFile, err = cc.File()
	case *net.UnixConn:
		cc := c.conn.(*net.UDPConn)
		connFile, err = cc.File()
	case *net.UDPConn:
		cc := c.conn.(*net.UnixConn)
		connFile, err = cc.File()
	}
	if err != nil {
		return fmt.Errorf("failed getting conn file: %w", err)
	}
	if err := c.setupNFD(nbdFD, info); err != nil {
		return fmt.Errorf("setting up nbd fd: %w", err)
	}
	if err := ioctl(nbdFD, nbd.NBD_SET_SOCK, uintptr(connFile.Fd())); err != nil {
		return fmt.Errorf("failed NBD_SET_SOCK: %w", err)
	}
	if err := ioctl(nbdFD, nbd.NBD_DO_IT, 0); err != nil {
		return fmt.Errorf("transmission phase over, err %w", err)
	}
	return nil
}

func (c *Client) setupNFD(nbdFD uintptr, info *ExportInfo) error {
	blksize := int(info.PreferredBlockSize)
	if blksize == 0 {
		blksize = nbd.DefaultBlockSize
	}
	if err := ioctl(nbdFD, nbd.NBD_SET_BLKSIZE, uintptr(blksize)); err != nil {
		return fmt.Errorf("failed setting NBD_SET_BLKSIZE: %w", err)
	}
	if err := ioctl(nbdFD, nbd.NBD_SET_SIZE_BLOCKS, uintptr(int(info.SizeInBytes)/blksize)); err != nil {
		return fmt.Errorf("failed setting NBD_SET_SIZE_BLOCKS: %w", err)
	}
	if err := ioctl(nbdFD, nbd.NBD_SET_FLAGS, uintptr(info.RawTransmissionFlags)); err != nil {
		return fmt.Errorf("failed setting NBD_SET_FLAGS: %w", err)
	}
	if info.ReadOnly {
		one := 1
		if err := ioctl(nbdFD, nbd.BLKROSET, uintptr(unsafe.Pointer(&one))); err != nil {
			return fmt.Errorf("failed setting BLKROSET: %w", err)
		}
	}
	return nil
}

func ioctl(fd, flag, data uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, flag, data)
	if errno != 0 {
		return errno
	}
	return nil
}
