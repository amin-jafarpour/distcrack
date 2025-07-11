package distnet

import (
	"fmt"
	"net"
	"encoding/gob"
	"bytes"
	"errors"
	"github.com/google/uuid"
	"io"
	"encoding/binary"
)

// Returns a unqiue session ID every time it is called. 
func NewSessionID(token string) string{
	return token + "-" + uuid.New().String()
}

// Serializes any type to []byte.
func Serialize[T any](item T) ([]byte, error) {
	var buffer bytes.Buffer // A Buffer is a variable-sized buffer of bytes.
	encoder := gob.NewEncoder(&buffer) // NewEncoder returns a new encoder that will transmit.
	err := encoder.Encode(item) // Encode transmits the data item.
	// If failed to encode, return zero value and the error.
	if err != nil{
		fmt.Println(err)
		return nil, err 
	}
	return buffer.Bytes(), nil // Convert back to []byte.
}

// Deserializes []byte back to any type.
func Deserialize[T any](itemBin []byte) (T, error) {
	var zeroVal T
	var item T // Variable of the type.
	// NewBuffer creates and initializes a new Buffer using the argument as its initial contents.
	buffer := bytes.NewBuffer(itemBin)
	decoder := gob.NewDecoder(buffer) // NewDecoder returns a new decoder that reads.
	err := decoder.Decode(&item) // Decode reads the next value into the argument.
	// If failed to decode, return zero value and the error.
	if err != nil{
		fmt.Println(err)
		return zeroVal, err
	}
	return item, err
}

// RecvGenPkt receives a Generic Packet from the connection.
func RecvGenPkt(conn net.Conn) (GenPkt, error) {
	var zeroGenPkt GenPkt

	// Read 4 bytes from the connection that represent the length.
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		fmt.Println("Error reading message length:", err)
		return zeroGenPkt, err
	}
	pktLen := int(binary.BigEndian.Uint32(lenBuf))

	// Read the actual serialized packet bytes.
	genPktBytes := make([]byte, pktLen)
	if _, err := io.ReadFull(conn, genPktBytes); err != nil {
		fmt.Println("Error reading full message:", err)
		return zeroGenPkt, err
	}

	// Deserialize the packet.
	genPkt, err := Deserialize[GenPkt](genPktBytes)
	if err != nil {
		fmt.Println("Error deserializing message:", err)
		return zeroGenPkt, err
	}
	return genPkt, nil
}

// sendGenPkt sends a Generic Packet into the connection.
func sendGenPkt(conn net.Conn, genPkt GenPkt) (int, error) {
	// Serialize the Generic Packet.
	genPktBytes, err := Serialize(genPkt)
	if err != nil {
		return -1, err
	}
	pktLen := len(genPktBytes)

	// Create a 4-byte buffer for the length and write the length using BigEndian.
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(pktLen))
	if _, err := conn.Write(lenBuf); err != nil {
		fmt.Println("Error sending message length:", err)
		return -1, err
	}

	// Write the serialized packet.
	lenWritten, err := conn.Write(genPktBytes)
	if err != nil || lenWritten != pktLen {
		fmt.Println("Error sending message:", err)
		return -1, err
	}
	return lenWritten, nil
}

// Returns the payload enclosed within a Generic packet as the packet type specified. 
func FetchPayload[T any](genPkt GenPkt, payloadType string) (T, error){
	var typeZeroVal T // Zerp value to return in case of an error.
	// If packet type specified is not the same as the packet type enclosed in
	// the Generic Packet, return zero value and the error.
	if genPkt.PayloadPktType != payloadType{
		return typeZeroVal, errors.New("Invalid payload type.") 
	}
	// Convert the payload bytes to the packet type specified. 
	payloadPkt, err := Deserialize[T](genPkt.PktBytes)
	// If failed to deserialize to the type specified, return zero value and the error.
	if err != nil{
		fmt.Println(err)
		return typeZeroVal, err 
	}
	// Return the payload as the packet type specified.
	return payloadPkt, nil
}


// Extracts packt type indicated by `TypePktStr` from the connection.
func RecvTypePkt[T any](conn net.Conn, TypePktStr string) (T, error){
	var genPkt GenPkt // Stores the generic packet to receive.
	// Default type value of T that will be returned in case of an error.
	var pktTypeZero T 
	// If encountered an error receiving a generic packet, 
	// return T zero value and the error.
	if pkt, err  := RecvGenPkt(conn); err != nil{
		return pktTypeZero, err 
	} else{
		// Otherwise, store the generic packet.
		genPkt = pkt 
	}
	// If encountered an error converting the generic 
	// packet to packet of type T,  return T zero value and the error. 
	if pkt, err := FetchPayload[T](genPkt, TypePktStr); err != nil{
		return pktTypeZero, err 
	} else {
		// Otherwise return the packet of type T and nil for the 
		// error part. 
		return pkt, nil 
	}
}

func IPAddrs() map[string]map[string][]string{
	// Create the nested map: Interface Name -> {"IPv4": [], "IPv6": []}
	netMap := make(map[string]map[string][]string)
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
		return nil 
	}
	// Iterate over interfaces
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()

		// Initialize the inner map for each interface
		netMap[iface.Name] = map[string][]string{
			"IPv4": {},
			"IPv6": {},
		}
		// Iterate over addresses
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.To4() != nil {
					netMap[iface.Name]["IPv4"] = append(netMap[iface.Name]["IPv4"], ipNet.IP.String())
				} else {
					netMap[iface.Name]["IPv6"] = append(netMap[iface.Name]["IPv6"], ipNet.IP.String())
				}
			}
		}
	}
	return netMap
}
