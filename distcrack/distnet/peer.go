package distnet

import (
	"fmt"
	"net"
	"sync"
	"time"
	"errors"
	"distcrack/hashcrack"
)

type PeerParams struct{
	TimeoutSeconds int 
	MaxTimeoutCounter int 
	IPAddr string 
	Port string  
	ThreadNumber int 
}

type WorkerInput struct{
	Idx int 
	Item string 
}

type WorkerOutput struct{
	Idx int 
	Success bool 
}

const(
	JobCkptStr 		string = "JobCkpt"
	JobCkptLastStr 	string = "JobCkptLast"
	JobSuccessStr   string = "JobSuccess"
)  

func Connect(params PeerParams){
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", params.IPAddr, params.Port))
	if err != nil {
		fmt.Println("Connect():", err)
		return 
	}
	fmt.Printf("Connected to coordinator at %s:%s\n", params.IPAddr, params.Port)

	session := makePeerSession()
	peerMiddleman(conn, session, params)
}



func peerMiddleman(conn net.Conn, session *sync.Map, params PeerParams){
	defer conn.Close()
	defer fmt.Println("peerMiddleman(): Terminating session.")

	sendChan 	:= make(chan string)
	recvChan  	:= make(chan string)
	jobChan   	:= make(chan string)

	go handlePeerSend(conn, session, sendChan)
	go handlePeerRecv(conn, session, recvChan)

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
	
	// [1] Send PeerHelloPkt
	go func(){sendChan <- PeerHelloPktStr}()

	fmt.Println("peerMiddleman(): PeerHelloPkt sent.")
	
	// [2] recv CoordHelloPkt
	if _, err := recvTimeout(CoordHelloPktStr, params.TimeoutSeconds); err != nil{
		fmt.Println("peerMiddleman(): Timed out on receiving CoordHelloPkt")
		return
	} 

	fmt.Println("peerMiddleman(): CoodHelloPkt received.")

	// [3] Send PeerNewTaskPkt
	go func(){sendChan <- PeerNewTaskPktStr}()

	fmt.Println("peerMiddleman(): PeerNewTask sent during handshake.")

	// [4] recv CoordTaskPkt
	if _, err := recvTimeout(CoordTaskPktStr, params.TimeoutSeconds); err != nil{
		fmt.Println("peerMiddleman(): Timed out on receiving CoordTaskPkt during handshake.")
		return 
	}

	fmt.Println("peerMiddleman(): CoodTaskPkt received.")

	go handlePeerJob(session, jobChan, params.ThreadNumber)
	jobHandleAlive := true 
	//terminate := false 
	timeoutCounter := 0 

	recvLoopMsgFunc := func(recvMsg string) bool{
		if recvMsg == CoordTaskPktStr && jobHandleAlive{
			errMsg := "Error: Received a CoordTaskPkt from coord even though current job is still in progress"
			fmt.Println(errMsg)
			// terminate = true 
			return true 
			
		} else if recvMsg == CoordTaskPktStr && !jobHandleAlive{
			fmt.Println("peerMiddleman(): CoordTaskPkt recevied.")
			go handlePeerJob(session, jobChan, params.ThreadNumber)
			jobHandleAlive = true  

		} else if recvMsg == CoordDiscPktStr{
			fmt.Println("peerMiddleman(): CoordDiscPkt received.")
			//terminate = true 
			return true 

		} else if recvMsg == CoordProbPktStr{
			fmt.Println("peerMiddleman(): CoordProbPkt received.")
			go func(){sendChan <- PeerAlivePktStr}()
			fmt.Println("peerMiddleman(): PeerAlivePkt sent.")

		} else if recvMsg == CoordAlivePktStr{
			fmt.Println("peerMiddleman() CoordAlivePkt received.")
			timeoutCounter = 0
		} else{
			panic("peerMiddleman(): Got an unrecognized messge from handlePeerRecv(): " + recvMsg)
		}
		return false 
	}

	successCond := false 

	jobLoopMsgFunc := func(jobMsg string){
			if jobMsg == JobCkptStr{
				go func(){sendChan <- PeerCkPtkStr}()

			} else if jobMsg == JobCkptLastStr {
				go func(){sendChan <- PeerCkPtkStr}()
				jobHandleAlive = false 
				time.Sleep(time.Second)
				go func(){sendChan <- PeerNewTaskPktStr}()
				fmt.Println("peerMiddleman(): Current job done. PeerNewTask sent.")

			} else if jobMsg == JobSuccessStr{
				go func(){sendChan <- PeerSuccessPktStr}()
				successCond = true
				fmt.Println("coordMiddleman(): PeerSuccessPkt sent.") 
				
			} else {
				panic("peerMiddleman(): Got an unrecognized messge from handlePeerJob: " + jobMsg)
			}
	}

	// !terminate
	for {
		select{
			case recvMsg := <-recvChan:
				terminate := recvLoopMsgFunc(recvMsg)
				if terminate{
					return 
				}

			case jobMsg := <- jobChan:
				jobLoopMsgFunc(jobMsg)
				
			case <-time.After(time.Duration(params.TimeoutSeconds) * time.Second):
		
				val, _ := session.Load("connFailFlag")
				connFailFlag := val.(bool)
				if connFailFlag{
					fmt.Println("peerMiddleman(): connFailFlag is set.")
					// terminate = true 
					return 
				}

				if successCond{
					go func(){sendChan <- PeerSuccessPktStr}()
					fmt.Println("peerMiddleman(): PeerSuccessPkt sent.")
				}

				go func(){sendChan <- PeerProbPktStr}() 
				fmt.Println("peerMiddleman(): Sent PeerProbPkt.")
				timeoutCounter++ 
				if timeoutCounter == params.MaxTimeoutCounter{  
					fmt.Println("peerMiddleman(): Max number of timeouts reached, coord is dead.")
					// terminate = true 
					return 
				}
		}
	}
}

