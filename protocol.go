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

func checkPidAndLength(buf *bytes.Buffer, expectedId PacketID, expectedLength int, atLeast bool) error {
	if !atLeast && buf.Len() != expectedLength {
		return EncodeDecodeError{"Length of buffer mismatch (should be " + strconv.Itoa(expectedLength) + " bytes)"}
	} else if buf.Len() < expectedLength {
		return EncodeDecodeError{"Length of buffer mismatch (should be at least" + strconv.Itoa(expectedLength) + " bytes)"}
	}

	pid, _ := buf.ReadByte()
	if pid != byte(expectedId) {
		return EncodeDecodeError{"Packet ID Mismatch, should be " + strconv.Itoa(int(expectedId)) + " (got " + strconv.Itoa(int(pid)) + ")"}
	}

	return nil
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

const (
	connectionRequestLength  = 10
	connectionAcceptedLength = 9
	connectionRejectedLength = 2
	channelOperationLength   = 3

	acknowledgedBaseLength = 6
	containerBaseLength    = 4
)

// ID: 0x01 - Send from client to server to try to open a connection
type ConnectionRequestPacket struct {
	ClientId        uint64
	ProtocolVersion uint8
}

func (packet *ConnectionRequestPacket) Encode() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Grow(connectionRequestLength)
	buf.WriteByte(byte(ConnectionRequestPid))

	WriteUInt64(&buf, packet.ClientId)
	buf.WriteByte(packet.ProtocolVersion)
	return buf.Bytes(), nil
}

func (packet *ConnectionRequestPacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := checkPidAndLength(buf, ConnectionRequestPid, connectionRequestLength, false)
	if err != nil {
		return err
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

func (packet *ConnectionAcceptedPacket) Encode() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Grow(connectionRejectedLength)
	buf.WriteByte(byte(ConnectionAcceptedPid))

	WriteUInt64(&buf, packet.ServerId)
	return buf.Bytes(), nil
}

func (packet *ConnectionAcceptedPacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := checkPidAndLength(buf, ConnectionAcceptedPid, connectionAcceptedLength, false)
	if err != nil {
		return err
	}

	serverIdBytes, _ := buf.ReadBytes(8)
	packet.ServerId = binary.BigEndian.Uint64(serverIdBytes)

	return nil
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

func (packet *ConnectionRejectedPacket) Encode() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Grow(connectionRejectedLength)

	buf.WriteByte(byte(ConnectionRejectedPid))
	buf.WriteByte(byte(packet.reason))

	return buf.Bytes(), nil
}

func (packet *ConnectionRejectedPacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := checkPidAndLength(buf, ConnectionRejectedPid, connectionRejectedLength, false)
	if err != nil {
		return err
	}

	reason, _ := buf.ReadByte()
	packet.reason = ConnectionRejectedReason(reason)

	return nil
}

// ID 0x0A - Packet sent whenever a packet with reliable flag was received. Can acknowledge multiple reliable packets
type AcknowledgePacket struct {
	// Prefixed by a uint8 of amount of ids (MAX 255!)
	sequenceIds []uint32
}

func (packet *AcknowledgePacket) Encode() ([]byte, error) {
	if len(packet.sequenceIds) < 1 {
		return nil, EncodeDecodeError{"sequenceIds array is empty!"}
	}

	buf := bytes.Buffer{}
	buf.Grow(acknowledgedBaseLength + ((len(packet.sequenceIds) - 1) * 4))

	buf.WriteByte(byte(AcknowledgedPid))
	buf.WriteByte(byte(len(packet.sequenceIds)))
	for _, id := range packet.sequenceIds {
		WriteUInt32(&buf, id)
	}

	return buf.Bytes(), nil
}

func (packet *AcknowledgePacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := checkPidAndLength(buf, AcknowledgedPid, acknowledgedBaseLength, true)
	if err != nil {
		return err
	}

	sequenceCount, _ := buf.ReadByte() // Read the amount of sequence IDs in the packet
	if sequenceCount < 1 {
		return nil
	}

	packet.sequenceIds = make([]uint32, sequenceCount)
	for i := 0; i < int(sequenceCount); i++ {
		sequenceBytes, _ := buf.ReadBytes(4)
		packet.sequenceIds[i] = binary.BigEndian.Uint32(sequenceBytes)
	}

	return nil
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

func (packet *ChannelOpPacket) Encode() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Grow(channelOperationLength)

	buf.WriteByte(byte(ChannelOperationPid))
	buf.WriteByte(packet.channel)

	return buf.Bytes(), nil
}

func (packet *ChannelOpPacket) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := checkPidAndLength(buf, ChannelOperationPid, channelOperationLength, false)
	if err != nil {
		return err
	}

	channel, _ := buf.ReadByte()
	packet.channel = channel

	return nil
}

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
