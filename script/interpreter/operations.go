package interpreter

import (
	"bytes"
	"crypto/sha1" // nolint:gosec // OP_SHA1 support requires this
	"crypto/sha256"
	"hash"
	"math/big"

	"golang.org/x/crypto/ripemd160" // nolint:staticcheck // required

	ec "github.com/bsv-blockchain/go-sdk/v2/primitives/ec"
	crypto "github.com/bsv-blockchain/go-sdk/v2/primitives/hash"
	script "github.com/bsv-blockchain/go-sdk/v2/script"
	"github.com/bsv-blockchain/go-sdk/v2/script/interpreter/errs"
	"github.com/bsv-blockchain/go-sdk/v2/script/interpreter/scriptflag"
	"github.com/bsv-blockchain/go-sdk/v2/transaction"
	sighash "github.com/bsv-blockchain/go-sdk/v2/transaction/sighash"
)

// Conditional execution constants.
const (
	opCondFalse = 0
	opCondTrue  = 1
	opCondSkip  = 2
)

var (
	externalVerifySignatureFn func(payload, signature, publicKey []byte) bool = nil
)

func InjectExternalVerifySignatureFn(fn func(payload, signature, publicKey []byte) bool) {
	externalVerifySignatureFn = fn
}

type opcode struct {
	val    byte
	name   string
	length int
	exec   func(*ParsedOpcode, *thread) error
}

func (o opcode) Name() string {
	return o.name
}

