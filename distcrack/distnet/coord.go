

package distnet 


import (
	"fmt"
	"net"
	"sync"
	"distcrack/hashcrack"
	"time"
	"errors"
	"sync/atomic"
)


type CoordParams struct{
	TimeoutSeconds int 
	MaxTimeoutCounter int 
	Data string 
	PartitionSize int 
	InclusiveMaxPasswdLen int 
	IPAddr string 
	Port string  
}


var globalSuccessCond atomic.Bool
var globalSuccessFound atomic.Value
 

func Listen(params CoordParams){

	listener, err := net.Listen("tcp", params.IPAddr + ":" + params.Port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return 
	}
	defer listener.Close() // Accept connection. 

	fmt.Printf("TCP Server is listening on IP address %s and port %s...\n", params.IPAddr, params.Port)

	// NOTE: Has to buffered channel of capacity ONE.
	unassignedJobs := make(chan Checkpoint, 1)
	abandonedJobs := make(chan Checkpoint, 1)
	// NOTE: Has to be run in a separate goroutine.
	go generateNewJobs(params.InclusiveMaxPasswdLen, params.PartitionSize, unassignedJobs)

	var wg sync.WaitGroup

	for !globalSuccessCond.Load(){
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println(fmt.Sprintf("Accpted new peer %s", conn.RemoteAddr().String()))

		sessionID := NewSessionID(conn.RemoteAddr().String())
		session := makeCoordSession(sessionID, unassignedJobs, abandonedJobs)

		wg.Add(1)
		go coordMiddleman(conn, session, &wg, params)
	}
	wg.Wait()
	return 
}


