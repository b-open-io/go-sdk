package interpreter

import (
	"math"
	"math/big"

	"github.com/bitcoin-sv/go-sdk/script/interpreter/errs"
)

// ScriptNumber represents a numeric value used in the scripting engine with
// special handling to deal with the subtle semantics required by consensus.
//
// All numbers are stored on the data and alternate stacks encoded as little
// endian with a sign bit.  All numeric opcodes such as OP_ADD, OP_SUB,
// and OP_MUL, are only allowed to operate on 4-byte integers in the range
// [-2^31 + 1, 2^31 - 1], however the results of numeric operations may overflow
// and remain valid so long as they are not used as inputs to other numeric
// operations or otherwise interpreted as an integer.
//
// For example, it is possible for OP_ADD to have 2^31 - 1 for its two operands
// resulting 2^32 - 2, which overflows, but is still pushed to the stack as the
// result of the addition.  That value can then be used as input to OP_VERIFY
// which will succeed because the data is being interpreted as a boolean.
// However, if that same value were to be used as input to another numeric
// opcode, such as OP_SUB, it must fail.
//
// This type handles the aforementioned requirements by storing all numeric
// operation results as an int64 to handle overflow and provides the Bytes
// method to get the serialized representation (including values that overflow).
//
// Then, whenever data is interpreted as an integer, it is converted to this
// type by using the NewNumber function which will return an error if the
// number is out of range or not minimally encoded depending on parameters.
// Since all numeric opcodes involve pulling data from the stack and
// interpreting it as an integer, it provides the required behavior.
type ScriptNumber struct {
	Val          *big.Int
	AfterGenesis bool
}

var Zero = big.NewInt(0)
var One = big.NewInt(1)

// MakeScriptNumber interprets the passed serialized bytes as an encoded integer
// and returns the result as a Number.
//
// Since the consensus rules dictate that serialized bytes interpreted as integers
// are only allowed to be in the range determined by a maximum number of bytes,
// on a per opcode basis, an error will be returned when the provided bytes
// would result in a number outside that range.  In particular, the range for
// the vast majority of opcodes dealing with numeric values are limited to 4
// bytes and therefore will pass that value to this function resulting in an
// allowed range of [-2^31 + 1, 2^31 - 1].
//
// The requireMinimal flag causes an error to be returned if additional checks
// on the encoding determine it is not represented with the smallest possible
// number of bytes or is the negative 0 encoding, [0x80].  For example, consider
// the number 127.  It could be encoded as [0x7f], [0x7f 0x00],
// [0x7f 0x00 0x00 ...], etc.  All forms except [0x7f] will return an error with
// requireMinimal enabled.
//
// The scriptNumLen is the maximum number of bytes the encoded value can be
// before an errs.ErrStackNumberTooBig is returned.  This effectively limits the
// range of allowed values.
// WARNING:  Great care should be taken if passing a value larger than
// defaultScriptNumLen, which could lead to addition and multiplication
// overflows.
//
// See the Bytes function documentation for example encodings.
func MakeScriptNumber(bb []byte, scriptNumLen int, requireMinimal, afterGenesis bool) (*ScriptNumber, error) {
	// Interpreting data requires that it is not larger than the passed scriptNumLen value.
	if len(bb) > scriptNumLen {
		return &ScriptNumber{Val: big.NewInt(0), AfterGenesis: false}, errs.NewError(
			errs.ErrNumberTooBig,
			"numeric value encoded as %x is %d bytes which exceeds the max allowed of %d",
			bb, len(bb), scriptNumLen,
		)
	}

	// Enforce minimal encoded if requested.
	if requireMinimal {
		if err := CheckMinimalDataEncoding(bb); err != nil {
			return &ScriptNumber{
				Val:          big.NewInt(0),
				AfterGenesis: false,
			}, err
		}
	}

	// Zero is encoded as an empty byte slice.
	if len(bb) == 0 {
		return &ScriptNumber{
			AfterGenesis: afterGenesis,
			Val:          big.NewInt(0),
		}, nil
	}

	// Decode from little endian.
	//
	// The following is the equivalent of:
	//    var v int64
	//    for i, b := range bb {
	//        v |= int64(b) << uint8(8*i)
	//    }
	v := new(big.Int)
	for i, b := range bb {
		v.Or(v, new(big.Int).Lsh(new(big.Int).SetBytes([]byte{b}), uint(8*i)))
	}

	// When the most significant byte of the input bytes has the sign bit
	// set, the result is negative.  So, remove the sign bit from the result
	// and make it negative.
	//
	// The following is the equivalent of:
	//    if bb[len(bb)-1]&0x80 != 0 {
	//        v &= ^(int64(0x80) << uint8(8*(len(bb)-1)))
	//        return -v, nil
	//    }
	if bb[len(bb)-1]&0x80 != 0 {
		// The maximum length of bb has already been determined to be 4
		// above, so uint8 is enough to cover the max possible shift
		// value of 24.
		shift := big.NewInt(int64(0x80))
		shift.Not(shift.Lsh(shift, uint(8*(len(bb)-1))))
		v.And(v, shift).Neg(v)
	}
	return &ScriptNumber{
		Val:          v,
		AfterGenesis: afterGenesis,
	}, nil
}

