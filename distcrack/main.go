package main

import (
	"distcrack/distnet"
	"distcrack/hashcrack"
	"fmt"
	"flag"
	"os"
	"runtime"
	"strconv"
)

func main(){


	NodeType := flag.String("type", "coord", "Type of code. Can be either coord or peer.")
	TimeoutSeconds := flag.Int("timeout", 10, "Timeout in seconds")
	MaxTimeoutCounter := flag.Int("attempt", 10, "Number of times connection can timeout after handshake before disconnecting.")
	Data := flag.String("hash", " ", "Hash to be decrypted")
	PartitionSize := flag.Int("work-size", 10000, "Number of password attempts per job packet.")
	InclusiveMaxPasswdLen := flag.Int("maxlen", 16, "Max length of password.")
	IPAddr := flag.String("server", "127.0.0.1", "IPv4")
	PortInt := flag.Int("port", 5000, "network port number")
	ThreadNumber := flag.Int("thread", runtime.NumCPU(), "Number of threads to use.")
	_ = flag.Int("checkpoint", 1, "How often to send a checkpoint.")

	flag.Parse()
	

	printCmdErr := func(msg string){
			fmt.Fprintln(os.Stderr, "Error " + msg)
			flag.Usage()
			os.Exit(1)
	}

	if *NodeType != "coord" && *NodeType != "peer"{
		printCmdErr("-type has to be either coord or peer")
	}

	if *TimeoutSeconds < 1{
		printCmdErr("-timeout has to be at least 1 seconds")
	} 
	
	if *MaxTimeoutCounter < 1{
		printCmdErr("-attempt has to be at least 1 seconds")
	}

	if realHashTokens := hashcrack.SplitHash(*Data); *NodeType == "coord" && len(realHashTokens) != 2{
		printCmdErr("Hash value given is invalid: " + *Data)
	} else if  *NodeType == "coord" && !hashcrack.IsValidSalt(realHashTokens[0]){
		printCmdErr("Hash type unsupported.")
	} 

	if *PartitionSize < 100{
		printCmdErr("-work-size has to be at least 100")
	}

	if *InclusiveMaxPasswdLen < 1{
		printCmdErr("-maxlen has to be at least 1")
	}

	if *IPAddr == ""{
		printCmdErr("-server is invalid")
	}

	if *PortInt < 0 || *PortInt > 65535{
		printCmdErr("-port must be in range 0 to 65535")
	}
	Port := strconv.Itoa(*PortInt)

	if *ThreadNumber < 1{
		printCmdErr("thread")
	}

	distnet.Init()

	if *NodeType == "coord"{
		params := distnet.CoordParams{
			TimeoutSeconds:  *TimeoutSeconds,
			MaxTimeoutCounter: *MaxTimeoutCounter,
			Data: *Data, 
			PartitionSize: *PartitionSize,
			InclusiveMaxPasswdLen: *InclusiveMaxPasswdLen,  
			IPAddr: *IPAddr,  
			Port: Port,  
		}

		distnet.Listen(params)

	} else if *NodeType == "peer"{


		params := distnet.PeerParams {
			TimeoutSeconds:  *TimeoutSeconds,
			MaxTimeoutCounter: *MaxTimeoutCounter,
			IPAddr:  *IPAddr,
			Port: Port,
			ThreadNumber: *ThreadNumber, 
		
		}

		distnet.Connect(params)

	}
}