func coordMiddleman(conn net.Conn, session *sync.Map, wg *sync.WaitGroup, params CoordParams){
	defer func(){
		conn.Close() 
		wg.Done()
	}()
	
	peerAddr := conn.RemoteAddr().String()
	print := coordSessionPrinter(peerAddr)

	defer func(){
		val, _ := session.Load("ckptValidFlag")
		ckptValidFlag := val.(bool)
		if ckptValidFlag{
			val, _ := session.Load("abandonedJobsChan")
			abandonedJobsChan := val.(chan Checkpoint)
			val, _ = session.Load("ckpt")
			ckpt := val.(Checkpoint)
			abandonedJobsChan <- ckpt
			templateText := "coordMiddleman() reassigning abandoned Job: (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d.\n"
			text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
			print(text)
		}
		print("coordMiddleman(): Terminating session.")
	}()

	sendChan 	:= make(chan string)
	recvChan  	:= make(chan string)

	go handleCoordSend(conn, session, sendChan, params.TimeoutSeconds, params.Data)
	go handleCoordRecv(conn, session, recvChan)

	recvTimeout := func(expectedRecvMsg string, timeout int)(string, error){
		select {
		case recvMsg := <-recvChan:
			if recvMsg != expectedRecvMsg {
				return recvMsg, errors.New("unexpected recvMsg")
			} else {
				return recvMsg, nil 
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			return "", errors.New("timeout on " + expectedRecvMsg) 
		}
	}
	
	// [1] Recv PeerHelloPkt.
	if _, err := recvTimeout(PeerHelloPktStr, params.TimeoutSeconds); err != nil{
		print("coordMiddleman(): Timed out to receive PeerHelloPkt packet.")
		return 
	}
	print("coordMiddleman(): PeerHelloPkt received.")

	// [2] Send CoordHelloPkt
	go func(){sendChan <- CoordHelloPktStr}()
	print("coordMiddleman(): CoordHelloPkt sent.")

	// [3] Recv PeerNewTaskPkt.
	if _, err := recvTimeout(PeerNewTaskPktStr, params.TimeoutSeconds); err != nil{
		print("coordMiddleman(): Timed out to receive PeerNewTaskPkt packet.")
		return 
	}
	print("coordMiddleman(): PeerNewTask received during handshake.")

	// [4] Send CoordTaskPk
	go func(){sendChan <- CoordTaskPktStr}() 
	print("coordMiddleman(): CoordTaskPkt sent during handshake.")

	timeoutCounter := 0
	// returning from inner function? bring terminate var back!
	recvLoopMsgFunc := func(recvMsg string) bool{

		if recvMsg == PeerNewTaskPktStr{

			print("coordMiddleman(): PeerNewTaskPkt received.")

			if globalSuccessCond.Load(){
				go func(){sendChan <- CoordDiscPktStr}() 
				time.Sleep(time.Duration(1) * time.Second)
				return true 
			}

			val, _ := session.Load("ckpt")
			ckpt := val.(Checkpoint)
			if !ckpt.Exhausted(){
				templateText := "Job: (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d\n"
				text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
				print("coordMiddleman(): Peer asking for more job but still has not finished its assigned job.", text)
				return true
			}
			go func(){sendChan <- CoordTaskPktStr}() 

			print("coordMiddleman(): CoordTaskPkt sent.")

		} else if recvMsg == PeerCkPtkStr{

			if globalSuccessCond.Load(){
				go func(){sendChan <- CoordDiscPktStr}() 
				time.Sleep(time.Duration(1) * time.Second)
				return true 
			}

			val, _ := session.Load("ckpt")
			ckpt := val.(Checkpoint)
			print(ckpt.GetPercent())
			if ckpt.Exhausted(){
				session.Store("ckptValidFlag", false)
			}

		} else if recvMsg == PeerSuccessPktStr{

			print("coordMiddleman(): PeerSuccessPkt received.")

			val, _ := session.Load("successVal")
			successVal := val.(string)
			print("\n**********PASSWORD FOUND**********\n", "Password:", successVal,"\n**********************************")
			globalSuccessCond.Store(true)  
			globalSuccessFound.Store(successVal)

		} else if recvMsg == PeerProbPktStr{

			print("coordMiddleman(): PeerProbPkt received.")
			go func(){sendChan <- CoordAlivePktStr}() 
			print("coordMiddleman(): CoordAlivePkt sent.")

		} else if recvMsg == PeerDiscPktStr{

			print("coordMiddleman(): PeerDiscPkt received.")

			val, _ := session.Load("ckpt")
			ckpt := val.(Checkpoint)
			print(ckpt.GetPercent())
			if ckpt.Exhausted(){ // do not reassign if job is done 
				session.Store("ckptValidFlag", false)
			}
			return true 
			
		} else if recvMsg == PeerAlivePktStr{
			print("coordMiddleman(): PeerAlivePkt received.")
			timeoutCounter = 0

		} else {
			panic("coordMiddleman(): Unrecognized message from handleCoordRecv()")
		}
		return false 
	}

	  
	//terminate := false
	// !terminate
	for {
		select{
			case recvMsg := <-recvChan:
				terminate := recvLoopMsgFunc(recvMsg)
				if terminate{
					return 
				}

			case <-time.After(time.Duration(params.TimeoutSeconds) * time.Second):

				val, _ := session.Load("connFailFlag")
				connFailFlag := val.(bool)
				if connFailFlag{
					print("coordMiddleman(): connFailFlag is set.")
					return
				}
				
				go func(){sendChan <- CoordProbPktStr}() 
				print("coordMiddleman(): Sent CoordProbPkt.")
				timeoutCounter++ 

				if timeoutCounter == params.MaxTimeoutCounter{  
					print("coordMiddleman(): Max number of timeouts reached, Peer is dead.")
					return 
			}
		}
	}
}


func handleCoordSend(conn net.Conn, session *sync.Map, ch  <-chan string, timeoutSeconds int, data string){

	val, _ := session.Load("unassignedJobsChan")
	unassignedJobsChan := val.(chan Checkpoint)    
	val, _ = session.Load("abandonedJobsChan")
	abandonedJobsChan := val.(chan Checkpoint)
	peerAddr := conn.RemoteAddr().String()
	print := coordSessionPrinter(peerAddr)

	fetchUnassignedJob := func() (Checkpoint, error){
		select {
		case job := <-abandonedJobsChan:
			return job, nil
		default:
		}

		select {
		case job := <-unassignedJobsChan:
			return job, nil
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			var checkpointZero Checkpoint
			return checkpointZero, errors.New("timeout fetching a job") 
		}

	}

	for{
		var err error
		var genPkt GenPkt
		msg := <-ch 

		val,_ := session.Load("sessionID")
		sessionID := val.(string)

		if CoordHelloPktStr == msg{

			if genPkt, err = MakeCoordHelloPkt(sessionID, data); err != nil{
				print("handleCoordSend(): Failed to make a CoordHelloPkt Packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if CoordTaskPktStr == msg{

			var ckpt Checkpoint
			if ckpt, err = fetchUnassignedJob(); err != nil{
				print("handleCoordSend(): Timed out fetching a job.")
				session.Store("connFailFlag", true)
				return 
			}

			templateText := "Assigning  Job: (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d\n"
			text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
			print("handleCoordSend():", text)
			
			session.Store("ckptValidFlag", true)
			session.Store("ckpt", ckpt)

			if genPkt, err = MakeCoordTaskPkt(sessionID, ckpt); err != nil{
				print("handleCoordSend(): Failed to make a CoordTaskPkt Packet.")
				session.Store("connFailFlag", true)
				return 
			}
			

		} else if CoordDiscPktStr == msg{

			if genPkt, err = MakeCoordDiscPkt(sessionID); err != nil{
				print("handleCoordSend(): Failed to make a CoordDiscPkt Packet.")
				session.Store("connFailFlag", true)
				return 
			}
			
		} else if CoordAlivePktStr == msg{

			if genPkt, err = MakeCoordAlivePkt(sessionID); err != nil{
				print("handleCoordSend(): Failed to make a CoordAlivePkt Packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if CoordProbPktStr == msg{

			if genPkt, err = MakeCoordProbPkt(sessionID); err != nil{
				print("handleCoordSend(): Failed to make a CoordProbPkt Packet.")
				session.Store("connFailFlag", true)
				return
			}

		} else{
			panic("handleCoordSend(): Unrecognized message from coordMiddleman().")
		}

		if _, err := sendGenPkt(conn, genPkt); err != nil{
			print("handleCoordSend(): Failed to send a GenPkt Packet.")
			session.Store("connFailFlag", true)
			return  
		}
	}
}

func handleCoordRecv(conn net.Conn, session *sync.Map, ch chan<- string){

	peerAddr := conn.RemoteAddr().String()
	print := coordSessionPrinter(peerAddr)
	for{
		var err error
		var genPkt GenPkt

		if genPkt, err = RecvGenPkt(conn); err != nil{
			print("handleCoordRecv(): Failed to receive GenPkt.")
			session.Store("connFailFlag", true)
			return
		}

		if _, err := FetchPayload[PeerNewTaskPkt](genPkt, PeerNewTaskPktStr); err == nil{

			ch <- PeerNewTaskPktStr

		} else if peerCkPtk, err := FetchPayload[PeerCkPtk](genPkt, PeerCkPtkStr); err == nil{

			val, _ := session.Load("ckpt")
			oldCkpt := val.(Checkpoint)
			oldLatestIdx, _ := oldCkpt.GetLatestCompletedIdx()
			latestIdx, _ := peerCkPtk.Ckpt.GetLatestCompletedIdx()
			if oldLatestIdx > latestIdx{
				print("handleCoordRecv(): Received checkpoint of PeerCkPtk packet lower than the existing checkpoint.")
				session.Store("connFailFlag", true)
				return
			}
			session.Store("ckpt", peerCkPtk.Ckpt)
			ch <- PeerCkPtkStr

		} else if peerSuccessPkt, err := FetchPayload[PeerSuccessPkt](genPkt, PeerSuccessPktStr); err == nil{

			session.Store("successVal", peerSuccessPkt.SuccessVal)
			ch <- PeerSuccessPktStr

		} else if peerDiscPkt, err := FetchPayload[PeerDiscPkt](genPkt, PeerDiscPktStr); err == nil{

			val, _ := session.Load("ckpt")
			oldCkpt := val.(Checkpoint)
			oldLatestIdx, _ := oldCkpt.GetLatestCompletedIdx()
			latestIdx, _ := peerDiscPkt.Ckpt.GetLatestCompletedIdx()
			if oldLatestIdx > latestIdx{
				print("handleCoordRecv(): Received checkpoint of PeerDiscPkt packet lower than the existing checkpoint.")
				session.Store("connFailFlag", true)
				return
			}

			ckpt := peerDiscPkt.Ckpt
			templateText := "Peer disconnecting at Job: (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d\n"
			text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
			print("handleCoordRecv():", text)
			
			session.Store("ckpt", peerDiscPkt.Ckpt)
			ch <- PeerDiscPktStr

		} else if _, err := FetchPayload[PeerProbPkt](genPkt, PeerProbPktStr); err == nil{

			ch <- PeerProbPktStr

		} else if  _, err := FetchPayload[PeerHelloPkt](genPkt, PeerHelloPktStr); err == nil{

			ch <- PeerHelloPktStr

		} else if _, err := FetchPayload[PeerAlivePkt](genPkt, PeerAlivePktStr); err == nil{

			ch <- PeerAlivePktStr

		} else {
			print("handleCoordRecv(): Received an unrecognized packet type.")
			session.Store("connFailFlag", true)
			return
		}
	}
}

// NOTE: Run in a separate goroutine. 
func generateNewJobs(inclusiveMaxPasswdLen, partitionSize int, unassignedJobsChan chan Checkpoint){
	uniqueCharCount :=  hashcrack.LastChar - hashcrack.FirstChar + 1
	for currentLen := 1; currentLen <= inclusiveMaxPasswdLen; currentLen++{
		endIdxExclusive := hashcrack.IntPow(uniqueCharCount, currentLen)
		currentIdx := -1
		paritionCounter := 0 
		for i := 0; i < endIdxExclusive; i++{
			if currentIdx == -1{
				currentIdx = i
			}
			paritionCounter++ 
			if paritionCounter == partitionSize{
				unassignedJobsChan <- *NewCheckpoint(currentIdx, i, currentLen)
				paritionCounter = 0
				currentIdx = -1
			}
		}
		if(paritionCounter == 0 && currentIdx != -1) || (paritionCounter != 0 && currentIdx == -1){
			panic("...")
		}
		if 0 != paritionCounter && currentIdx != -1{
			unassignedJobsChan <- *NewCheckpoint(currentIdx, endIdxExclusive - 1, currentLen)
		}
	}

}



func coordSessionPrinter(peerAddr string) func(strs ...string) {
	return func(strs ...string) {
		acc := ""
		for _, str := range strs {
			acc = acc + str + " "
		}
		if len(acc) > 0 {
			acc = acc[:len(acc)-1] 
		}
		formattedAcc := peerAddr + "-> " + acc
		fmt.Println(formattedAcc)
	}
}


func makeCoordSession(sessionID string, unassignedJobsChan, abandonedJobsChan chan Checkpoint) *sync.Map{
	var session sync.Map
	var checkpointZero Checkpoint
	
	session.Store("unassignedJobsChan", unassignedJobsChan)
	session.Store("abandonedJobsChan", abandonedJobsChan)
	session.Store("sessionID", sessionID)
	session.Store("ckpt", checkpointZero)
	session.Store("ckptValidFlag", false)
	session.Store("successVal", "")
	session.Store("connFailFlag", false)

	return &session
} 