// Add adds the receiver and the number, sets the result over the receiver and returns.
func (n *ScriptNumber) Add(o *ScriptNumber) *ScriptNumber {
	*n.Val = *new(big.Int).Add(n.Val, o.Val)
	return n
}

// Sub subtracts the number from the receiver, sets the result over the receiver and returns.
func (n *ScriptNumber) Sub(o *ScriptNumber) *ScriptNumber {
	*n.Val = *new(big.Int).Sub(n.Val, o.Val)
	return n
}

// Mul multiplies the receiver by the number, sets the result over the receiver and returns.
func (n *ScriptNumber) Mul(o *ScriptNumber) *ScriptNumber {
	*n.Val = *new(big.Int).Mul(n.Val, o.Val)
	return n
}

// Div divides the receiver by the number, sets the result over the receiver and returns.
func (n *ScriptNumber) Div(o *ScriptNumber) *ScriptNumber {
	*n.Val = *new(big.Int).Quo(n.Val, o.Val)
	return n
}

// Mod divides the receiver by the number, sets the remainder over the receiver and returns.
func (n *ScriptNumber) Mod(o *ScriptNumber) *ScriptNumber {
	*n.Val = *new(big.Int).Rem(n.Val, o.Val)
	return n
}

// LessThanInt returns true if the receiver is smaller than the integer passed.
func (n *ScriptNumber) LessThanInt(i int64) bool {
	return n.LessThan(&ScriptNumber{Val: big.NewInt(i)})
}

// LessThan returns true if the receiver is smaller than the number passed.
func (n *ScriptNumber) LessThan(o *ScriptNumber) bool {
	return n.Val.Cmp(o.Val) == -1
}

// LessThanOrEqual returns ture if the receiver is smaller or equal to the number passed.
func (n *ScriptNumber) LessThanOrEqual(o *ScriptNumber) bool {
	return n.Val.Cmp(o.Val) < 1
}

// GreaterThanInt returns true if the receiver is larger than the integer passed.
func (n *ScriptNumber) GreaterThanInt(i int64) bool {
	return n.GreaterThan(&ScriptNumber{Val: big.NewInt(i)})
}

// GreaterThan returns true if the receiver is larger than the number passed.
func (n *ScriptNumber) GreaterThan(o *ScriptNumber) bool {
	return n.Val.Cmp(o.Val) == 1
}

// GreaterThanOrEqual returns true if the receiver is larger or equal to the number passed.
func (n *ScriptNumber) GreaterThanOrEqual(o *ScriptNumber) bool {
	return n.Val.Cmp(o.Val) > -1
}

// EqualInt returns true if the receiver is equal to the integer passed.
func (n *ScriptNumber) EqualInt(i int64) bool {
	return n.Equal(&ScriptNumber{Val: big.NewInt(i)})
}

// Equal returns true if the receiver is equal to the number passed.
func (n *ScriptNumber) Equal(o *ScriptNumber) bool {
	return n.Val.Cmp(o.Val) == 0
}

// IsZero return strue if hte receiver equals zero.
func (n *ScriptNumber) IsZero() bool {
	return n.Val.Cmp(Zero) == 0
}

// Incr increment the receiver by one.
func (n *ScriptNumber) Incr() *ScriptNumber {
	*n.Val = *new(big.Int).Add(n.Val, One)
	return n
}

// Decr decrement the receiver by one.
func (n *ScriptNumber) Decr() *ScriptNumber {
	*n.Val = *new(big.Int).Sub(n.Val, One)
	return n
}

// Neg sets the receiver to the negative of the receiver.
func (n *ScriptNumber) Neg() *ScriptNumber {
	*n.Val = *new(big.Int).Neg(n.Val)
	return n
}

// Abs sets the receiver to the absolute value of hte receiver.
func (n *ScriptNumber) Abs() *ScriptNumber {
	*n.Val = *new(big.Int).Abs(n.Val)
	return n
}

// Int returns the receivers value as an int.
func (n *ScriptNumber) Int() int {
	return int(n.Val.Int64())
}

