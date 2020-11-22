package ojnet

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

type Packet interface {
	Encode() ([]byte, error)
	Decode([]byte) error
}

type EncodeDecodeError struct {
	Reason string
}

func (err EncodeDecodeError) Error() string {
	return err.Reason
}

const PROTOCOL_VERSION uint8 = 0

type PacketID uint8

const (
	ConnectionRequestPid PacketID = iota
	ConnectionAcceptedPid
	ConnectionRejectedPid
	AcknowledgedPid     PacketID = 0x0A
	ChannelOperationPid PacketID = 0x0B
	ContainerPid        PacketID = 0x0C
)

// ID: 0x01 - Send from client to server to try to open a connection
type ConnectionRequestPacket struct {
	ClientId        uint64
	ProtocolVersion uint8
}

func (packet *ConnectionRequestPacket) Encode() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Grow(10)
	buf.WriteByte(byte(ConnectionRequestPid))

	WriteUInt64(&buf, packet.ClientId)
	buf.WriteByte(packet.ProtocolVersion)
	return buf.Bytes(), nil
}

func (packet *ConnectionRequestPacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	if buf.Len() != 10 {
		return EncodeDecodeError{"Length of buffer mismatch (should be 10 bytes)"}
	}

	pid, _ := buf.ReadByte()
	if pid != byte(ConnectionRequestPid) {
		return EncodeDecodeError{"Packet ID Mismatch, should be " + strconv.Itoa(int(ConnectionRequestPid)) + " (got " + strconv.Itoa(int(pid)) + ")"}
	}

	clientIdBytes, _ := buf.ReadBytes(8)
	packet.ClientId = binary.BigEndian.Uint64(clientIdBytes)
	pVer, _ := buf.ReadByte()
	packet.ProtocolVersion = pVer

	return nil
}

// ID 0x02 - Response from server to accept and open a connection
type ConnectionAcceptedPacket struct {
	ServerId uint64
}

// ID 0x03 - Response from server to reject a connection request
type ConnectionRejectedPacket struct {
	reason ConnectionRejectedReason
}

type ConnectionRejectedReason uint8

const (
	INCOMPATIBLE_PROTOCOL_VER ConnectionRejectedReason = iota
	MAX_CONNECTIONS_REACHED
	RATELIMITED
)

// ID 0x0A - Packet sent whenever a packet with reliable flag was received. Can acknowledge multiple reliable packets
type AcknowledgePacket struct {
	// Prefixed by a uint8 of amount of ids (MAX 255!)
	sequenceIds []uint32
}

// ID 0x0B - Packet used to signal different network operations
// This also is used to close the entire connection, by closing channel 0 (reserved for protocol operations)
type ChannelOpPacket struct {
	operation ChannelOperation // 0: Open Channel, 1: Close Channel, 2: Reset Ordered Ids
	channel   uint8            // Channel ID
}

type ChannelOperation uint8

const (
	OPEN_CHANNEL ChannelOperation = iota
	CLOSE_CHANNEL
	RESET_ORDERED_IDS
)

// ID 0x0C Container Packet acts as a header for all packets sent over the network
type ContainerPacket struct {
	channel uint8

	// These flags all become part of a single byte
	reliable   bool
	ordered    bool
	compressed bool

	// Only present if reliable is true
	sequenceId uint32
	// Only present if ordered is true
	orderedId uint16

	// Prefixed by a uint16 of length of payload
	payload []byte
}