// opcodeArray associates an opcode with its respective function, and defines them in order as to
// be correctly placed in an array
var opcodeArray = [256]opcode{
	// Data push opcodes.
	script.OpFALSE:     {script.OpFALSE, "OP_0", 1, opcodeFalse},
	script.OpDATA1:     {script.OpDATA1, "OP_DATA_1", 2, opcodePushData},
	script.OpDATA2:     {script.OpDATA2, "OP_DATA_2", 3, opcodePushData},
	script.OpDATA3:     {script.OpDATA3, "OP_DATA_3", 4, opcodePushData},
	script.OpDATA4:     {script.OpDATA4, "OP_DATA_4", 5, opcodePushData},
	script.OpDATA5:     {script.OpDATA5, "OP_DATA_5", 6, opcodePushData},
	script.OpDATA6:     {script.OpDATA6, "OP_DATA_6", 7, opcodePushData},
	script.OpDATA7:     {script.OpDATA7, "OP_DATA_7", 8, opcodePushData},
	script.OpDATA8:     {script.OpDATA8, "OP_DATA_8", 9, opcodePushData},
	script.OpDATA9:     {script.OpDATA9, "OP_DATA_9", 10, opcodePushData},
	script.OpDATA10:    {script.OpDATA10, "OP_DATA_10", 11, opcodePushData},
	script.OpDATA11:    {script.OpDATA11, "OP_DATA_11", 12, opcodePushData},
	script.OpDATA12:    {script.OpDATA12, "OP_DATA_12", 13, opcodePushData},
	script.OpDATA13:    {script.OpDATA13, "OP_DATA_13", 14, opcodePushData},
	script.OpDATA14:    {script.OpDATA14, "OP_DATA_14", 15, opcodePushData},
	script.OpDATA15:    {script.OpDATA15, "OP_DATA_15", 16, opcodePushData},
	script.OpDATA16:    {script.OpDATA16, "OP_DATA_16", 17, opcodePushData},
	script.OpDATA17:    {script.OpDATA17, "OP_DATA_17", 18, opcodePushData},
	script.OpDATA18:    {script.OpDATA18, "OP_DATA_18", 19, opcodePushData},
	script.OpDATA19:    {script.OpDATA19, "OP_DATA_19", 20, opcodePushData},
	script.OpDATA20:    {script.OpDATA20, "OP_DATA_20", 21, opcodePushData},
	script.OpDATA21:    {script.OpDATA21, "OP_DATA_21", 22, opcodePushData},
	script.OpDATA22:    {script.OpDATA22, "OP_DATA_22", 23, opcodePushData},
	script.OpDATA23:    {script.OpDATA23, "OP_DATA_23", 24, opcodePushData},
	script.OpDATA24:    {script.OpDATA24, "OP_DATA_24", 25, opcodePushData},
	script.OpDATA25:    {script.OpDATA25, "OP_DATA_25", 26, opcodePushData},
	script.OpDATA26:    {script.OpDATA26, "OP_DATA_26", 27, opcodePushData},
	script.OpDATA27:    {script.OpDATA27, "OP_DATA_27", 28, opcodePushData},
	script.OpDATA28:    {script.OpDATA28, "OP_DATA_28", 29, opcodePushData},
	script.OpDATA29:    {script.OpDATA29, "OP_DATA_29", 30, opcodePushData},
	script.OpDATA30:    {script.OpDATA30, "OP_DATA_30", 31, opcodePushData},
	script.OpDATA31:    {script.OpDATA31, "OP_DATA_31", 32, opcodePushData},
	script.OpDATA32:    {script.OpDATA32, "OP_DATA_32", 33, opcodePushData},
	script.OpDATA33:    {script.OpDATA33, "OP_DATA_33", 34, opcodePushData},
	script.OpDATA34:    {script.OpDATA34, "OP_DATA_34", 35, opcodePushData},
	script.OpDATA35:    {script.OpDATA35, "OP_DATA_35", 36, opcodePushData},
	script.OpDATA36:    {script.OpDATA36, "OP_DATA_36", 37, opcodePushData},
	script.OpDATA37:    {script.OpDATA37, "OP_DATA_37", 38, opcodePushData},
	script.OpDATA38:    {script.OpDATA38, "OP_DATA_38", 39, opcodePushData},
	script.OpDATA39:    {script.OpDATA39, "OP_DATA_39", 40, opcodePushData},
	script.OpDATA40:    {script.OpDATA40, "OP_DATA_40", 41, opcodePushData},
	script.OpDATA41:    {script.OpDATA41, "OP_DATA_41", 42, opcodePushData},
	script.OpDATA42:    {script.OpDATA42, "OP_DATA_42", 43, opcodePushData},
	script.OpDATA43:    {script.OpDATA43, "OP_DATA_43", 44, opcodePushData},
	script.OpDATA44:    {script.OpDATA44, "OP_DATA_44", 45, opcodePushData},
	script.OpDATA45:    {script.OpDATA45, "OP_DATA_45", 46, opcodePushData},
	script.OpDATA46:    {script.OpDATA46, "OP_DATA_46", 47, opcodePushData},
	script.OpDATA47:    {script.OpDATA47, "OP_DATA_47", 48, opcodePushData},
	script.OpDATA48:    {script.OpDATA48, "OP_DATA_48", 49, opcodePushData},
	script.OpDATA49:    {script.OpDATA49, "OP_DATA_49", 50, opcodePushData},
	script.OpDATA50:    {script.OpDATA50, "OP_DATA_50", 51, opcodePushData},
	script.OpDATA51:    {script.OpDATA51, "OP_DATA_51", 52, opcodePushData},
	script.OpDATA52:    {script.OpDATA52, "OP_DATA_52", 53, opcodePushData},
	script.OpDATA53:    {script.OpDATA53, "OP_DATA_53", 54, opcodePushData},
	script.OpDATA54:    {script.OpDATA54, "OP_DATA_54", 55, opcodePushData},
	script.OpDATA55:    {script.OpDATA55, "OP_DATA_55", 56, opcodePushData},
	script.OpDATA56:    {script.OpDATA56, "OP_DATA_56", 57, opcodePushData},
	script.OpDATA57:    {script.OpDATA57, "OP_DATA_57", 58, opcodePushData},
	script.OpDATA58:    {script.OpDATA58, "OP_DATA_58", 59, opcodePushData},
	script.OpDATA59:    {script.OpDATA59, "OP_DATA_59", 60, opcodePushData},
	script.OpDATA60:    {script.OpDATA60, "OP_DATA_60", 61, opcodePushData},
	script.OpDATA61:    {script.OpDATA61, "OP_DATA_61", 62, opcodePushData},
	script.OpDATA62:    {script.OpDATA62, "OP_DATA_62", 63, opcodePushData},
	script.OpDATA63:    {script.OpDATA63, "OP_DATA_63", 64, opcodePushData},
	script.OpDATA64:    {script.OpDATA64, "OP_DATA_64", 65, opcodePushData},
	script.OpDATA65:    {script.OpDATA65, "OP_DATA_65", 66, opcodePushData},
	script.OpDATA66:    {script.OpDATA66, "OP_DATA_66", 67, opcodePushData},
	script.OpDATA67:    {script.OpDATA67, "OP_DATA_67", 68, opcodePushData},
	script.OpDATA68:    {script.OpDATA68, "OP_DATA_68", 69, opcodePushData},
	script.OpDATA69:    {script.OpDATA69, "OP_DATA_69", 70, opcodePushData},
	script.OpDATA70:    {script.OpDATA70, "OP_DATA_70", 71, opcodePushData},
	script.OpDATA71:    {script.OpDATA71, "OP_DATA_71", 72, opcodePushData},
	script.OpDATA72:    {script.OpDATA72, "OP_DATA_72", 73, opcodePushData},
	script.OpDATA73:    {script.OpDATA73, "OP_DATA_73", 74, opcodePushData},
	script.OpDATA74:    {script.OpDATA74, "OP_DATA_74", 75, opcodePushData},
	script.OpDATA75:    {script.OpDATA75, "OP_DATA_75", 76, opcodePushData},
	script.OpPUSHDATA1: {script.OpPUSHDATA1, "OP_PUSHDATA1", -1, opcodePushData},
	script.OpPUSHDATA2: {script.OpPUSHDATA2, "OP_PUSHDATA2", -2, opcodePushData},
	script.OpPUSHDATA4: {script.OpPUSHDATA4, "OP_PUSHDATA4", -4, opcodePushData},
	script.Op1NEGATE:   {script.Op1NEGATE, "OP_1NEGATE", 1, opcode1Negate},
	script.OpRESERVED:  {script.OpRESERVED, "OP_RESERVED", 1, opcodeReserved},
	script.OpTRUE:      {script.OpTRUE, "OP_1", 1, opcodeN},
	script.Op2:         {script.Op2, "OP_2", 1, opcodeN},
	script.Op3:         {script.Op3, "OP_3", 1, opcodeN},
	script.Op4:         {script.Op4, "OP_4", 1, opcodeN},
	script.Op5:         {script.Op5, "OP_5", 1, opcodeN},
	script.Op6:         {script.Op6, "OP_6", 1, opcodeN},
	script.Op7:         {script.Op7, "OP_7", 1, opcodeN},
	script.Op8:         {script.Op8, "OP_8", 1, opcodeN},
	script.Op9:         {script.Op9, "OP_9", 1, opcodeN},
	script.Op10:        {script.Op10, "OP_10", 1, opcodeN},
	script.Op11:        {script.Op11, "OP_11", 1, opcodeN},
	script.Op12:        {script.Op12, "OP_12", 1, opcodeN},
	script.Op13:        {script.Op13, "OP_13", 1, opcodeN},
	script.Op14:        {script.Op14, "OP_14", 1, opcodeN},
	script.Op15:        {script.Op15, "OP_15", 1, opcodeN},
	script.Op16:        {script.Op16, "OP_16", 1, opcodeN},

	// Control opcodes.
	script.OpNOP:                 {script.OpNOP, "OP_NOP", 1, opcodeNop},
	script.OpVER:                 {script.OpVER, "OP_VER", 1, opcodeReserved},
	script.OpIF:                  {script.OpIF, "OP_IF", 1, opcodeIf},
	script.OpNOTIF:               {script.OpNOTIF, "OP_NOTIF", 1, opcodeNotIf},
	script.OpVERIF:               {script.OpVERIF, "OP_VERIF", 1, opcodeVerConditional},
	script.OpVERNOTIF:            {script.OpVERNOTIF, "OP_VERNOTIF", 1, opcodeVerConditional},
	script.OpELSE:                {script.OpELSE, "OP_ELSE", 1, opcodeElse},
	script.OpENDIF:               {script.OpENDIF, "OP_ENDIF", 1, opcodeEndif},
	script.OpVERIFY:              {script.OpVERIFY, "OP_VERIFY", 1, opcodeVerify},
	script.OpRETURN:              {script.OpRETURN, "OP_RETURN", 1, opcodeReturn},
	script.OpCHECKLOCKTIMEVERIFY: {script.OpCHECKLOCKTIMEVERIFY, "OP_CHECKLOCKTIMEVERIFY", 1, opcodeCheckLockTimeVerify},
	script.OpCHECKSEQUENCEVERIFY: {script.OpCHECKSEQUENCEVERIFY, "OP_CHECKSEQUENCEVERIFY", 1, opcodeCheckSequenceVerify},

	// Stack opcodes.
	script.OpTOALTSTACK:   {script.OpTOALTSTACK, "OP_TOALTSTACK", 1, opcodeToAltStack},
	script.OpFROMALTSTACK: {script.OpFROMALTSTACK, "OP_FROMALTSTACK", 1, opcodeFromAltStack},
	script.Op2DROP:        {script.Op2DROP, "OP_2DROP", 1, opcode2Drop},
	script.Op2DUP:         {script.Op2DUP, "OP_2DUP", 1, opcode2Dup},
	script.Op3DUP:         {script.Op3DUP, "OP_3DUP", 1, opcode3Dup},
	script.Op2OVER:        {script.Op2OVER, "OP_2OVER", 1, opcode2Over},
	script.Op2ROT:         {script.Op2ROT, "OP_2ROT", 1, opcode2Rot},
	script.Op2SWAP:        {script.Op2SWAP, "OP_2SWAP", 1, opcode2Swap},
	script.OpIFDUP:        {script.OpIFDUP, "OP_IFDUP", 1, opcodeIfDup},
	script.OpDEPTH:        {script.OpDEPTH, "OP_DEPTH", 1, opcodeDepth},
	script.OpDROP:         {script.OpDROP, "OP_DROP", 1, opcodeDrop},
	script.OpDUP:          {script.OpDUP, "OP_DUP", 1, opcodeDup},
	script.OpNIP:          {script.OpNIP, "OP_NIP", 1, opcodeNip},
	script.OpOVER:         {script.OpOVER, "OP_OVER", 1, opcodeOver},
	script.OpPICK:         {script.OpPICK, "OP_PICK", 1, opcodePick},
	script.OpROLL:         {script.OpROLL, "OP_ROLL", 1, opcodeRoll},
	script.OpROT:          {script.OpROT, "OP_ROT", 1, opcodeRot},
	script.OpSWAP:         {script.OpSWAP, "OP_SWAP", 1, opcodeSwap},
	script.OpTUCK:         {script.OpTUCK, "OP_TUCK", 1, opcodeTuck},

	// Splice opcodes.
	script.OpCAT:     {script.OpCAT, "OP_CAT", 1, opcodeCat},
	script.OpSPLIT:   {script.OpSPLIT, "OP_SPLIT", 1, opcodeSplit},
	script.OpNUM2BIN: {script.OpNUM2BIN, "OP_NUM2BIN", 1, opcodeNum2bin},
	script.OpBIN2NUM: {script.OpBIN2NUM, "OP_BIN2NUM", 1, opcodeBin2num},
	script.OpSIZE:    {script.OpSIZE, "OP_SIZE", 1, opcodeSize},

	// Bitwise logic opcodes.
	script.OpINVERT:      {script.OpINVERT, "OP_INVERT", 1, opcodeInvert},
	script.OpAND:         {script.OpAND, "OP_AND", 1, opcodeAnd},
	script.OpOR:          {script.OpOR, "OP_OR", 1, opcodeOr},
	script.OpXOR:         {script.OpXOR, "OP_XOR", 1, opcodeXor},
	script.OpEQUAL:       {script.OpEQUAL, "OP_EQUAL", 1, opcodeEqual},
	script.OpEQUALVERIFY: {script.OpEQUALVERIFY, "OP_EQUALVERIFY", 1, opcodeEqualVerify},
	script.OpRESERVED1:   {script.OpRESERVED1, "OP_RESERVED1", 1, opcodeReserved},
	script.OpRESERVED2:   {script.OpRESERVED2, "OP_RESERVED2", 1, opcodeReserved},

	// Numeric related opcodes.
	script.Op1ADD:               {script.Op1ADD, "OP_1ADD", 1, opcode1Add},
	script.Op1SUB:               {script.Op1SUB, "OP_1SUB", 1, opcode1Sub},
	script.Op2MUL:               {script.Op2MUL, "OP_2MUL", 1, opcodeDisabled},
	script.Op2DIV:               {script.Op2DIV, "OP_2DIV", 1, opcodeDisabled},
	script.OpNEGATE:             {script.OpNEGATE, "OP_NEGATE", 1, opcodeNegate},
	script.OpABS:                {script.OpABS, "OP_ABS", 1, opcodeAbs},
	script.OpNOT:                {script.OpNOT, "OP_NOT", 1, opcodeNot},
	script.Op0NOTEQUAL:          {script.Op0NOTEQUAL, "OP_0NOTEQUAL", 1, opcode0NotEqual},
	script.OpADD:                {script.OpADD, "OP_ADD", 1, opcodeAdd},
	script.OpSUB:                {script.OpSUB, "OP_SUB", 1, opcodeSub},
	script.OpMUL:                {script.OpMUL, "OP_MUL", 1, opcodeMul},
	script.OpDIV:                {script.OpDIV, "OP_DIV", 1, opcodeDiv},
	script.OpMOD:                {script.OpMOD, "OP_MOD", 1, opcodeMod},
	script.OpLSHIFT:             {script.OpLSHIFT, "OP_LSHIFT", 1, opcodeLShift},
	script.OpRSHIFT:             {script.OpRSHIFT, "OP_RSHIFT", 1, opcodeRShift},
	script.OpBOOLAND:            {script.OpBOOLAND, "OP_BOOLAND", 1, opcodeBoolAnd},
	script.OpBOOLOR:             {script.OpBOOLOR, "OP_BOOLOR", 1, opcodeBoolOr},
	script.OpNUMEQUAL:           {script.OpNUMEQUAL, "OP_NUMEQUAL", 1, opcodeNumEqual},
	script.OpNUMEQUALVERIFY:     {script.OpNUMEQUALVERIFY, "OP_NUMEQUALVERIFY", 1, opcodeNumEqualVerify},
	script.OpNUMNOTEQUAL:        {script.OpNUMNOTEQUAL, "OP_NUMNOTEQUAL", 1, opcodeNumNotEqual},
	script.OpLESSTHAN:           {script.OpLESSTHAN, "OP_LESSTHAN", 1, opcodeLessThan},
	script.OpGREATERTHAN:        {script.OpGREATERTHAN, "OP_GREATERTHAN", 1, opcodeGreaterThan},
	script.OpLESSTHANOREQUAL:    {script.OpLESSTHANOREQUAL, "OP_LESSTHANOREQUAL", 1, opcodeLessThanOrEqual},
	script.OpGREATERTHANOREQUAL: {script.OpGREATERTHANOREQUAL, "OP_GREATERTHANOREQUAL", 1, opcodeGreaterThanOrEqual},
	script.OpMIN:                {script.OpMIN, "OP_MIN", 1, opcodeMin},
	script.OpMAX:                {script.OpMAX, "OP_MAX", 1, opcodeMax},
	script.OpWITHIN:             {script.OpWITHIN, "OP_WITHIN", 1, opcodeWithin},

	// Crypto opcodes.
	script.OpRIPEMD160:           {script.OpRIPEMD160, "OP_RIPEMD160", 1, opcodeRipemd160},
	script.OpSHA1:                {script.OpSHA1, "OP_SHA1", 1, opcodeSha1},
	script.OpSHA256:              {script.OpSHA256, "OP_SHA256", 1, opcodeSha256},
	script.OpHASH160:             {script.OpHASH160, "OP_HASH160", 1, opcodeHash160},
	script.OpHASH256:             {script.OpHASH256, "OP_HASH256", 1, opcodeHash256},
	script.OpCODESEPARATOR:       {script.OpCODESEPARATOR, "OP_CODESEPARATOR", 1, opcodeCodeSeparator},
	script.OpCHECKSIG:            {script.OpCHECKSIG, "OP_CHECKSIG", 1, opcodeCheckSig},
	script.OpCHECKSIGVERIFY:      {script.OpCHECKSIGVERIFY, "OP_CHECKSIGVERIFY", 1, opcodeCheckSigVerify},
	script.OpCHECKMULTISIG:       {script.OpCHECKMULTISIG, "OP_CHECKMULTISIG", 1, opcodeCheckMultiSig},
	script.OpCHECKMULTISIGVERIFY: {script.OpCHECKMULTISIGVERIFY, "OP_CHECKMULTISIGVERIFY", 1, opcodeCheckMultiSigVerify},

	// Reserved opcodes.
	script.OpNOP1:  {script.OpNOP1, "OP_NOP1", 1, opcodeNop},
	script.OpNOP4:  {script.OpNOP4, "OP_NOP4", 1, opcodeNop},
	script.OpNOP5:  {script.OpNOP5, "OP_NOP5", 1, opcodeNop},
	script.OpNOP6:  {script.OpNOP6, "OP_NOP6", 1, opcodeNop},
	script.OpNOP7:  {script.OpNOP7, "OP_NOP7", 1, opcodeNop},
	script.OpNOP8:  {script.OpNOP8, "OP_NOP8", 1, opcodeNop},
	script.OpNOP9:  {script.OpNOP9, "OP_NOP9", 1, opcodeNop},
	script.OpNOP10: {script.OpNOP10, "OP_NOP10", 1, opcodeNop},

	// Undefined opcodes.
	script.OpUNKNOWN186: {script.OpUNKNOWN186, "OP_UNKNOWN186", 1, opcodeInvalid},
	script.OpUNKNOWN187: {script.OpUNKNOWN187, "OP_UNKNOWN187", 1, opcodeInvalid},
	script.OpUNKNOWN188: {script.OpUNKNOWN188, "OP_UNKNOWN188", 1, opcodeInvalid},
	script.OpUNKNOWN189: {script.OpUNKNOWN189, "OP_UNKNOWN189", 1, opcodeInvalid},
	script.OpUNKNOWN190: {script.OpUNKNOWN190, "OP_UNKNOWN190", 1, opcodeInvalid},
	script.OpUNKNOWN191: {script.OpUNKNOWN191, "OP_UNKNOWN191", 1, opcodeInvalid},
	script.OpUNKNOWN192: {script.OpUNKNOWN192, "OP_UNKNOWN192", 1, opcodeInvalid},
	script.OpUNKNOWN193: {script.OpUNKNOWN193, "OP_UNKNOWN193", 1, opcodeInvalid},
	script.OpUNKNOWN194: {script.OpUNKNOWN194, "OP_UNKNOWN194", 1, opcodeInvalid},
	script.OpUNKNOWN195: {script.OpUNKNOWN195, "OP_UNKNOWN195", 1, opcodeInvalid},
	script.OpUNKNOWN196: {script.OpUNKNOWN196, "OP_UNKNOWN196", 1, opcodeInvalid},
	script.OpUNKNOWN197: {script.OpUNKNOWN197, "OP_UNKNOWN197", 1, opcodeInvalid},
	script.OpUNKNOWN198: {script.OpUNKNOWN198, "OP_UNKNOWN198", 1, opcodeInvalid},
	script.OpUNKNOWN199: {script.OpUNKNOWN199, "OP_UNKNOWN199", 1, opcodeInvalid},
	script.OpUNKNOWN200: {script.OpUNKNOWN200, "OP_UNKNOWN200", 1, opcodeInvalid},
	script.OpUNKNOWN201: {script.OpUNKNOWN201, "OP_UNKNOWN201", 1, opcodeInvalid},
	script.OpUNKNOWN202: {script.OpUNKNOWN202, "OP_UNKNOWN202", 1, opcodeInvalid},
	script.OpUNKNOWN203: {script.OpUNKNOWN203, "OP_UNKNOWN203", 1, opcodeInvalid},
	script.OpUNKNOWN204: {script.OpUNKNOWN204, "OP_UNKNOWN204", 1, opcodeInvalid},
	script.OpUNKNOWN205: {script.OpUNKNOWN205, "OP_UNKNOWN205", 1, opcodeInvalid},
	script.OpUNKNOWN206: {script.OpUNKNOWN206, "OP_UNKNOWN206", 1, opcodeInvalid},
	script.OpUNKNOWN207: {script.OpUNKNOWN207, "OP_UNKNOWN207", 1, opcodeInvalid},
	script.OpUNKNOWN208: {script.OpUNKNOWN208, "OP_UNKNOWN208", 1, opcodeInvalid},
	script.OpUNKNOWN209: {script.OpUNKNOWN209, "OP_UNKNOWN209", 1, opcodeInvalid},
	script.OpUNKNOWN210: {script.OpUNKNOWN210, "OP_UNKNOWN210", 1, opcodeInvalid},
	script.OpUNKNOWN211: {script.OpUNKNOWN211, "OP_UNKNOWN211", 1, opcodeInvalid},
	script.OpUNKNOWN212: {script.OpUNKNOWN212, "OP_UNKNOWN212", 1, opcodeInvalid},
	script.OpUNKNOWN213: {script.OpUNKNOWN213, "OP_UNKNOWN213", 1, opcodeInvalid},
	script.OpUNKNOWN214: {script.OpUNKNOWN214, "OP_UNKNOWN214", 1, opcodeInvalid},
	script.OpUNKNOWN215: {script.OpUNKNOWN215, "OP_UNKNOWN215", 1, opcodeInvalid},
	script.OpUNKNOWN216: {script.OpUNKNOWN216, "OP_UNKNOWN216", 1, opcodeInvalid},
	script.OpUNKNOWN217: {script.OpUNKNOWN217, "OP_UNKNOWN217", 1, opcodeInvalid},
	script.OpUNKNOWN218: {script.OpUNKNOWN218, "OP_UNKNOWN218", 1, opcodeInvalid},
	script.OpUNKNOWN219: {script.OpUNKNOWN219, "OP_UNKNOWN219", 1, opcodeInvalid},
	script.OpUNKNOWN220: {script.OpUNKNOWN220, "OP_UNKNOWN220", 1, opcodeInvalid},
	script.OpUNKNOWN221: {script.OpUNKNOWN221, "OP_UNKNOWN221", 1, opcodeInvalid},
	script.OpUNKNOWN222: {script.OpUNKNOWN222, "OP_UNKNOWN222", 1, opcodeInvalid},
	script.OpUNKNOWN223: {script.OpUNKNOWN223, "OP_UNKNOWN223", 1, opcodeInvalid},
	script.OpUNKNOWN224: {script.OpUNKNOWN224, "OP_UNKNOWN224", 1, opcodeInvalid},
	script.OpUNKNOWN225: {script.OpUNKNOWN225, "OP_UNKNOWN225", 1, opcodeInvalid},
	script.OpUNKNOWN226: {script.OpUNKNOWN226, "OP_UNKNOWN226", 1, opcodeInvalid},
	script.OpUNKNOWN227: {script.OpUNKNOWN227, "OP_UNKNOWN227", 1, opcodeInvalid},
	script.OpUNKNOWN228: {script.OpUNKNOWN228, "OP_UNKNOWN228", 1, opcodeInvalid},
	script.OpUNKNOWN229: {script.OpUNKNOWN229, "OP_UNKNOWN229", 1, opcodeInvalid},
	script.OpUNKNOWN230: {script.OpUNKNOWN230, "OP_UNKNOWN230", 1, opcodeInvalid},
	script.OpUNKNOWN231: {script.OpUNKNOWN231, "OP_UNKNOWN231", 1, opcodeInvalid},
	script.OpUNKNOWN232: {script.OpUNKNOWN232, "OP_UNKNOWN232", 1, opcodeInvalid},
	script.OpUNKNOWN233: {script.OpUNKNOWN233, "OP_UNKNOWN233", 1, opcodeInvalid},
	script.OpUNKNOWN234: {script.OpUNKNOWN234, "OP_UNKNOWN234", 1, opcodeInvalid},
	script.OpUNKNOWN235: {script.OpUNKNOWN235, "OP_UNKNOWN235", 1, opcodeInvalid},
	script.OpUNKNOWN236: {script.OpUNKNOWN236, "OP_UNKNOWN236", 1, opcodeInvalid},
	script.OpUNKNOWN237: {script.OpUNKNOWN237, "OP_UNKNOWN237", 1, opcodeInvalid},
	script.OpUNKNOWN238: {script.OpUNKNOWN238, "OP_UNKNOWN238", 1, opcodeInvalid},
	script.OpUNKNOWN239: {script.OpUNKNOWN239, "OP_UNKNOWN239", 1, opcodeInvalid},
	script.OpUNKNOWN240: {script.OpUNKNOWN240, "OP_UNKNOWN240", 1, opcodeInvalid},
	script.OpUNKNOWN241: {script.OpUNKNOWN241, "OP_UNKNOWN241", 1, opcodeInvalid},
	script.OpUNKNOWN242: {script.OpUNKNOWN242, "OP_UNKNOWN242", 1, opcodeInvalid},
	script.OpUNKNOWN243: {script.OpUNKNOWN243, "OP_UNKNOWN243", 1, opcodeInvalid},
	script.OpUNKNOWN244: {script.OpUNKNOWN244, "OP_UNKNOWN244", 1, opcodeInvalid},
	script.OpUNKNOWN245: {script.OpUNKNOWN245, "OP_UNKNOWN245", 1, opcodeInvalid},
	script.OpUNKNOWN246: {script.OpUNKNOWN246, "OP_UNKNOWN246", 1, opcodeInvalid},
	script.OpUNKNOWN247: {script.OpUNKNOWN247, "OP_UNKNOWN247", 1, opcodeInvalid},
	script.OpUNKNOWN248: {script.OpUNKNOWN248, "OP_UNKNOWN248", 1, opcodeInvalid},
	script.OpUNKNOWN249: {script.OpUNKNOWN249, "OP_UNKNOWN249", 1, opcodeInvalid},

	// Bitcoin Core internal use opcode.  Defined here for completeness.
	script.OpSMALLINTEGER: {script.OpSMALLINTEGER, "OP_SMALLINTEGER", 1, opcodeInvalid},
	script.OpPUBKEYS:      {script.OpPUBKEYS, "OP_PUBKEYS", 1, opcodeInvalid},
	script.OpUNKNOWN252:   {script.OpUNKNOWN252, "OP_UNKNOWN252", 1, opcodeInvalid},
	script.OpPUBKEYHASH:   {script.OpPUBKEYHASH, "OP_PUBKEYHASH", 1, opcodeInvalid},
	script.OpPUBKEY:       {script.OpPUBKEY, "OP_PUBKEY", 1, opcodeInvalid},

	script.OpINVALIDOPCODE: {script.OpINVALIDOPCODE, "OP_INVALIDOPCODE", 1, opcodeInvalid},
}