func handlePeerSend(conn net.Conn, session *sync.Map, ch  <-chan string){
	for {
		var err error
		var genPkt GenPkt
		msg := <-ch 

		// NOTE: sessionID assignment must be exactly here. Don't move it. 
		val,_ := session.Load("sessionID")
		sessionID := val.(string)

		if  PeerHelloPktStr == msg{
			if genPkt, err = MakePeerHelloPkt("ipv4", "ipv6", "mac"); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerHelloPkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerCkPtkStr == msg{
			val, _ := session.Load("ckpt")
			ckpt := val.(Checkpoint)

			if genPkt, err = MakePeerCkPtk(sessionID, ckpt); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerCkPkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerNewTaskPktStr == msg{
			if genPkt, err =  MakePeerNewTaskPkt(sessionID); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerNewTaskPkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerDiscPktStr == msg{
			val, _ := session.Load("ckpt")
			ckpt := val.(Checkpoint)

			if genPkt, err = MakePeerDiscPkt(sessionID, ckpt); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerHelloPkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerSuccessPktStr == msg{
			val, _ := session.Load("successVal")
			successVal := val.(string)

			if genPkt, err = MakePeerSuccessPkt(sessionID, successVal); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerSuccessPkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerAlivePktStr == msg{
			if genPkt, err = MakePeerAlivePkt(sessionID); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerAlivePkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else if PeerProbPktStr ==  msg{
			if genPkt, err = MakePeerProbPkt(sessionID); err != nil{
				fmt.Println("handlePeerSend(): failed to make PeerProbePkt packet.")
				session.Store("connFailFlag", true)
				return 
			}

		} else {
			panic("handlePeerSend() got an invalid message from peerMiddleman()")
		}

		if _, err := sendGenPkt(conn, genPkt); err != nil{
			fmt.Println("handlePeerSend(): failed to send GenPkt Packet.")
			session.Store("connFailFlag", true)
			return 
		}
	}
}

func  handlePeerRecv(conn net.Conn, session *sync.Map, ch chan<- string){
	for {
		var err error
		var genPkt GenPkt

		if genPkt, err = RecvGenPkt(conn); err != nil{
			fmt.Println("handlePeerRecv() failed to receive GenPkt.")
			session.Store("connFailFlag", true)
			return 
		}

		if coordTaskPkt, err := FetchPayload[CoordTaskPkt](genPkt, CoordTaskPktStr); err == nil{
			session.Store("ckpt", coordTaskPkt.Ckpt)
			ch <- CoordTaskPktStr 

		} else if _, err := FetchPayload[CoordProbPkt](genPkt, CoordProbPktStr); err == nil{
			ch <- CoordProbPktStr


		} else if _, err := FetchPayload[CoordAlivePkt](genPkt, CoordAlivePktStr); err == nil{
			ch <- CoordAlivePktStr

		} else if coordHelloPkt, err := FetchPayload[CoordHelloPkt](genPkt, CoordHelloPktStr); err == nil{
			session.Store("sessionID", coordHelloPkt.SessionID)
			session.Store("data", coordHelloPkt.Data)
			ch <- CoordHelloPktStr

		} else if _, err := FetchPayload[CoordDiscPkt](genPkt, CoordDiscPktStr); err == nil{
			ch <- CoordDiscPktStr

		} else {
			panic("handlePeerRecv() got an invalid message from peerMiddleman().")  
		}
	}
}

func handlePeerJob(session *sync.Map, ch chan<- string, threadNumber int){

	val, _ := session.Load("ckpt")
	ckpt := val.(Checkpoint)
	val, _ = session.Load("data")
	data := val.(string)
	ckptStartIdx := -1

	templateText := "Beginning to work on new job: (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d.\n"
	text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
	fmt.Println("handlePeerJob()", text)

	for !ckpt.Exhausted(){

		reached := ckpt.IncCheckpoint()
		idxLatest, _:= ckpt.GetLatestCompletedIdx()
		if ckptStartIdx == -1{
			ckptStartIdx = idxLatest
		}

		if reached{
			if successVal, ok := isCkptSuccess(ckptStartIdx, idxLatest, ckpt.GetJobTypeID(), threadNumber, data); ok{
				session.Store("successFlag", true)
				session.Store("successVal", successVal)
				fmt.Println("**********PASSWORD FOUND**********\n", "Password:", successVal,"\n**********************************")
				ch <- JobSuccessStr
			}
			session.Store("ckpt", ckpt)
			ch <- JobCkptStr
			ckptStartIdx = -1

			templateText := "Job at (Start Index: %d, Last Completed Index: %d, End: %d, PasswdLen: %d.\n"
			text := fmt.Sprintf(templateText, ckpt.GetInclusiveStartIdx(), ckpt.LatestCompletedIdx, ckpt.GetInclusiveEndIdx(), ckpt.GetJobTypeID())
			fmt.Println("handlePeerJob(): Work progress", ckpt.GetPercent(), text)
		}
	}

	ch <- JobCkptLastStr
}


func isCkptSuccess(ckptStartIdx, ckptEndIdx, jobID, threadNumber int, data string)(string, bool){

	inputChan := make(chan WorkerInput)
	outputChan := make(chan WorkerOutput)
	comb := hashcrack.GenerateASCIIComb(jobID) 

	go func(){
		for i := ckptStartIdx; i <= ckptEndIdx; i++{
			if guess, err := comb.Index(i); err != nil{
				panic("...")
			} else{
				inputChan <- WorkerInput{Idx: i, Item: guess}
			}
		}
		close(inputChan)
	}()

	var wg sync.WaitGroup
	wg.Add(threadNumber)
	for i := 0; i < threadNumber; i++{
		go workerFunc(inputChan, outputChan, data, &wg)
	}

	go func(){
		wg.Wait() 
		close(outputChan)
	}()

	var err error 
	successFlag := false
	successVal := ""

	for output := range outputChan{
		if output.Success{
			successFlag = true 
			if successVal, err = comb.Index(output.Idx); err != nil{
				panic("...")
			} 
		}
	}

	return successVal, successFlag
}

func workerFunc(inputChan chan WorkerInput, outputChan chan WorkerOutput, data string, wg *sync.WaitGroup){
	defer wg.Done()
	for input := range inputChan {
		if ok := isPasswd(input.Item, data); ok{
			outputChan <- WorkerOutput{Success: true, Idx: input.Idx}
		} else{
			outputChan <- WorkerOutput{Success: false, Idx: input.Idx}
		}
	} 
}

func isPasswd(guess, hash string) bool{
	realHashTokens := hashcrack.SplitHash(hash) 
	if len(realHashTokens) != 2{
		fmt.Println("Error: Invalid hash format.")
		panic("...")
	}
	if guessHash, ok := hashcrack.GenHash(guess, realHashTokens[0]); ok {
		return guessHash == hash
		
	} else{
		panic("...")
		return false 
	}
}


func makePeerSession() *sync.Map{ 
	var session sync.Map
	var checkpointZero Checkpoint

	session.Store("sessionID", "")
	session.Store("ckpt", checkpointZero)
	session.Store("successFlag", false)
	session.Store("successVal", "")
	session.Store("data", "")
	session.Store("connFailFlag", false)

	return &session
}
