/* 
	This package implements hex encoding with 0x prefix

	Hex encoding is performed by converting the 8 bit data to 2 
	hex characters. The hex characters are then stored as the two 
	byte string representation of the characters. 

	Often some kind of separator is used to make the encoded data 
	easire for humans to read 

	For byte slices, the hex data must be of even length. Any empty 
	byte slice encode as "0x" 

	Integers are encoded using the least amount of digits (no leading zero digits). 
	Their encoding may be of uneven length. The number zero encodes as "0x0"


*/

package hexutil

import (
	"encoding/hex" 
	"fmt" 
	"math/big" 
	"strconv"
)

const uintBits = 32 << (uint64(^uint(0)) >> 63)

// Errors 
var (
	ErrEmptyString = &decError{"empty hex string"}
	ErrSyntax = &decError{"invalid hex string"}
	ErrMissingPrefix = &decError{"hex string withouth a 0x prefix"} 
	ErrOddLength = &decError{"hex string of odd length"} 
	ErrEmptyNumber = &decError{"hex string \"0x\""} 
	ErrLeadingZero = &decError{"hex number without leading zero digits"} 
	ErrUint64Range = &decError{"hex number > 64 bits"} 
	ErrUintRange= &decError{fmt.Sprintf("hex number > %d bits", uintBits)}
	ErrBig256Range = &decError{"hex number > 256 bits"}
)


type decError struct{ msg string}  

func (err decError) Error() string { return err.msg }

// Decode decodes a hex string with 0x prefix 
func Decode(input string)([]byte, error){
	if len(input) == 0{
		return nil, ErrEmptyString
	}
	if !has0xPrefix(input){
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil{
		err = mapError(err)
	}

	return b, err 
}


// MustDecode decodes a hex string with 0x prefix. It panics for invalid input 
func MustDecode(input string) []byte{
	dec, err := Decode(input) 
	if err != nil{
		panic(err)
	}
	return dec
}

// Encode encodes b as hex string with 0x prefix 
func Encode(b []byte) string{
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x") 
	hex.Encode(enc[2:], b)
	return string(enc)
}

// DecodeUint64 decodes a hex  string with 0x prefix as a quantity 
func DecodeUint64(input string) (uint64, error){
	raw, err := checkNumber(input)
	if err != nil{
		return 0, nil
	}
	dec, err:= strconv.ParseUint(raw, 16, 64)
	if err != nil{
		err = mapError(err)
	}
	return dec, err 
}

// MustDecodeUint64 decodes a hex string with 0x prefix as a quantity 
// It panics for invalid input 
func MustDecodeUint64(input string) uint64{
	dec, err := DecodeUint64(input) 
	if err != nil{
		panic(err) 
	}
	return dec 
}

// EncodeUint64 encodes i as a hex string with 0x prefix 
func EncodeUint64(i uint64) string{
	enc := make([]byte, 2, 10)
	copy(enc, "0x") 
	return string(strconv.AppendUint(enc, i, 16))
}


var bigWordNibbles int

func init(){
	// Compute the number of nibbles required for big.word.  
	// The usual way would be to use constant arithmetic  bu 
	// go  can't handle that 
	b, _ := new(big.Int).SetString("FFFFFFFFF", 16) 
	switch len(b.Bits()){
	case 1:
		bigWordNibbles = 16 
	case 2: 
		bigWordNibbles = 8 
	default: 
		panic("weird big.Word size")
	}
}

// DecodeBig decodes a hex string with 0x prefix as a quantity 
// Numbers larger than 256 are not accepted 
func DecodeBig(input string)(*big.Int, error){
	raw, err := checkNumber(input) 
	if err != nil{
		return nil, err 
	}
	if len(raw) > 64{
		return nil, ErrBig256Range
	}

	words := make([]big.Word, len(raw)/bigWordNibbles+1) 
	end := len(raw)
	for i := range words{
		start := end - bigWordNibbles 
		if start < 0{
			start = 0 
		}

		for ri := start; ri < end; ri++{
			nib := decodeNibble(raw[ri]) 
			if nib == badNibble{
				return nil, ErrSyntax
			}
			words[i] *= 16 
			words[i] += big.Word(nib)
		}
		end = start
	}
	dec := new(big.Int).SetBits(words) 
	return dec, nil 
}

// MustDecodeBig decodes a hex string with 0x prefix as a quantity 
// It panics for invalid input 
func MustDecodeBig(input string) *big.Int{
	dec, err := DecodeBig(input) 
	if err != nil{
		panic(err)
	}
	return dec
}

// EncodeBig encodes bigint as a hex string with 0x prefix 
func EncodeBig(bigint *big.Int) string{
	if sign := bigint.Sign(); sign == 0{
		return "0x0"
	} else if sign > 0{
		return "0x" + bigint.Text(16)
	}else {
		return "-0x" + bigint.Text(16)[1:]
	}
}


func has0xPrefix(input string) bool{
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func checkNumber(input string) (raw string, err error){
	if len(input) == 0{
		return "", ErrEmptyString
	}

	if !has0xPrefix(input){
		return "", ErrMissingPrefix
	}

	input = input[2:] 
	if len(input) == 0{
		return "", ErrEmptyNumber
	}

	if len(input) > 1 && input[0] == '0'{
		return "", ErrLeadingZero
	}

	return input, nil 
}

const badNibble = ^uint64(0)

func decodeNibble(in byte) uint64{
	switch{
	case in >= '0'  && in <= '9': 
		return uint64(in - '0') 
	case in >= 'A' && in <= 'F': 
		return uint64(in - 'A' + 10)
	case in >= 'a' && in <= 'f': 
		return uint64(in - 'a' + 10)
	default:
		return badNibble
	}
}

func mapError(err error) error{
	if err, ok := err.(*strconv.NumError); ok{
		switch err.Err{
		case strconv.ErrRange:
			return ErrUint64Range
		case strconv.ErrSyntax: 
			return ErrSyntax
		}
	}

	if _, ok := err.(hex.InvalidByteError); ok{
		return ErrSyntax
	}
	if err == hex.ErrLength{
		return ErrOddLength	
	}

	return err
}