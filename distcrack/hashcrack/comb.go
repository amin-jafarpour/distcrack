package hashcrack 

import(
	"fmt"
	"errors"
	"bytes"
)


const FirstChar = 32
const LastChar = 126

// Represent a "combination" structure. It takes characters to be placed in a single 
// slot and number of slots. For example, for {A, B, C} and 3 slots: 
// AAA, BAA, CAA, ABA, BBA, CBA, ACA, BCA, CCA, AAB, BAB, CAB, ABB, BBB, CBB, ACB, BCB, CCB
// AAC, BAC, CAC, ABC, BBC, CBC, ACC, BCC, CCC. 
// 3^3 = 27 combinations. 
type Comb struct{
	units []byte // Slice of characters to be included. 
	unitLen int  // Length of `units`.
	seqLen int  // Length of sequence of characters/units to be put together. 
	combsLen int  // Length/number of combinations of `seqLen` of `units`, i.e, len(units)^seqLen
}

// Constructs and returns a new `Comb` object. 
func NewComb(units []byte, seqLen int) *Comb{
	unitLen := len(units) // Store number of characters/units.
	// Compute total number of combinations with repetition allowed, i.e., len(units)^seqLen. 
	combsLen := IntPow(unitLen, seqLen) 
	return &Comb{
		units: units,
		unitLen: unitLen,
		seqLen: seqLen,
		combsLen: combsLen,
	}
}

// Give the combination index, it returns the string represetation of the character's
// combinations, represeted by the index. For example, for {A, B, C} and 3 slots case,
// index 0 returns "AAA" and index 1 returns "BAA" and so on. 
func (comb *Comb) Index(idx int)(string, error){
	// If index is invalid, return an empty string and the error.
	if idx >= comb.combsLen{
		errMsg := fmt.Sprintf("Index Out of Range %d/%d", idx, comb.combsLen - 1)
		return "", errors.New(errMsg)
	}
	// Allocate a byte slice of size `comb.seqLen` and fill it with 
	// the first character of the character set. 
	byteSeq := bytes.Repeat([]byte{comb.units[0]}, comb.seqLen)

	// Until `quotient` is zero, derive one character based on `idx` 
	// and place it onto the `byteSeq` slice. Reduce the size of `quotient`,
	// and repeat the same steps. 
	quotient := idx
	for i := 0; quotient != 0; i++{
		rem := quotient % comb.unitLen
		byteSeq[i] = comb.units[rem]
		quotient /= comb.unitLen
	}
	return string(byteSeq), nil 
}

// Returns the total number of combinations possible. 
func (comb *Comb) CombsLen() int{
	return comb.combsLen
}

// Just a simple integer power function, pow(base, exp) = base ^ exp,
// because Golang doesn't have a built-in one yet!
func IntPow(base, exp int) int {
    result := 1
    for exp > 0 {
        if exp & 1 == 1 { // if exp is odd
            result *= base
        }
        base *= base // square the base
        exp >>= 1    // exp = exp / 2
    }
    return result
}

func GenerateASCIIComb(seqLen int) *Comb{
	units := []byte{}
	for i := FirstChar; i <= LastChar; i++{ 
		units = append(units, byte(i)) 
	}
	return NewComb(units , seqLen) 
}