// *******************************************
// Opcode implementation functions start here.
// *******************************************

// opcodeDisabled is a common handler for disabled opcodes.  It returns an
// appropriate error indicating the opcode is disabled.  While it would
// ordinarily make more sense to detect if the script contains any disabled
// opcodes before executing in an initial parse step, the consensus rules
// dictate the script doesn't fail until the program counter passes over a
// disabled opcode (even when they appear in a branch that is not executed).
func opcodeDisabled(op *ParsedOpcode, t *thread) error {
	return errs.NewError(errs.ErrDisabledOpcode, "attempt to execute disabled opcode %s", op.Name())
}

func opcodeVerConditional(op *ParsedOpcode, t *thread) error {
	if t.afterGenesis && !t.shouldExec(*op) {
		return nil
	}
	return opcodeReserved(op, t)
}

// opcodeReserved is a common handler for all reserved opcodes.  It returns an
// appropriate error indicating the opcode is reserved.
func opcodeReserved(op *ParsedOpcode, t *thread) error {
	return errs.NewError(errs.ErrReservedOpcode, "attempt to execute reserved opcode %s", op.Name())
}

// opcodeInvalid is a common handler for all invalid opcodes.  It returns an
// appropriate error indicating the opcode is invalid.
func opcodeInvalid(op *ParsedOpcode, t *thread) error {
	return errs.NewError(errs.ErrReservedOpcode, "attempt to execute invalid opcode %s", op.Name())
}

