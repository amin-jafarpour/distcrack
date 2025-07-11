package distnet

import (
		"encoding/gob"
		"reflect"
		"fmt"
)

// Constant strings of the packet type names
const (
	GenPktStr string = "GenPkt"
	PeerHelloPktStr string = "PeerHelloPkt"
	CoordHelloPktStr string = "CoordHelloPkt"
	CoordTaskPktStr string = "CoordTaskPkt"
	PeerCkPtkStr string = "PeerCkPtk"
	PeerNewTaskPktStr string = "PeerNewTaskPkt"
	PeerDiscPktStr string = "PeerDiscPkt"
	PeerSuccessPktStr string = "PeerSuccessPkt"
	CoordDiscPktStr string = "CoordDiscPkt"
	CoordProbPktStr string = "CoordProbPkt"
	PeerAlivePktStr string = "PeerAlivePkt"
	PeerProbPktStr string = "PeerProbPkt"
	CoordAlivePktStr string = "CoordAlivePkt"
) 


// This is a global Generic packet variable. 
var GenPktZero GenPkt 

// Generic Packet that will enclose other packets. 
type GenPkt struct{
	PktType string // Name of this struct as a string. 
	PayloadPktType string // The type of the packet enclosed in this Generic Packet.
	PktBytes []byte // The packet enclosed in bytes format.

}
// Returns a Generic Packet.
func MakeGenPkt[T any](pkt T) (GenPkt, error){
	// Default GenPkt Packet type returned in case of an error as the default value. 
	// Get the payload packet type name dynamically.
	payloadPktName := reflect.TypeOf(pkt).Name()
	// Serialize the packet to be enclosed. 
	PktBytes, err := Serialize(pkt) 
	// If an issue serializing, return default type value and the error.
	if err != nil{
		fmt.Println(err)
		return GenPktZero, err 
	}
	// Return the Generic Packet encolsing the payload packet in bytes format. 
	return GenPkt{
		PktType: GenPktStr, 
		PayloadPktType: payloadPktName,
		PktBytes: PktBytes,
	}, nil 
}

// Packet Peer sends to init connection with the coordinator.
type PeerHelloPkt struct{
	PktType string // Name of this struct as a string.
	IPv4 string // IPv4 address of the peer.
	IPv6 string // IPv6 address of the peer if available. Optional field. 
	MAC string // MAC address of the peer. 
}
// Returns a Peer Hello Packet enclosed in a Generic Packet.
func MakePeerHelloPkt(ipv4, ipv6, mac string) (GenPkt, error){
	peerHelloPkt := PeerHelloPkt{
		PktType: PeerHelloPktStr,
		IPv4: ipv4,
		IPv6: ipv6,
		MAC:  mac,
	}
	genPkt, err := MakeGenPkt[PeerHelloPkt](peerHelloPkt)
	if err != nil{
		return GenPktZero, err 
	}
	return genPkt, nil 
}

// Packet coordinator sends to acknowledge Peer Hello Packet.
type CoordHelloPkt struct{
	PktType string // Name of this struct as a string.  
	// Session ID is guaranteed to be unique for every session.  
	SessionID string 
	Data string 
}
// Returns a Coordinator Hello Packet enclosed in a Generic Packet.
func MakeCoordHelloPkt(sessionID, data string) (GenPkt, error){
	coordHelloPkt := CoordHelloPkt{
		PktType: CoordHelloPktStr, 
		SessionID: sessionID,
		Data: data,
	}
	genPkt, err := MakeGenPkt[CoordHelloPkt](coordHelloPkt)
	if err != nil{
		return GenPktZero, err
	}
	return genPkt, nil 
}