// Int32 returns the Number clamped to a valid int32.  That is to say
// when the script number is higher than the max allowed int32, the max int32
// value is returned and vice versa for the minimum value.  Note that this
// behavior is different from a simple int32 cast because that truncates
// and the consensus rules dictate numbers which are directly cast to integers
// provide this behavior.
//
// In practice, for most opcodes, the number should never be out of range since
// it will have been created with makeScriptNumber using the defaultScriptLen
// value, which rejects them.  In case something in the future ends up calling
// this function against the result of some arithmetic, which IS allowed to be
// out of range before being reinterpreted as an integer, this will provide the
// correct behavior.
func (n *ScriptNumber) Int32() int32 {
	v := n.Val.Int64()
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// Int64 returns the Number clamped to a valid int64.  That is to say
// when the script number is higher than the max allowed int64, the max int64
// value is returned and vice versa for the minimum value.  Note that this
// behavior is different from a simple int64 cast because that truncates
// and the consensus rules dictate numbers which are directly cast to integers
// provide this behavior.
//
// In practice, for most opcodes, the number should never be out of range since
// it will have been created with makeScriptNumber using the defaultScriptLen
// value, which rejects them.  In case something in the future ends up calling
// this function against the result of some arithmetic, which IS allowed to be
// out of range before being reinterpreted as an integer, this will provide the
// correct behavior.
func (n *ScriptNumber) Int64() int64 {
	if n.GreaterThanInt(math.MaxInt64) {
		return math.MaxInt64
	}
	if n.LessThanInt(math.MinInt64) {
		return math.MinInt64
	}
	return n.Val.Int64()
}

// Set the value of the receiver.
func (n *ScriptNumber) Set(i int64) *ScriptNumber {
	*n.Val = *new(big.Int).SetInt64(i)
	return n
}

// Bytes returns the number serialized as a little endian with a sign bit.
//
// Example encodings:
//
//	   127 -> [0x7f]
//	  -127 -> [0xff]
//	   128 -> [0x80 0x00]
//	  -128 -> [0x80 0x80]
//	   129 -> [0x81 0x00]
//	  -129 -> [0x81 0x80]
//	   256 -> [0x00 0x01]
//	  -256 -> [0x00 0x81]
//	 32767 -> [0xff 0x7f]
//	-32767 -> [0xff 0xff]
//	 32768 -> [0x00 0x80 0x00]
//	-32768 -> [0x00 0x80 0x80]
func (n *ScriptNumber) Bytes() []byte {
	// Zero encodes as an empty byte slice.
	if n.IsZero() {
		return []byte{}
	}

	// Take the absolute value and keep track of whether it was originally
	// negative.
	isNegative := n.Val.Cmp(Zero) == -1
	if isNegative {
		n.Neg()
	}

	var bb []byte
	if !n.AfterGenesis {
		v := n.Val.Int64()
		if v > math.MaxInt32 {
			bb = big.NewInt(int64(math.MaxInt32)).Bytes()
		} else if v < math.MinInt32 {
			bb = big.NewInt(int64(math.MinInt32)).Bytes()
		}
	}
	if bb == nil {
		bb = n.Val.Bytes()
	}

	// Encode to little endian.  The maximum number of encoded bytes is len(bb)+1
	// (8 bytes for max int64 plus a potential byte for sign extension).
	//
	// The following is the equivalent of:
	//    result := make([]byte, 0, len(bb)+1)
	//    for n > 0 {
	//        result = append(result, byte(n&0xff))
	//        n >>= 8
	//    }
	result := make([]byte, 0, len(bb)+1)
	cpy := new(big.Int).SetBytes(n.Val.Bytes())
	for cpy.Cmp(Zero) == 1 {
		result = append(result, byte(cpy.Int64()&0xff))
		cpy.Rsh(cpy, 8)
	}

	// When the most significant byte already has the high bit set, an
	// additional high byte is required to indicate whether the number is
	// negative or positive.  The additional byte is removed when converting
	// back to an integral and its high bit is used to denote the sign.
	//
	// Otherwise, when the most significant byte does not already have the
	// high bit set, use it to indicate the value is negative, if needed.
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)
	} else if isNegative {
		result[len(result)-1] |= 0x80
	}

	return result
}

func MinimallyEncode(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	last := data[len(data)-1]
	if last&0x7f != 0 {
		return data
	}

	if len(data) == 1 {
		return []byte{}
	}

	if data[len(data)-2]&0x80 != 0 {
		return data
	}

	for i := len(data) - 1; i > 0; i-- {
		if data[i-1] != 0 {
			if data[i-1]&0x80 != 0 {
				data[i] = last
				i++
			} else {
				data[i-1] |= last
			}

			return data[:i]
		}
	}

	return []byte{}
}

// CheckMinimalDataEncoding returns whether the passed byte array adheres
// to the minimal encoding requirements.
func CheckMinimalDataEncoding(v []byte) error {
	if len(v) == 0 {
		return nil
	}

	// Check that the number is encoded with the minimum possible
	// number of bytes.
	//
	// If the most-significant-byte - excluding the sign bit - is zero
	// then we're not minimal.  Note how this test also rejects the
	// negative-zero encoding, [0x80].
	if v[len(v)-1]&0x7f == 0 {
		// One exception: if there's more than one byte and the most
		// significant bit of the second-most-significant-byte is set
		// it would conflict with the sign bit.  An example of this case
		// is +-255, which encode to 0xff00 and 0xff80 respectively.
		// (big-endian).
		if len(v) == 1 || v[len(v)-2]&0x80 == 0 {
			return errs.NewError(errs.ErrMinimalData, "numeric value encoded as %x is not minimally encoded", v)
		}
	}

	return nil
}
