package nbd

const (
	// ASCII 'NBDMAGIC', also known as the INIT_PASSWD
	NBDMAGIC = 0x4e42444d41474943

	// A magic number signs that this is a old style negotiation
	CliservMagic = 0x00420281861253

	// ASCII 'IHAVEOPT' (note different magic number)
	IHAVEOPT = 0x49484156454F5054

	// Magic number returned when by server when doing option haggling.
	ReplyMagic = 0x3e889045565a9
)

const (
	// Server supports the fixed newstyle protocol; Client wants to use fixed newstyle protocol
	HandshakeFlagMaskFixedNewStyle = 0x01

	// Server can not-output extra zeros after handshake; Client wants not to see extra zeros after handshake
	HandshakeFlagMaskNoZeroes = 0x02
)

const (
	TansmissionFlagMaskHasFlags        = 0x0001
	TansmissionFlagMaskReadOnly        = 0x0002
	TansmissionFlagMaskFlush           = 0x0004
	TansmissionFlagMaskFUA             = 0x0008
	TansmissionFlagMaskRotational      = 0x0010
	TansmissionFlagMaskSendTrim        = 0x0020
	TansmissionFlagMaskSendWriteZeroes = 0x0040
	TansmissionFlagMaskSendDF          = 0x0080
	TansmissionFlagMaskCanMultiConn    = 0x0100
	TansmissionFlagMaskSendResize      = 0x0200
	TansmissionFlagMaskSendCache       = 0x0400
	TansmissionFlagMaskFastZero        = 0x0800
)

const (
	// Defined in <linux/fs.h>:
	BLKROSET = 4701

	// Defined in <linux/nbd.h>:
	NBD_SET_SOCK        = 43776
	NBD_SET_BLKSIZE     = 43777
	NBD_SET_SIZE        = 43778
	NBD_DO_IT           = 43779
	NBD_CLEAR_SOCK      = 43780
	NBD_CLEAR_QUE       = 43781
	NBD_PRINT_DEBUG     = 43782
	NBD_SET_SIZE_BLOCKS = 43783
	NBD_DISCONNECT      = 43784
	NBD_SET_TIMEOUT     = 43785
	NBD_SET_FLAGS       = 43786
)

type Option uint32

const (
	OptExportName      Option = 1
	OptAbort                  = 2
	OptList                   = 3
	OptStartTLS               = 5
	OptInfo                   = 6
	OptGo                     = 7
	OptStructuredReply        = 8
	OptListMetaContext        = 9
	OptSetMetaContext         = 9
)

type OptionReply uint32

const (
	OptRepAck              OptionReply = 1
	OptRepServer                       = 2
	OptRepInfo                         = 3
	OptRepMetaContext                  = 4
	OptRepErrUnsup                     = 0x80000000 + 1
	OptRepErrPolicy                    = 0x80000000 + 2
	OptRepErrInvalid                   = 0x80000000 + 3
	OptRepErrPlatform                  = 0x80000000 + 4
	OptRepErrTLSReqd                   = 0x80000000 + 5
	OptRepErrUnknown                   = 0x80000000 + 6
	OptRepErrShutdown                  = 0x80000000 + 7
	OptRepErrBlockSizeReqd             = 0x80000000 + 8
	OptRepErrTooBig                    = 0x80000000 + 9
)

type InfoType uint16

const (
	InfoExport      InfoType = 0
	InfoName                 = 1
	InfoDescription          = 2
	InfoBlockSize            = 3
	InfoNone                 = 0xFFFF
)

const (
	DefaultBlockSize = 4096
)