// Packet Coordinator sends to asign a task to the peer. 
type CoordTaskPkt struct{
	PktType string // Name of this struct as a string. 
	SessionID string 
	Ckpt Checkpoint 
}
// Returns a Coordinator Task Packet enclosed in a Generic Packet.
func MakeCoordTaskPkt(sessionID string, Ckpt Checkpoint) (GenPkt, error){
	coordTaskPkt := CoordTaskPkt{
		PktType: CoordTaskPktStr, 
		SessionID: sessionID,
		Ckpt: Ckpt,
	}
	genPkt, err := MakeGenPkt[CoordTaskPkt](coordTaskPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Checkpoint packet peer sends to the coordinator.  
type PeerCkPtk struct{
	PktType string // Name of this struct as a string.
	Ckpt Checkpoint 
	SessionID string 
}
// Returns a Peer Checkpoint Packet enclosed in a Generic Packet.
func MakePeerCkPtk(sessionID string, Ckpt Checkpoint ) (GenPkt, error){ //  ckPtVal int
	peerCkPtk := PeerCkPtk{
		PktType:  PeerCkPtkStr,
		Ckpt: Ckpt,  
		SessionID: sessionID,
	}
	genPkt, err := MakeGenPkt[PeerCkPtk](peerCkPtk)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet peer sends when peer has completely finished its assigned task and wants a new task,
// or when the connection between the coordinator and the peer has just been established and 
// the peer wishes to take on a task. 
type PeerNewTaskPkt struct{
	PktType string  // Name of this struct as a string.
	SessionID string 
}
// Returns a Peer New Task Packet enclosed in a Generic Packet.
func MakePeerNewTaskPkt(sessionID string) (GenPkt, error){
	peerNewTaskPkt := PeerNewTaskPkt{
		PktType: PeerNewTaskPktStr,
		SessionID: sessionID, 
	}
	genPkt, err := MakeGenPkt[PeerNewTaskPkt](peerNewTaskPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet peer sends when wishing to disconnect from the coordinator.
type PeerDiscPkt struct{
	PktType string // Name of this struct as a string.
	Ckpt Checkpoint 
	SessionID string 
}
// Returns a Peer Disconnect Packet enclosed in a Generic Packet.
func MakePeerDiscPkt(sessionID string, Ckpt Checkpoint) (GenPkt, error){ 
	peerDiscPkt := PeerDiscPkt{
		PktType: PeerDiscPktStr,  
		Ckpt: Ckpt,  
		SessionID: sessionID,  
	}
	genPkt, err := MakeGenPkt[PeerDiscPkt](peerDiscPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet a peer sends when it has made a significant break thorough, which needs to be shared 
// with the coordinator immediately.  
type PeerSuccessPkt struct{
	PktType string // Name of this struct as a string.
	SuccessVal string // The value or message that must be shared with the coordinator immediately. 
	SessionID string 
}
// Returns a Peer Success Packet enclosed in a Generic Packet.
func MakePeerSuccessPkt(sessionID, successVal string) (GenPkt, error){
	peerSuccessPkt := PeerSuccessPkt{
		PktType: PeerSuccessPktStr, 
		SuccessVal: successVal, 
		SessionID: sessionID,  
	}
	genPkt, err := MakeGenPkt[PeerSuccessPkt](peerSuccessPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet coordinator sends when wishing to disconnec from a peer. 
type CoordDiscPkt struct{
	PktType string // Name of this struct as a string.
	SessionID string
}
// Returns a Coordinator Disconect Packet enclosed in a Generic Packet.
func MakeCoordDiscPkt(sessionID string) (GenPkt, error){
	coordDiscPkt := CoordDiscPkt{
		PktType: CoordDiscPktStr,  
		SessionID: sessionID,
	}
	genPkt, err := MakeGenPkt[CoordDiscPkt](coordDiscPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet Coordinator sends when trying to see if a peer is alive
type CoordProbPkt struct {
	PktType string // Name of this struct as a string.
	SessionID string 
}
// Returns a Coordinator Prob Packet enclosed in a Generic Packet.
func MakeCoordProbPkt(sessionID string) (GenPkt, error){
	coordProbPkt := CoordProbPkt{
		PktType: CoordProbPktStr,  
		SessionID: sessionID,
	}
	genPkt, err := MakeGenPkt[CoordProbPkt](coordProbPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet peer sends to tell coordinator that it is alive. 
type PeerAlivePkt struct{
	PktType string // Name of this struct as a string.
	SessionID string 
}
// Returns a Peer Alive Packet enclosed in a Generic Packet.
func MakePeerAlivePkt(sessionID string) (GenPkt, error){
	peerAlivePkt := PeerAlivePkt{
		PktType: PeerAlivePktStr,  
		SessionID: sessionID,
	} 
	genPkt, err := MakeGenPkt[PeerAlivePkt](peerAlivePkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet peer sends to check whether coordinator is alive. 
type PeerProbPkt struct{
	PktType string // Name of this struct as a string.
	SessionID string 
}
// Returns a Peer Prob Packet enclosed in a Generic Packet.
func MakePeerProbPkt(sessionID string) (GenPkt, error){
	peerProbPkt := PeerProbPkt{
		PktType: PeerProbPktStr,  
		SessionID: sessionID,
	}
	genPkt, err := MakeGenPkt[PeerProbPkt](peerProbPkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}
// Packet Coordinator sends to tell peer that it is alive. 
type CoordAlivePkt struct{
	PktType string // Name of this struct as a string.
	SessionID string 
}
// Returns a Coordinator Alive Packet enclosed in a Generic Packet.
func MakeCoordAlivePkt(sessionID string) (GenPkt, error){
	coordAlivePkt := CoordAlivePkt{
		PktType: CoordAlivePktStr,  
		SessionID: sessionID,
	}
	genPkt, err := MakeGenPkt[CoordAlivePkt](coordAlivePkt)
    if err != nil{
        return GenPktZero, err
    }
    return genPkt, nil 
}

// Regiser all Structs declared so they may be serialized. 
func Init(){
	gob.Register(Checkpoint{})
	gob.Register(GenPkt{})
	gob.Register(PeerHelloPkt{})
	gob.Register(CoordHelloPkt{})
	gob.Register(CoordTaskPkt{})
	gob.Register(PeerCkPtk{})
	gob.Register(PeerNewTaskPkt{})
	gob.Register(PeerDiscPkt{})
	gob.Register(PeerSuccessPkt{})
	gob.Register(CoordDiscPkt{})
	gob.Register(CoordProbPkt{})
	gob.Register(PeerAlivePkt{})
	gob.Register(PeerProbPkt{})
	gob.Register(CoordAlivePkt{})
}