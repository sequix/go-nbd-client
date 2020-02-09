package client

import (
	"errors"
)

var (
	ErrNotValidNBDServer = errors.New("not a valid nbd server")
	ErrUnknownMagicNumber = errors.New("unknown magic number")
	ErrNegotiationStyleMismatch = errors.New("negotiation style mismatch")
	ErrOptReplyTypeMismatch = errors.New("opt reply type mismatch")
	ErrUnknownInfoType = errors.New("unknown info type")
	ErrUnknownOptReplyType = errors.New("unknown opt reply type")

	// Errors returned by NBD_OPT_INFO and NBD_OPT_GO.
	ErrTLSRequired = errors.New("tls required")
	ErrBlockSizeRequired = errors.New("block size required")
	ErrUnknown = errors.New("unknown error")

	ErrOptInfoRepAck = errors.New("opt info reply ack")
)