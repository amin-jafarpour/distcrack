package distnet 



var MajorPointStrs = map[int]string{
	0: "25%",
	1: "50%",
	2: "75%",
} 

type Checkpoint struct{
	InclusiveStartIdx 	int 
	InclusiveEndIdx   	int
	LatestCompletedIdx  int 
	JobTypeID 			int 
	MajorPoints	        []int 
	MajorPointIdx 		int 
} 

func NewCheckpoint(inclusiveStartIdx, inclusiveEndIdx, jobTypeID int) *Checkpoint{
	if inclusiveStartIdx > inclusiveEndIdx{
		panic("NewCheckpoint(): inclusiveEndIdx can't be less than inclusiveStartIdx.")
	}

	// BUG: Major point calculations may be wrong. 
	exclusiveBase := inclusiveStartIdx - 1
	delta := inclusiveEndIdx - inclusiveStartIdx + 1
	one   := (delta / 4) * 1  + exclusiveBase
	two   := (delta / 4) * 2  + exclusiveBase
	three := (delta / 4) * 3  + exclusiveBase

	
	return &Checkpoint{
		InclusiveStartIdx: inclusiveStartIdx,
		InclusiveEndIdx: inclusiveEndIdx,
		LatestCompletedIdx: inclusiveStartIdx - 1,
		JobTypeID: jobTypeID,
		MajorPoints: []int{one, two, three},
		MajorPointIdx: 0,
	}
}

func (ckpt *Checkpoint) IncCheckpoint() bool{
	if ckpt.LatestCompletedIdx + 1 > ckpt.InclusiveEndIdx{
		panic("NewCheckpoint(): latestCompletedIdx can't be greater than inclusiveCurrentIdx.")
	}
	
	ckpt.LatestCompletedIdx += 1

	if ckpt.MajorPointIdx < len(ckpt.MajorPoints) && ckpt.MajorPoints[ckpt.MajorPointIdx] == ckpt.LatestCompletedIdx{
		ckpt.MajorPointIdx += 1
		return true
	}

	if ckpt.LatestCompletedIdx >= ckpt.InclusiveEndIdx{
		return true 
	}
	return false
}


func (ckpt *Checkpoint) GetPercent() string{

	if ckpt.LatestCompletedIdx < ckpt.MajorPoints[0]{
		return "0%"
	} else if ckpt.LatestCompletedIdx >= ckpt.MajorPoints[0] && ckpt.LatestCompletedIdx < ckpt.MajorPoints[1]{
		return "25%"
	}else if ckpt.LatestCompletedIdx >= ckpt.MajorPoints[1] && ckpt.LatestCompletedIdx < ckpt.MajorPoints[2]{
		return "50%"
	} else if ckpt.LatestCompletedIdx >= ckpt.MajorPoints[2] && ckpt.LatestCompletedIdx < ckpt.InclusiveEndIdx{
		return "75%"
	} else if ckpt.LatestCompletedIdx >= ckpt.InclusiveEndIdx{
		return "100%"
	}
	panic("...")

	return "N/A%"
}

func (ckpt *Checkpoint) GetInclusiveStartIdx() int{
	return ckpt.InclusiveStartIdx
}

func (ckpt *Checkpoint) GetInclusiveEndIdx() int{
	return ckpt.InclusiveEndIdx
}

func (ckpt *Checkpoint) GetLatestCompletedIdx() (int, bool){
	ok := true														
	if ckpt.LatestCompletedIdx == ckpt.InclusiveStartIdx - 1 || ckpt.LatestCompletedIdx > ckpt.InclusiveEndIdx{
		ok = false
	}
	return ckpt.LatestCompletedIdx, ok 
}

func (ckpt *Checkpoint) Exhausted() bool{
	if ckpt.LatestCompletedIdx >= ckpt.InclusiveEndIdx{
		return true
	}
	return false 
}

func (ckpt *Checkpoint) GetJobTypeID() int{
	return ckpt.JobTypeID
}