// opcodeFalse pushes an empty array to the data stack to represent false.  Note
// that 0, when encoded as a number according to the numeric encoding consensus
// rules, is an empty array.
func opcodeFalse(op *ParsedOpcode, t *thread) error {
	t.dstack.PushByteArray(nil)
	return nil
}

// opcodePushData is a common handler for the vast majority of opcodes that push
// raw data (bytes) to the data stack.
func opcodePushData(op *ParsedOpcode, t *thread) error {
	t.dstack.PushByteArray(op.Data)
	return nil
}

// opcode1Negate pushes -1, encoded as a number, to the data stack.
func opcode1Negate(op *ParsedOpcode, t *thread) error {
	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(-1),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeN is a common handler for the small integer data push opcodes.  It
// pushes the numeric value the opcode represents (which will be from 1 to 16)
// onto the data stack.
func opcodeN(op *ParsedOpcode, t *thread) error {
	// The opcodes are all defined consecutively, so the numeric value is
	// the difference.
	t.dstack.PushByteArray([]byte{(op.op.val - (script.Op1 - 1))})
	return nil
}

// opcodeNop is a common handler for the NOP family of opcodes.  As the name
// implies it generally does nothing, however, it will return an error when
// the flag to discourage use of NOPs is set for select opcodes.
func opcodeNop(op *ParsedOpcode, t *thread) error {
	switch op.op.val {
	case script.OpNOP1, script.OpNOP4, script.OpNOP5,
		script.OpNOP6, script.OpNOP7, script.OpNOP8, script.OpNOP9, script.OpNOP10:
		if t.hasFlag(scriptflag.DiscourageUpgradableNops) {
			return errs.NewError(
				errs.ErrDiscourageUpgradableNOPs,
				"script.OpNOP%d reserved for soft-fork upgrades",
				op.op.val-(script.OpNOP1-1),
			)
		}
	}

	return nil
}

// popIfBool pops the top item off the stack and returns a bool
func popIfBool(t *thread) (bool, error) {
	if t.hasFlag(scriptflag.VerifyMinimalIf) {
		b, err := t.dstack.PopByteArray()
		if err != nil {
			return false, err
		}

		if len(b) > 1 {
			return false, errs.NewError(errs.ErrMinimalIf, "conditionl has data of length %d", len(b))
		}
		if len(b) == 1 && b[0] != 1 {
			return false, errs.NewError(errs.ErrMinimalIf, "conditional failed")
		}

		return asBool(b), nil
	}

	return t.dstack.PopBool()
}

// opcodeIf treats the top item on the data stack as a boolean and removes it.
//
// An appropriate entry is added to the conditional stack depending on whether
// the boolean is true and whether this if is on an executing branch in order
// to allow proper execution of further opcodes depending on the conditional
// logic.  When the boolean is true, the first branch will be executed (unless
// this opcode is nested in a non-executed branch).
//
// <expression> if [statements] [else [statements]] endif
//
// Note that, unlike for all non-conditional opcodes, this is executed even when
// it is on a non-executing branch so proper nesting is maintained.
//
// Data stack transformation: [... bool] -> [...]
// Conditional stack transformation: [...] -> [... OpCondValue]
func opcodeIf(op *ParsedOpcode, t *thread) error {
	condVal := opCondFalse
	if t.shouldExec(*op) {
		if t.isBranchExecuting() {
			ok, err := popIfBool(t)
			if err != nil {
				return err
			}

			if ok {
				condVal = opCondTrue
			}
		} else {
			condVal = opCondSkip
		}
	}

	t.condStack = append(t.condStack, condVal)
	t.elseStack.PushBool(false)
	return nil
}

// opcodeNotIf treats the top item on the data stack as a boolean and removes
// it.
//
// An appropriate entry is added to the conditional stack depending on whether
// the boolean is true and whether this if is on an executing branch in order
// to allow proper execution of further opcodes depending on the conditional
// logic.  When the boolean is false, the first branch will be executed (unless
// this opcode is nested in a non-executed branch).
//
// <expression> notif [statements] [else [statements]] endif
//
// Note that, unlike for all non-conditional opcodes, this is executed even when
// it is on a non-executing branch so proper nesting is maintained.
//
// Data stack transformation: [... bool] -> [...]
// Conditional stack transformation: [...] -> [... OpCondValue]
func opcodeNotIf(op *ParsedOpcode, t *thread) error {
	condVal := opCondFalse
	if t.shouldExec(*op) {
		if t.isBranchExecuting() {
			ok, err := popIfBool(t)
			if err != nil {
				return err
			}

			if !ok {
				condVal = opCondTrue
			}
		} else {
			condVal = opCondSkip
		}
	}

	t.condStack = append(t.condStack, condVal)
	t.elseStack.PushBool(false)
	return nil
}

// opcodeElse inverts conditional execution for other half of if/else/endif.
//
// An error is returned if there has not already been a matching script.OpIF.
//
// Conditional stack transformation: [... OpCondValue] -> [... !OpCondValue]
func opcodeElse(op *ParsedOpcode, t *thread) error {
	if len(t.condStack) == 0 {
		return errs.NewError(errs.ErrUnbalancedConditional,
			"encountered opcode %s with no matching opcode to begin conditional execution", op.Name())
	}

	// Only one ELSE allowed in IF after genesis
	ok, err := t.elseStack.PopBool()
	if err != nil {
		return err
	}
	if ok {
		return errs.NewError(errs.ErrUnbalancedConditional,
			"encountered opcode %s with no matching opcode to begin conditional execution", op.Name())
	}

	conditionalIdx := len(t.condStack) - 1
	switch t.condStack[conditionalIdx] {
	case opCondTrue:
		t.condStack[conditionalIdx] = opCondFalse
	case opCondFalse:
		t.condStack[conditionalIdx] = opCondTrue
	case opCondSkip:
		// Value doesn't change in skip since it indicates this opcode
		// is nested in a non-executed branch.
	}

	t.elseStack.PushBool(true)
	return nil
}

// opcodeEndif terminates a conditional block, removing the value from the
// conditional execution stack.
//
// An error is returned if there has not already been a matching script.OpIF.
//
// Conditional stack transformation: [... OpCondValue] -> [...]
func opcodeEndif(op *ParsedOpcode, t *thread) error {
	if len(t.condStack) == 0 {
		return errs.NewError(errs.ErrUnbalancedConditional,
			"encountered opcode %s with no matching opcode to begin conditional execution", op.Name())
	}

	t.condStack = t.condStack[:len(t.condStack)-1]
	if _, err := t.elseStack.PopBool(); err != nil {
		return err
	}

	return nil
}

// abstractVerify examines the top item on the data stack as a boolean value and
// verifies it evaluates to true.  An error is returned either when there is no
// item on the stack or when that item evaluates to false.  In the latter case
// where the verification fails specifically due to the top item evaluating
// to false, the returned error will use the passed error code.
func abstractVerify(op *ParsedOpcode, t *thread, c errs.ErrorCode) error {
	verified, err := t.dstack.PopBool()
	if err != nil {
		return err
	}
	if !verified {
		return errs.NewError(c, "%s failed", op.Name())
	}

	return nil
}

// opcodeVerify examines the top item on the data stack as a boolean value and
// verifies it evaluates to true.  An error is returned if it does not.
func opcodeVerify(op *ParsedOpcode, t *thread) error {
	return abstractVerify(op, t, errs.ErrVerify)
}

// opcodeReturn returns an appropriate error since it is always an error to
// return early from a script.
func opcodeReturn(op *ParsedOpcode, t *thread) error {
	if !t.afterGenesis {
		return errs.NewError(errs.ErrEarlyReturn, "script returned early")
	}

	t.earlyReturnAfterGenesis = true
	if len(t.condStack) == 0 {
		// Terminate the execution as successful. The remaining of the script does not affect the validity (even in
		// presence of unbalanced IFs, invalid opcodes etc)
		return success()
	}

	return nil
}

// verifyLockTime is a helper function used to validate locktimes.
func verifyLockTime(txLockTime, threshold, lockTime int64) error {
	// The lockTimes in both the script and transaction must be of the same
	// type.
	if (txLockTime < threshold && lockTime >= threshold) ||
		(txLockTime >= threshold && lockTime < threshold) {
		return errs.NewError(errs.ErrUnsatisfiedLockTime,
			"mismatched locktime types -- tx locktime %d, stack locktime %d", txLockTime, lockTime)
	}

	if lockTime > txLockTime {
		return errs.NewError(errs.ErrUnsatisfiedLockTime,
			"locktime requirement not satisfied -- locktime is greater than the transaction locktime: %d > %d",
			lockTime, txLockTime)
	}

	return nil
}

// opcodeCheckLockTimeVerify compares the top item on the data stack to the
// LockTime field of the transaction containing the script signature
// validating if the transaction outputs are spendable yet.  If flag
// ScriptVerifyCheckLockTimeVerify is not set, the code continues as if script.OpNOP2
// were executed.
func opcodeCheckLockTimeVerify(op *ParsedOpcode, t *thread) error {
	// If the ScriptVerifyCheckLockTimeVerify script flag is not set, treat
	// opcode as script.OpNOP2 instead.
	if !t.hasFlag(scriptflag.VerifyCheckLockTimeVerify) || t.afterGenesis {
		if t.hasFlag(scriptflag.DiscourageUpgradableNops) {
			return errs.NewError(errs.ErrDiscourageUpgradableNOPs, "script.OpNOP2 reserved for soft-fork upgrades")
		}

		return nil
	}

	// The current transaction locktime is a uint32 resulting in a maximum
	// locktime of 2^32-1 (the year 2106).  However, scriptNums are signed
	// and therefore a standard 4-byte scriptNum would only support up to a
	// maximum of 2^31-1 (the year 2038).  Thus, a 5-byte scriptNum is used
	// here since it will support up to 2^39-1 which allows dates beyond the
	// current locktime limit.
	//
	// PeekByteArray is used here instead of PeekInt because we do not want
	// to be limited to a 4-byte integer for reasons specified above.
	so, err := t.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	lockTime, err := MakeScriptNumber(so, 5, t.dstack.verifyMinimalData, t.afterGenesis)
	if err != nil {
		return err
	}

	// In the rare event that the argument needs to be < 0 due to some
	// arithmetic being done first, you can always use
	// 0 script.OpMAX script.OpCHECKLOCKTIMEVERIFY.
	if lockTime.LessThanInt(0) {
		return errs.NewError(errs.ErrNegativeLockTime, "negative lock time: %d", lockTime.Int64())
	}

	// The lock time field of a transaction is either a block height at
	// which the transaction is finalized or a timestamp depending on if the
	// value is before the interpreter.LockTimeThreshold.  When it is under the
	// threshold it is a block height.
	if err = verifyLockTime(int64(t.tx.LockTime), LockTimeThreshold, lockTime.Int64()); err != nil {
		return err
	}

	// The lock time feature can also be disabled, thereby bypassing
	// script.OpCHECKLOCKTIMEVERIFY, if every transaction input has been finalized by
	// setting its sequence to the maximum value (transaction.MaxTxInSequenceNum).  This
	// condition would result in the transaction being allowed into the blockchain
	// making the opcode ineffective.
	//
	// This condition is prevented by enforcing that the input being used by
	// the opcode is unlocked (its sequence number is less than the max
	// value).  This is sufficient to prove correctness without having to
	// check every input.
	//
	// NOTE: This implies that even if the transaction is not finalized due to
	// another input being unlocked, the opcode execution will still fail when the
	// input being used by the opcode is locked.
	if t.tx.Inputs[t.inputIdx].SequenceNumber == transaction.MaxTxInSequenceNum {
		return errs.NewError(errs.ErrUnsatisfiedLockTime, "transaction input is finalized")
	}

	return nil
}

// opcodeCheckSequenceVerify compares the top item on the data stack to the
// LockTime field of the transaction containing the script signature
// validating if the transaction outputs are spendable yet.  If flag
// ScriptVerifyCheckSequenceVerify is not set, the code continues as if script.OpNOP3
// were executed.
func opcodeCheckSequenceVerify(op *ParsedOpcode, t *thread) error {
	// If the ScriptVerifyCheckSequenceVerify script flag is not set, treat
	// opcode as script.OpNOP3 instead.
	if !t.hasFlag(scriptflag.VerifyCheckSequenceVerify) || t.afterGenesis {
		if t.hasFlag(scriptflag.DiscourageUpgradableNops) {
			return errs.NewError(errs.ErrDiscourageUpgradableNOPs, "script.OpNOP3 reserved for soft-fork upgrades")
		}

		return nil
	}

	// The current transaction sequence is a uint32 resulting in a maximum
	// sequence of 2^32-1.  However, scriptNums are signed and therefore a
	// standard 4-byte scriptNum would only support up to a maximum of
	// 2^31-1.  Thus, a 5-byte scriptNum is used here since it will support
	// up to 2^39-1 which allows sequences beyond the current sequence
	// limit.
	//
	// PeekByteArray is used here instead of PeekInt because we do not want
	// to be limited to a 4-byte integer for reasons specified above.
	so, err := t.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}
	stackSequence, err := MakeScriptNumber(so, 5, t.dstack.verifyMinimalData, t.afterGenesis)
	if err != nil {
		return err
	}

	// In the rare event that the argument needs to be < 0 due to some
	// arithmetic being done first, you can always use
	// 0 script.OpMAX script.OpCHECKSEQUENCEVERIFY.
	if stackSequence.LessThanInt(0) {
		return errs.NewError(errs.ErrNegativeLockTime, "negative sequence: %d", stackSequence.Int64())
	}

	sequence := stackSequence.Int64()

	// To provide for future soft-fork extensibility, if the
	// operand has the disabled lock-time flag set,
	// CHECKSEQUENCEVERIFY behaves as a NOP.
	if sequence&int64(transaction.SequenceLockTimeDisabled) != 0 {
		return nil
	}

	// Transaction version numbers not high enough to trigger CSV rules must
	// fail.
	if t.tx.Version < 2 {
		return errs.NewError(errs.ErrUnsatisfiedLockTime, "invalid transaction version: %d", t.tx.Version)
	}

	// Sequence numbers with their most significant bit set are not
	// consensus constrained. Testing that the transaction's sequence
	// number does not have this bit set prevents using this property
	// to get around a CHECKSEQUENCEVERIFY check.
	txSequence := int64(t.tx.Inputs[t.inputIdx].SequenceNumber)
	if txSequence&int64(transaction.SequenceLockTimeDisabled) != 0 {
		return errs.NewError(errs.ErrUnsatisfiedLockTime,
			"transaction sequence has sequence locktime disabled bit set: 0x%x", txSequence)
	}

	// Mask off non-consensus bits before doing comparisons.
	lockTimeMask := int64(transaction.SequenceLockTimeIsSeconds | transaction.SequenceLockTimeMask)

	return verifyLockTime(txSequence&lockTimeMask, transaction.SequenceLockTimeIsSeconds, sequence&lockTimeMask)
}

