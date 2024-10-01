package pkg

import (
	"math/bits"
	//"fmt"
)

type Packet [8]byte

type PacketType uint8

const (
	PacketTypeData PacketType = 0
	PacketTypeError PacketType = 1
	PacketTypeWrite PacketType = 2
	PacketTypeRead PacketType = 3
)

func (p Packet) Type() PacketType {
	return PacketType(p[0] & 3)
}

func (p Packet) Chip() uint8 {
	return (p[0] >> 2) | (p[1] << 6)
}

func (p Packet) Channel() uint8 {
	return p[1] >> 2
}

func (p Packet) Timestamp() uint32 {
	return uint32(p[2]) |
		(uint32(p[3]) << 8) |
		(uint32(p[4]) << 16) |
		(uint32((p[5] & 0x7F)) << 24)
}

func (p Packet) First() bool {
	return p[5] >> 7 == 1
}

func (p Packet) Data() uint8 {
	return p[6]
}

func (p Packet) TrigType() uint8 {
	return p[7] & 3
}

func (p Packet) LocalFifoFlags() uint8 {
	return (p[7] >> 2) & 3
}

func (p Packet) SharedFifoFlags() uint8 {
	return (p[7] >> 4) & 3
}

func (p Packet) Downstream() bool {
	return (p[7] >> 6) & 1 == 1
}

func (p Packet) ParityBit() uint8 {
	return p[7] >> 7
}

func (p Packet) ValidParity() bool {
	onesCount := 0
	for i, b := range p {
		if i == 7 {
			onesCount = onesCount + bits.OnesCount(uint(b & 0x7F)) // Skip parity bit
		} else {
			onesCount = onesCount + bits.OnesCount(uint(b))
		}
	}
	return (1 - (onesCount % 2)) == int(p.ParityBit())
}
