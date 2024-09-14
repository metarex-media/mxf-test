package mxftest

import (
	"encoding/binary"
	"fmt"

	"github.com/metarex-media/mrx-tool/klv"
)

// PartitionType is a string value of the partition type
type PartitionType string

const (
	// The key used for identifying the header partition
	// it also flags footer partitions.
	Header PartitionType = "header"
	// the key for using identifying essence streams
	Body PartitionType = "essence"
	// The key used for identifying the Generic Body partition
	GenericBody PartitionType = "generickey"
)

// FullNameMask converts an array of 16 bytes into the universal label format of
// "%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x"
//
// Any masked bytes are masked as 7f, and the maskbyte value is
// the integer position in the byte array (starting at 0).
func FullNameMask(key []byte, maskedBytes ...int) string {
	mid := make([]byte, len(key))
	copy(mid, key)

	for _, i := range maskedBytes {
		mid[i] = 0x7f
	}
	return fullName(mid)
}

// fullName returns a slice of 16 bytes as a the universal label format
func fullName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15])
}

// Partition is the layout of an mxf partition
// with type accurate fields (or as close as possible)
type Partition struct {
	Signature         string // Must be, hex: 06 0E 2B 34
	PartitionLength   int    // All but first block size
	MajorVersion      uint16 // Must be, hex: 01 00
	MinorVersion      uint16
	SizeKAG           uint32
	ThisPartition     uint64
	PreviousPartition uint64
	FooterPartition   uint64 // First block size
	HeaderByteCount   uint64
	IndexByteCount    uint64
	IndexSID          uint32
	BodyOffset        uint64
	BodySID           uint32

	// useful information from the partition
	PartitionType     string
	IndexTable        bool
	TotalHeaderLength int
	MetadataStart     int
}

var (
	order = binary.BigEndian
)

const (
	// keys for identifying the type of partition.
	HeaderPartition        = "header"
	BodyPartition          = "body"
	GenericStreamPartition = "genericstreampartition"
	FooterPartition        = "footer"
	RIPPartition           = "rip"
)

// PartitionExtract extracts the partition from a KLV packet
func PartitionExtract(partitionKLV *klv.KLV) Partition {

	var partPack Partition
	// error checking on the length is done before parsing the stream to this function
	// return early to prevent errors
	if len(partitionKLV.Value) < 64 {
		return partPack
	}

	switch partitionKLV.Key[13] {
	case 02:
		// header
		partPack.PartitionType = HeaderPartition
	case 03:
		// body
		if partitionKLV.Key[14] == 17 {
			partPack.PartitionType = GenericStreamPartition
		} else {
			partPack.PartitionType = BodyPartition
		}
	case 04:
		// footer
		partPack.PartitionType = FooterPartition
	default:
		// is nothing
		partPack.PartitionType = "invalid"
		return partPack
	}

	partPack.Signature = fullName(partitionKLV.Key)

	//	packLength, lengthlength := berDecode(ber)
	partPack.PartitionLength = partitionKLV.LengthValue
	partPack.MajorVersion = order.Uint16(partitionKLV.Value[:2:2])
	partPack.MinorVersion = order.Uint16(partitionKLV.Value[2:4:4])
	partPack.SizeKAG = order.Uint32(partitionKLV.Value[4:8:8])
	partPack.ThisPartition = order.Uint64(partitionKLV.Value[8:16:16])
	partPack.PreviousPartition = order.Uint64(partitionKLV.Value[16:24:24])
	partPack.FooterPartition = order.Uint64(partitionKLV.Value[24:32:32])
	partPack.HeaderByteCount = order.Uint64(partitionKLV.Value[32:40:40])
	partPack.IndexByteCount = order.Uint64(partitionKLV.Value[40:48:48])
	partPack.IndexSID = order.Uint32(partitionKLV.Value[48:52:52])
	partPack.BodyOffset = order.Uint64(partitionKLV.Value[52:60:60])
	partPack.BodySID = order.Uint32(partitionKLV.Value[60:64:64])

	kag := int(partPack.SizeKAG)
	headerLength := int(partPack.HeaderByteCount)
	indexLength := int(partPack.IndexByteCount)

	totalLength := kag + headerLength + indexLength
	partPack.MetadataStart = kag

	if kag == 1 {
		packLength := partitionKLV.TotalLength()
		totalLength += packLength - kag
		partPack.MetadataStart = packLength
		// else metadata start is the kag
	}

	if indexLength > 0 {

		// develop and index table body to use
		partPack.IndexTable = true
	}

	partPack.TotalHeaderLength = totalLength

	// partition extract returns the type of partition and the length.
	//	fmt.Println(partPack, "pack here")
	//	fmt.Println(partPack, totalLength, "My partition oack")
	return partPack
}

// RIP is the random index position struct
type RIP struct {
	Sid        uint32
	ByteOffset uint64
}