// opcodeToAltStack removes the top item from the main data stack and pushes it
// onto the alternate data stack.
//
// Main data stack transformation: [... x1 x2 x3] -> [... x1 x2]
// Alt data stack transformation:  [... y1 y2 y3] -> [... y1 y2 y3 x3]
func opcodeToAltStack(op *ParsedOpcode, t *thread) error {
	so, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	t.astack.PushByteArray(so)

	return nil
}

// opcodeFromAltStack removes the top item from the alternate data stack and
// pushes it onto the main data stack.
//
// Main data stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 y3]
// Alt data stack transformation:  [... y1 y2 y3] -> [... y1 y2]
func opcodeFromAltStack(op *ParsedOpcode, t *thread) error {
	so, err := t.astack.PopByteArray()
	if err != nil {
		return err
	}

	t.dstack.PushByteArray(so)

	return nil
}

// opcode2Drop removes the top 2 items from the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1]
func opcode2Drop(op *ParsedOpcode, t *thread) error {
	return t.dstack.DropN(2)
}

// opcode2Dup duplicates the top 2 items on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x2 x3]
func opcode2Dup(op *ParsedOpcode, t *thread) error {
	return t.dstack.DupN(2)
}

// opcode3Dup duplicates the top 3 items on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x1 x2 x3]
func opcode3Dup(op *ParsedOpcode, t *thread) error {
	return t.dstack.DupN(3)
}

// opcode2Over duplicates the 2 items before the top 2 items on the data stack.
//
// Stack transformation: [... x1 x2 x3 x4] -> [... x1 x2 x3 x4 x1 x2]
func opcode2Over(op *ParsedOpcode, t *thread) error {
	return t.dstack.OverN(2)
}

// opcode2Rot rotates the top 6 items on the data stack to the left twice.
//
// Stack transformation: [... x1 x2 x3 x4 x5 x6] -> [... x3 x4 x5 x6 x1 x2]
func opcode2Rot(op *ParsedOpcode, t *thread) error {
	return t.dstack.RotN(2)
}

// opcode2Swap swaps the top 2 items on the data stack with the 2 that come
// before them.
//
// Stack transformation: [... x1 x2 x3 x4] -> [... x3 x4 x1 x2]
func opcode2Swap(op *ParsedOpcode, t *thread) error {
	return t.dstack.SwapN(2)
}

// opcodeIfDup duplicates the top item of the stack if it is not zero.
//
// Stack transformation (x1==0): [... x1] -> [... x1]
// Stack transformation (x1!=0): [... x1] -> [... x1 x1]
func opcodeIfDup(op *ParsedOpcode, t *thread) error {
	so, err := t.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

	// Push copy of data iff it isn't zero
	if asBool(so) {
		t.dstack.PushByteArray(so)
	}

	return nil
}

// opcodeDepth pushes the depth of the data stack prior to executing this
// opcode, encoded as a number, onto the data stack.
//
// Stack transformation: [...] -> [... <num of items on the stack>]
// Example with 2 items: [x1 x2] -> [x1 x2 2]
// Example with 3 items: [x1 x2 x3] -> [x1 x2 x3 3]
func opcodeDepth(op *ParsedOpcode, t *thread) error {
	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(int64(t.dstack.Depth())),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeDrop removes the top item from the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2]
func opcodeDrop(op *ParsedOpcode, t *thread) error {
	return t.dstack.DropN(1)
}

// opcodeDup duplicates the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x3]
func opcodeDup(op *ParsedOpcode, t *thread) error {
	return t.dstack.DupN(1)
}

// opcodeNip removes the item before the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x3]
func opcodeNip(op *ParsedOpcode, t *thread) error {
	return t.dstack.NipN(1)
}

// opcodeOver duplicates the item before the top item on the data stack.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2 x3 x2]
func opcodeOver(op *ParsedOpcode, t *thread) error {
	return t.dstack.OverN(1)
}

// opcodePick treats the top item on the data stack as an integer and duplicates
// the item on the stack that number of items back to the top.
//
// Stack transformation: [xn ... x2 x1 x0 n] -> [xn ... x2 x1 x0 xn]
// Example with n=1: [x2 x1 x0 1] -> [x2 x1 x0 x1]
// Example with n=2: [x2 x1 x0 2] -> [x2 x1 x0 x2]
func opcodePick(op *ParsedOpcode, t *thread) error {
	val, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	return t.dstack.PickN(val.Int32())
}

// opcodeRoll treats the top item on the data stack as an integer and moves
// the item on the stack that number of items back to the top.
//
// Stack transformation: [xn ... x2 x1 x0 n] -> [... x2 x1 x0 xn]
// Example with n=1: [x2 x1 x0 1] -> [x2 x0 x1]
// Example with n=2: [x2 x1 x0 2] -> [x1 x0 x2]
func opcodeRoll(op *ParsedOpcode, t *thread) error {
	val, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	return t.dstack.RollN(val.Int32())
}

// opcodeRot rotates the top 3 items on the data stack to the left.
//
// Stack transformation: [... x1 x2 x3] -> [... x2 x3 x1]
func opcodeRot(op *ParsedOpcode, t *thread) error {
	return t.dstack.RotN(1)
}

// opcodeSwap swaps the top two items on the stack.
//
// Stack transformation: [... x1 x2] -> [... x2 x1]
func opcodeSwap(op *ParsedOpcode, t *thread) error {
	return t.dstack.SwapN(1)
}

// opcodeTuck inserts a duplicate of the top item of the data stack before the
// second-to-top item.
//
// Stack transformation: [... x1 x2] -> [... x2 x1 x2]
func opcodeTuck(op *ParsedOpcode, t *thread) error {
	return t.dstack.Tuck()
}

// opcodeCat concatenates two byte sequences. The result must
// not be larger than MaxScriptElementSize.
//
// Stack transformation: {Ox11} {0x22, 0x33} script.OpCAT -> 0x112233
func opcodeCat(op *ParsedOpcode, t *thread) error {
	b, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	c := bytes.Join([][]byte{a, b}, nil)
	if len(c) > t.cfg.MaxScriptElementSize() {
		return errs.NewError(errs.ErrElementTooBig,
			"concatenated size %d exceeds max allowed size %d", len(c), t.cfg.MaxScriptElementSize())
	}

	t.dstack.PushByteArray(c)
	return nil
}

// opcodeSplit splits the operand at the given position.
// This operation is the exact inverse of script.OpCAT
//
// Stack transformation: x n script.OpSPLIT -> x1 x2
func opcodeSplit(op *ParsedOpcode, t *thread) error {
	n, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	c, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	if n.Int32() > int32(len(c)) {
		return errs.NewError(errs.ErrNumberTooBig, "n is larger than length of array")
	}
	if n.LessThanInt(0) {
		return errs.NewError(errs.ErrNumberTooSmall, "n is negative")
	}

	a := c[:n.Int()]
	b := c[n.Int():]
	t.dstack.PushByteArray(a)
	t.dstack.PushByteArray(b)

	return nil
}

// opcodeNum2Bin converts the numeric value into a byte sequence of a
// certain size, taking account of the sign bit. The byte sequence
// produced uses the little-endian encoding.
//
// Stack transformation: a b script.OpNUM2BIN -> x
func opcodeNum2bin(op *ParsedOpcode, t *thread) error {
	n, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	if n.GreaterThanInt(int64(t.cfg.MaxScriptElementSize())) {
		return errs.NewError(errs.ErrNumberTooBig, "n is larger than the max of %d", t.cfg.MaxScriptElementSize())
	}

	// encode a as a script num so that we we take the bytes it
	// will be minimally encoded.
	sn, err := MakeScriptNumber(a, len(a), false, t.afterGenesis)
	if err != nil {
		return err
	}

	b := sn.Bytes()
	if n.LessThanInt(int64(len(b))) {
		return errs.NewError(errs.ErrNumberTooSmall, "cannot fit it into n sized array")
	}
	if n.EqualInt(int64(len(b))) {
		t.dstack.PushByteArray(b)
		return nil
	}

	signbit := byte(0x00)
	if len(b) > 0 {
		signbit = b[len(b)-1] & 0x80
		b[len(b)-1] &= 0x7f
	}

	for n.GreaterThanInt(int64(len(b) + 1)) {
		b = append(b, 0x00)
	}

	b = append(b, signbit)

	t.dstack.PushByteArray(b)
	return nil
}

// opcodeBin2num converts the byte sequence into a numeric value,
// including minimal encoding. The byte sequence must encode the
// value in little-endian encoding.
//
// Stack transformation: a script.OpBIN2NUM -> x
func opcodeBin2num(op *ParsedOpcode, t *thread) error {
	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	b := make([]byte, len(a))
	// Copy the bytes so that we don't corrupt the original stack value
	copy(b, a)
	b = MinimallyEncode(b)
	if len(b) > t.cfg.MaxScriptNumberLength() {
		return errs.NewError(errs.ErrNumberTooBig, "script numbers are limited to %d bytes", t.cfg.MaxScriptNumberLength())
	}

	t.dstack.PushByteArray(b)
	return nil
}

// opcodeSize pushes the size of the top item of the data stack onto the data
// stack.
//
// Stack transformation: [... x1] -> [... x1 len(x1)]
func opcodeSize(op *ParsedOpcode, t *thread) error {
	so, err := t.dstack.PeekByteArray(0)
	if err != nil {
		return err
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(int64(len(so))),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeInvert flips all of the top stack item's bits
//
// Stack transformation: a -> ~a
func opcodeInvert(op *ParsedOpcode, t *thread) error {
	ba, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	// We need to invert without modifying the bytes in place.
	// If we modify in place then these changes are reflected elsewhere in the stack.
	baInverted := make([]byte, len(ba))
	for i := range ba {
		baInverted[i] = ba[i] ^ 0xFF
	}

	t.dstack.PushByteArray(baInverted)

	return nil
}

// opcodeAnd executes a boolean and between each bit in the operands
//
// Stack transformation: x1 x2 script.OpAND -> out
func opcodeAnd(op *ParsedOpcode, t *thread) error { //nolint:dupl // to keep functionality with function signature
	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	b, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	if len(a) != len(b) {
		return errs.NewError(errs.ErrInvalidInputLength, "byte arrays are not the same length")
	}

	c := make([]byte, len(a))
	for i := range a {
		c[i] = a[i] & b[i]
	}

	t.dstack.PushByteArray(c)
	return nil
}

// opcodeOr executes a boolean or between each bit in the operands
//
// Stack transformation: x1 x2 script.OpOR -> out
func opcodeOr(op *ParsedOpcode, t *thread) error { //nolint:dupl // to keep functionality with function signature
	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	b, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	if len(a) != len(b) {
		return errs.NewError(errs.ErrInvalidInputLength, "byte arrays are not the same length")
	}

	c := make([]byte, len(a))
	for i := range a {
		c[i] = a[i] | b[i]
	}

	t.dstack.PushByteArray(c)
	return nil
}

// opcodeXor executes a boolean xor between each bit in the operands
//
// Stack transformation: x1 x2 script.OpXOR -> out
func opcodeXor(op *ParsedOpcode, t *thread) error { //nolint:dupl // to keep functionality with function signature
	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	b, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	if len(a) != len(b) {
		return errs.NewError(errs.ErrInvalidInputLength, "byte arrays are not the same length")
	}

	c := make([]byte, len(a))
	for i := range a {
		c[i] = a[i] ^ b[i]
	}

	t.dstack.PushByteArray(c)
	return nil
}

// opcodeEqual removes the top 2 items of the data stack, compares them as raw
// bytes, and pushes the result, encoded as a boolean, back to the stack.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeEqual(op *ParsedOpcode, t *thread) error {
	a, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	b, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	t.dstack.PushBool(bytes.Equal(a, b))
	return nil
}

// opcodeEqualVerify is a combination of opcodeEqual and opcodeVerify.
// Specifically, it removes the top 2 items of the data stack, compares them,
// and pushes the result, encoded as a boolean, back to the stack.  Then, it
// examines the top item on the data stack as a boolean value and verifies it
// evaluates to true.  An error is returned if it does not.
//
// Stack transformation: [... x1 x2] -> [... bool] -> [...]
func opcodeEqualVerify(op *ParsedOpcode, t *thread) error {
	if err := opcodeEqual(op, t); err != nil {
		return err
	}

	return abstractVerify(op, t, errs.ErrEqualVerify)
}

// opcode1Add treats the top item on the data stack as an integer and replaces
// it with its incremented value (plus 1).
//
// Stack transformation: [... x1 x2] -> [... x1 x2+1]
func opcode1Add(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(m.Incr())
	return nil
}

// opcode1Sub treats the top item on the data stack as an integer and replaces
// it with its decremented value (minus 1).
//
// Stack transformation: [... x1 x2] -> [... x1 x2-1]
func opcode1Sub(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(m.Decr())
	return nil
}

// opcodeNegate treats the top item on the data stack as an integer and replaces
// it with its negation.
//
// Stack transformation: [... x1 x2] -> [... x1 -x2]
func opcodeNegate(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(m.Neg())
	return nil
}

// opcodeAbs treats the top item on the data stack as an integer and replaces it
// it with its absolute value.
//
// Stack transformation: [... x1 x2] -> [... x1 abs(x2)]
func opcodeAbs(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(m.Abs())
	return nil
}

// opcodeNot treats the top item on the data stack as an integer and replaces
// it with its "inverted" value (0 becomes 1, non-zero becomes 0).
//
// NOTE: While it would probably make more sense to treat the top item as a
// boolean, and push the opposite, which is really what the intention of this
// opcode is, it is extremely important that is not done because integers are
// interpreted differently than booleans and the consensus rules for this opcode
// dictate the item is interpreted as an integer.
//
// Stack transformation (x2==0): [... x1 0] -> [... x1 1]
// Stack transformation (x2!=0): [... x1 1] -> [... x1 0]
// Stack transformation (x2!=0): [... x1 17] -> [... x1 0]
func opcodeNot(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if m.IsZero() {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcode0NotEqual treats the top item on the data stack as an integer and
// replaces it with either a 0 if it is zero, or a 1 if it is not zero.
//
// Stack transformation (x2==0): [... x1 0] -> [... x1 0]
// Stack transformation (x2!=0): [... x1 1] -> [... x1 1]
// Stack transformation (x2!=0): [... x1 17] -> [... x1 1]
func opcode0NotEqual(op *ParsedOpcode, t *thread) error {
	m, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	if !m.IsZero() {
		m.Set(1)
	}

	t.dstack.PushInt(m)
	return nil
}

// opcodeAdd treats the top two items on the data stack as integers and replaces
// them with their sum.
//
// Stack transformation: [... x1 x2] -> [... x1+x2]
func opcodeAdd(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(v0.Add(v1))
	return nil
}

// opcodeSub treats the top two items on the data stack as integers and replaces
// them with the result of subtracting the top entry from the second-to-top
// entry.
//
// Stack transformation: [... x1 x2] -> [... x1-x2]
func opcodeSub(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(v1.Sub(v0))
	return nil
}

// opcodeMul treats the top two items on the data stack as integers and replaces
// them with the result of subtracting the top entry from the second-to-top
// entry.
func opcodeMul(op *ParsedOpcode, t *thread) error {
	n1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	n2, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	t.dstack.PushInt(n1.Mul(n2))
	return nil
}

// opcodeDiv return the integer quotient of a and b. If the result
// would be a non-integer it is rounded towards zero.
//
// Stack transformation: a b script.OpDIV -> out
func opcodeDiv(op *ParsedOpcode, t *thread) error {
	b, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	a, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	if b.IsZero() {
		return errs.NewError(errs.ErrDivideByZero, "divide by zero")
	}

	t.dstack.PushInt(a.Div(b))
	return nil
}

// opcodeMod returns the remainder after dividing a by b. The output will
// be represented using the least number of bytes required.
//
// Stack transformation: a b script.OpMOD -> out
func opcodeMod(op *ParsedOpcode, t *thread) error {
	b, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	a, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	if b.IsZero() {
		return errs.NewError(errs.ErrDivideByZero, "mod by zero")
	}

	t.dstack.PushInt(a.Mod(b))
	return nil
}

func opcodeLShift(op *ParsedOpcode, t *thread) error {
	num, err := t.dstack.PopInt()
	if err != nil {
		return err
	}
	n := num.Int()

	if n < 0 {
		return errs.NewError(errs.ErrNumberTooSmall, "n less than 0")
	}

	x, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	bitShift := n % 8
	byteShift := n / 8
	mask := []byte{0xFF, 0x7F, 0x3F, 0x1F, 0x0F, 0x07, 0x03, 0x01}[bitShift]
	overflowMask := ^mask

	result := make([]byte, len(x))
	for idx := len(x); idx > 0; idx-- {
		i := idx - 1
		if byteShift <= i {
			k := i - byteShift
			val := x[i] & mask
			val <<= bitShift
			result[k] |= val

			if k >= 1 {
				carryval := x[i] & overflowMask
				carryval >>= 8 - bitShift
				result[k-1] |= carryval
			}
		}
	}

	t.dstack.PushByteArray(result)
	return nil
}

func opcodeRShift(op *ParsedOpcode, t *thread) error {
	num, err := t.dstack.PopInt()
	if err != nil {
		return err
	}
	n := num.Int()

	if n < 0 {
		return errs.NewError(errs.ErrNumberTooSmall, "n less than 0")
	}

	x, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	byteShift := n / 8
	bitShift := n % 8
	mask := []byte{0xFF, 0xFE, 0xFC, 0xF8, 0xF0, 0xE0, 0xC0, 0x80}[bitShift]
	overflowMask := ^mask
	result := make([]byte, len(x))
	for i, b := range x {
		k := i + byteShift
		if k < len(x) {
			val := b & mask
			val >>= bitShift
			result[k] |= val
		}

		if k+1 < len(x) {
			carryval := b & overflowMask
			carryval <<= 8 - bitShift
			result[k+1] |= carryval
		}
	}

	t.dstack.PushByteArray(result)
	return nil
}

// opcodeBoolAnd treats the top two items on the data stack as integers.  When
// both of them are not zero, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==0, x2==0): [... 0 0] -> [... 0]
// Stack transformation (x1!=0, x2==0): [... 5 0] -> [... 0]
// Stack transformation (x1==0, x2!=0): [... 0 7] -> [... 0]
// Stack transformation (x1!=0, x2!=0): [... 4 8] -> [... 1]
func opcodeBoolAnd(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if !v0.IsZero() && !v1.IsZero() {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeBoolOr treats the top two items on the data stack as integers.  When
// either of them are not zero, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==0, x2==0): [... 0 0] -> [... 0]
// Stack transformation (x1!=0, x2==0): [... 5 0] -> [... 1]
// Stack transformation (x1==0, x2!=0): [... 0 7] -> [... 1]
// Stack transformation (x1!=0, x2!=0): [... 4 8] -> [... 1]
func opcodeBoolOr(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if !v0.IsZero() || !v1.IsZero() {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeNumEqual treats the top two items on the data stack as integers.  When
// they are equal, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==x2): [... 5 5] -> [... 1]
// Stack transformation (x1!=x2): [... 5 7] -> [... 0]
func opcodeNumEqual(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if v0.Equal(v1) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeNumEqualVerify is a combination of opcodeNumEqual and opcodeVerify.
//
// Specifically, treats the top two items on the data stack as integers.  When
// they are equal, they are replaced with a 1, otherwise a 0.  Then, it examines
// the top item on the data stack as a boolean value and verifies it evaluates
// to true.  An error is returned if it does not.
//
// Stack transformation: [... x1 x2] -> [... bool] -> [...]
func opcodeNumEqualVerify(op *ParsedOpcode, t *thread) error {
	if err := opcodeNumEqual(op, t); err != nil {
		return err
	}

	return abstractVerify(op, t, errs.ErrNumEqualVerify)
}

// opcodeNumNotEqual treats the top two items on the data stack as integers.
// When they are NOT equal, they are replaced with a 1, otherwise a 0.
//
// Stack transformation (x1==x2): [... 5 5] -> [... 0]
// Stack transformation (x1!=x2): [... 5 7] -> [... 1]
func opcodeNumNotEqual(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if !v0.Equal(v1) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeLessThan treats the top two items on the data stack as integers.  When
// the second-to-top item is less than the top item, they are replaced with a 1,
// otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeLessThan(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if v1.LessThan(v0) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeGreaterThan treats the top two items on the data stack as integers.
// When the second-to-top item is greater than the top item, they are replaced
// with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeGreaterThan(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if v1.GreaterThan(v0) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeLessThanOrEqual treats the top two items on the data stack as integers.
// When the second-to-top item is less than or equal to the top item, they are
// replaced with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeLessThanOrEqual(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if v1.LessThanOrEqual(v0) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeGreaterThanOrEqual treats the top two items on the data stack as
// integers.  When the second-to-top item is greater than or equal to the top
// item, they are replaced with a 1, otherwise a 0.
//
// Stack transformation: [... x1 x2] -> [... bool]
func opcodeGreaterThanOrEqual(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if v1.GreaterThanOrEqual(v0) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// opcodeMin treats the top two items on the data stack as integers and replaces
// them with the minimum of the two.
//
// Stack transformation: [... x1 x2] -> [... min(x1, x2)]
func opcodeMin(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	n := v0
	if v1.LessThan(v0) {
		n = v1
	}

	t.dstack.PushInt(n)
	return nil
}

// opcodeMax treats the top two items on the data stack as integers and replaces
// them with the maximum of the two.
//
// Stack transformation: [... x1 x2] -> [... max(x1, x2)]
func opcodeMax(op *ParsedOpcode, t *thread) error {
	v0, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	v1, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	n := v0
	if v1.GreaterThan(v0) {
		n = v1
	}

	t.dstack.PushInt(n)
	return nil
}

// opcodeWithin treats the top 3 items on the data stack as integers.  When the
// value to test is within the specified range (left inclusive), they are
// replaced with a 1, otherwise a 0.
//
// The top item is the max value, the second-top-item is the minimum value, and
// the third-to-top item is the value to test.
//
// Stack transformation: [... x1 min max] -> [... bool]
func opcodeWithin(op *ParsedOpcode, t *thread) error {
	maxVal, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	minVal, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	x, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	var n int64
	if minVal.LessThanOrEqual(x) && x.LessThan(maxVal) {
		n = 1
	}

	t.dstack.PushInt(&ScriptNumber{
		Val:          big.NewInt(n),
		AfterGenesis: t.afterGenesis,
	})
	return nil
}

// calcHash calculates the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

// opcodeRipemd160 treats the top item of the data stack as raw bytes and
// replaces it with ripemd160(data).
//
// Stack transformation: [... x1] -> [... ripemd160(x1)]
func opcodeRipemd160(op *ParsedOpcode, t *thread) error {
	buf, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	t.dstack.PushByteArray(calcHash(buf, ripemd160.New())) //nolint:gosec // required
	return nil
}

// opcodeSha1 treats the top item of the data stack as raw bytes and replaces it
// with sha1(data).
//
// Stack transformation: [... x1] -> [... sha1(x1)]
func opcodeSha1(op *ParsedOpcode, t *thread) error {
	buf, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha1.Sum(buf) //nolint:gosec // operation is for sha1
	t.dstack.PushByteArray(hash[:])
	return nil
}

// opcodeSha256 treats the top item of the data stack as raw bytes and replaces
// it with sha256(data).
//
// Stack transformation: [... x1] -> [... sha256(x1)]
func opcodeSha256(op *ParsedOpcode, t *thread) error {
	buf, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(buf)
	t.dstack.PushByteArray(hash[:])
	return nil
}

// opcodeHash160 treats the top item of the data stack as raw bytes and replaces
// it with ripemd160(sha256(data)).
//
// Stack transformation: [... x1] -> [... ripemd160(sha256(x1))]
func opcodeHash160(op *ParsedOpcode, t *thread) error {
	buf, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	hash := sha256.Sum256(buf)
	t.dstack.PushByteArray(calcHash(hash[:], ripemd160.New())) //nolint:gosec // required
	return nil
}

// opcodeHash256 treats the top item of the data stack as raw bytes and replaces
// it with sha256(sha256(data)).
//
// Stack transformation: [... x1] -> [... sha256(sha256(x1))]
func opcodeHash256(op *ParsedOpcode, t *thread) error {
	buf, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	t.dstack.PushByteArray(crypto.Sha256d(buf))
	return nil
}

// opcodeCodeSeparator stores the current script offset as the most recently
// seen script.OpCODESEPARATOR which is used during signature checking.
//
// This opcode does not change the contents of the data stack.
func opcodeCodeSeparator(op *ParsedOpcode, t *thread) error {
	t.lastCodeSep = t.scriptOff
	return nil
}

// opcodeCheckSig treats the top 2 items on the stack as a public key and a
// signature and replaces them with a bool which indicates if the signature was
// successfully verified.
//
// The process of verifying a signature requires calculating a signature hash in
// the same way the transaction signer did.  It involves hashing portions of the
// transaction based on the hash type byte (which is the final byte of the
// signature) and the portion of the script starting from the most recent
// script.OpCODESEPARATOR (or the beginning of the script if there are none) to the
// end of the script (with any other script.OpCODESEPARATORs removed).  Once this
// "script hash" is calculated, the signature is checked using standard
// cryptographic methods against the provided public key.
//
// Stack transformation: [... signature pubkey] -> [... bool]
func opcodeCheckSig(op *ParsedOpcode, t *thread) error {
	pkBytes, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	fullSigBytes, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	// The signature actually needs needs to be longer than this, but at
	// least 1 byte is needed for the hash type below.  The full length is
	// checked depending on the script flags and upon parsing the signature.
	if len(fullSigBytes) < 1 {
		t.dstack.PushBool(false)
		return nil
	}

	// Trim off hashtype from the signature string and check if the
	// signature and pubkey conform to the strict encoding requirements
	// depending on the flags.
	//
	// NOTE: When the strict encoding flags are set, any errors in the
	// signature or public encoding here result in an immediate script error
	// (and thus no result bool is pushed to the data stack).  This differs
	// from the logic below where any errors in parsing the signature is
	// treated as the signature failure resulting in false being pushed to
	// the data stack.  This is required because the more general script
	// validation consensus rules do not have the new strict encoding
	// requirements enabled by the flags.
	shf := sighash.Flag(fullSigBytes[len(fullSigBytes)-1])
	sigBytes := fullSigBytes[:len(fullSigBytes)-1]
	if err = t.checkHashTypeEncoding(shf); err != nil {
		return err
	}
	if err = t.checkSignatureEncoding(sigBytes); err != nil {
		return err
	}
	if err = t.checkPubKeyEncoding(pkBytes); err != nil {
		return err
	}

	// Get script starting from the most recent script.OpCODESEPARATOR.
	subScript := t.subScript()

	// Generate the signature hash based on the signature hash type.
	var hash []byte

	// Remove the signature since there is no way for a signature
	// to sign itself.
	if !t.hasFlag(scriptflag.EnableSighashForkID) || !shf.Has(sighash.ForkID) {
		subScript = subScript.removeOpcodeByData(fullSigBytes)
		subScript = subScript.removeOpcode(script.OpCODESEPARATOR)
	}

	up, err := t.scriptParser.Unparse(subScript)
	if err != nil {
		return err
	}

	txCopy := t.tx.ShallowClone()
	sourceTxOut := txCopy.Inputs[t.inputIdx].SourceTxOutput()
	sourceTxOut.LockingScript = up

	hash, err = txCopy.CalcInputSignatureHash(uint32(t.inputIdx), shf)
	if err != nil {
		t.dstack.PushBool(false)
		return err
	}

	var sigBytesDer []byte
	// if the signature is in DER format, we can set it here and just use it
	// directly if an externalVerifySignatureFn is set
	if t.hasAny(scriptflag.VerifyStrictEncoding, scriptflag.VerifyDERSignatures) {
		sigBytesDer = sigBytes
	}

	var ok bool
	var signature *ec.Signature

	if externalVerifySignatureFn != nil {
		if sigBytesDer == nil {
			// signature is not in DER format, so we must parse it and set the bytes
			signature, err = ec.ParseSignature(sigBytes)
			if err != nil {
				t.dstack.PushBool(false)
				return nil //nolint:nilerr // only need a false push in this case
			}
			if sigBytesDer, err = signature.ToDER(); err != nil {
				return err
			}
		}
		ok = externalVerifySignatureFn(hash, sigBytesDer, pkBytes)
	} else {
		var pubKey *ec.PublicKey
		pubKey, err = ec.ParsePubKey(pkBytes)
		if err != nil {
			t.dstack.PushBool(false)
			return nil //nolint:nilerr // only need a false push in this case
		}

		if t.hasAny(scriptflag.VerifyStrictEncoding, scriptflag.VerifyDERSignatures) {
			signature, err = ec.ParseDERSignature(sigBytes)
		} else {
			signature, err = ec.ParseSignature(sigBytes)
		}
		if err != nil {
			t.dstack.PushBool(false)
			return nil //nolint:nilerr // only need a false push in this case
		}

		ok = signature.Verify(hash, pubKey)
	}

	if !ok && t.hasFlag(scriptflag.VerifyNullFail) && len(sigBytes) > 0 {
		return errs.NewError(errs.ErrNullFail, "signature not empty on failed checksig")
	}

	t.dstack.PushBool(ok)
	return nil
}

// opcodeCheckSigVerify is a combination of opcodeCheckSig and opcodeVerify.
// The opcodeCheckSig function is invoked followed by opcodeVerify.  See the
// documentation for each of those opcodes for more details.
//
// Stack transformation: signature pubkey] -> [... bool] -> [...]
func opcodeCheckSigVerify(op *ParsedOpcode, t *thread) error {
	if err := opcodeCheckSig(op, t); err != nil {
		return err
	}

	return abstractVerify(op, t, errs.ErrCheckSigVerify)
}

// parsedSigInfo houses a raw signature along with its parsed form and a flag
// for whether or not it has already been parsed.  It is used to prevent parsing
// the same signature multiple times when verifying a multisig.
type parsedSigInfo struct {
	signature       []byte
	parsedSignature *ec.Signature
	parsed          bool
}

// opcodeCheckMultiSig treats the top item on the stack as an integer number of
// public keys, followed by that many entries as raw data representing the public
// keys, followed by the integer number of signatures, followed by that many
// entries as raw data representing the signatures.
//
// Due to a bug in the original Satoshi client implementation, an additional
// dummy argument is also required by the consensus rules, although it is not
// used.  The dummy value SHOULD be an script.Op0, although that is not required by
// the consensus rules.  When the ScriptStrictMultiSig flag is set, it must be
// script.Op0.
//
// All of the aforementioned stack items are replaced with a bool which
// indicates if the requisite number of signatures were successfully verified.
//
// See the opcodeCheckSigVerify documentation for more details about the process
// for verifying each signature.
//
// Stack transformation:
// [... dummy [sig ...] numsigs [pubkey ...] numpubkeys] -> [... bool]
func opcodeCheckMultiSig(op *ParsedOpcode, t *thread) error {
	numKeys, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	numPubKeys := numKeys.Int()
	if numPubKeys < 0 {
		return errs.NewError(errs.ErrInvalidPubKeyCount, "number of pubkeys %d is negative", numPubKeys)
	}
	if numPubKeys > t.cfg.MaxPubKeysPerMultiSig() {
		return errs.NewError(
			errs.ErrInvalidPubKeyCount,
			"too many pubkeys: %d > %d",
			numPubKeys, t.cfg.MaxPubKeysPerMultiSig(),
		)
	}
	t.numOps += numPubKeys
	if t.numOps > t.cfg.MaxOps() {
		return errs.NewError(errs.ErrTooManyOperations, "exceeded max operation limit of %d", t.cfg.MaxOps())
	}

	pubKeys := make([][]byte, 0, numPubKeys)
	for i := 0; i < numPubKeys; i++ {
		pubKey, err := t.dstack.PopByteArray()
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, pubKey)
	}

	numSigs, err := t.dstack.PopInt()
	if err != nil {
		return err
	}

	numSignatures := numSigs.Int()
	if numSignatures < 0 {
		return errs.NewError(errs.ErrInvalidSignatureCount, "number of signatures %d is negative", numSignatures)
	}
	if numSignatures > numPubKeys {
		return errs.NewError(
			errs.ErrInvalidSignatureCount,
			"more signatures than pubkeys: %d > %d",
			numSignatures, numPubKeys,
		)
	}

	signatures := make([]*parsedSigInfo, 0, numSignatures)
	for i := 0; i < numSignatures; i++ {
		signature, err := t.dstack.PopByteArray()
		if err != nil {
			return err
		}
		sigInfo := &parsedSigInfo{signature: signature}
		signatures = append(signatures, sigInfo)
	}

	// A bug in the original Satoshi client implementation means one more
	// stack value than should be used must be popped.  Unfortunately, this
	// buggy behavior is now part of the consensus and a hard fork would be
	// required to fix it.
	dummy, err := t.dstack.PopByteArray()
	if err != nil {
		return err
	}

	// Since the dummy argument is otherwise not checked, it could be any
	// value which unfortunately provides a source of malleability.  Thus,
	// there is a script flag to force an error when the value is NOT 0.
	if t.hasFlag(scriptflag.StrictMultiSig) && len(dummy) != 0 {
		return errs.NewError(errs.ErrSigNullDummy, "multisig dummy argument has length %d instead of 0", len(dummy))
	}

	// Get script starting from the most recent script.OpCODESEPARATOR.
	scr := t.subScript()

	for _, sigInfo := range signatures {
		scr = scr.removeOpcodeByData(sigInfo.signature)
		scr = scr.removeOpcode(script.OpCODESEPARATOR)
	}

	success := true
	numPubKeys++
	pubKeyIdx := -1
	signatureIdx := 0
	for numSignatures > 0 {
		// When there are more signatures than public keys remaining,
		// there is no way to succeed since too many signatures are
		// invalid, so exit early.
		pubKeyIdx++
		numPubKeys--
		if numSignatures > numPubKeys {
			success = false
			break
		}

		sigInfo := signatures[signatureIdx]
		pubKey := pubKeys[pubKeyIdx]

		// The order of the signature and public key evaluation is
		// important here since it can be distinguished by an
		// script.OpCHECKMULTISIG NOT when the strict encoding flag is set.

		rawSig := sigInfo.signature
		if len(rawSig) == 0 {
			// Skip to the next pubkey if signature is empty.
			continue
		}

		// Split the signature into hash type and signature components.
		shf := sighash.Flag(rawSig[len(rawSig)-1])
		signature := rawSig[:len(rawSig)-1]

		// Only parse and check the signature encoding once.
		var parsedSig *ec.Signature
		if !sigInfo.parsed {
			if err := t.checkHashTypeEncoding(shf); err != nil {
				return err
			}
			if err := t.checkSignatureEncoding(signature); err != nil {
				return err
			}

			// Parse the signature.
			var err error
			if t.hasAny(scriptflag.VerifyStrictEncoding, scriptflag.VerifyDERSignatures) {
				parsedSig, err = ec.ParseDERSignature(signature)
			} else {
				parsedSig, err = ec.ParseSignature(signature)
			}
			sigInfo.parsed = true
			if err != nil {
				continue
			}
			sigInfo.parsedSignature = parsedSig
		} else {
			// Skip to the next pubkey if the signature is invalid.
			if sigInfo.parsedSignature == nil {
				continue
			}

			// Use the already parsed signature.
			parsedSig = sigInfo.parsedSignature
		}

		if err := t.checkPubKeyEncoding(pubKey); err != nil {
			return err
		}

		// Parse the pubkey.
		parsedPubKey, err := ec.ParsePubKey(pubKey)
		if err != nil {
			continue
		}

		up, err := t.scriptParser.Unparse(scr)
		if err != nil {
			t.dstack.PushBool(false)
			return nil //nolint:nilerr // only need a false push in this case
		}

		// Generate the signature hash based on the signature hash type.
		txCopy := t.tx.ShallowClone()
		input := txCopy.Inputs[t.inputIdx]
		sourceOut := input.SourceTxOutput()
		if sourceOut != nil {
			sourceOut.LockingScript = up
		}

		signatureHash, err := txCopy.CalcInputSignatureHash(uint32(t.inputIdx), shf)
		if err != nil {
			t.dstack.PushBool(false)
			return nil //nolint:nilerr // only need a false push in this case
		}

		if ok := parsedSig.Verify(signatureHash, parsedPubKey); ok {
			// PubKey verified, move on to the next signature.
			signatureIdx++
			numSignatures--
		}
	}

	if !success && t.hasFlag(scriptflag.VerifyNullFail) {
		for _, sig := range signatures {
			if len(sig.signature) > 0 {
				return errs.NewError(errs.ErrNullFail, "not all signatures empty on failed checkmultisig")
			}
		}
	}

	t.dstack.PushBool(success)
	return nil
}

// opcodeCheckMultiSigVerify is a combination of opcodeCheckMultiSig and
// opcodeVerify.  The opcodeCheckMultiSig is invoked followed by opcodeVerify.
// See the documentation for each of those opcodes for more details.
//
// Stack transformation:
// [... dummy [sig ...] numsigs [pubkey ...] numpubkeys] -> [... bool] -> [...]
func opcodeCheckMultiSigVerify(op *ParsedOpcode, t *thread) error {
	if err := opcodeCheckMultiSig(op, t); err != nil {
		return err
	}

	return abstractVerify(op, t, errs.ErrCheckMultiSigVerify)
}

func success() errs.Error {
	return errs.NewError(errs.ErrOK, "success")
}
