package compat_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	compat "github.com/bsv-blockchain/go-sdk/v2/compat/bip32"
	ec "github.com/bsv-blockchain/go-sdk/v2/primitives/ec"
	script "github.com/bsv-blockchain/go-sdk/v2/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateHDKey will test the method GenerateHDKey()
func TestGenerateHDKey(t *testing.T) {

	t.Parallel()

	var tests = []struct {
		inputSeed     uint8
		expectedNil   bool
		expectedError bool
	}{
		{0, false, false},
		{1, true, true},
		{15, true, true},
		{65, true, true},
		{compat.RecommendedSeedLength, false, false},
		{compat.SecureSeedLength, false, false},
	}

	for _, test := range tests {
		if hdKey, err := compat.GenerateHDKey(test.inputSeed); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%d] inputted and error not expected but got: %s", t.Name(), test.inputSeed, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%d] inputted and error was expected", t.Name(), test.inputSeed)
		} else if hdKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%d] inputted and was nil but not expected", t.Name(), test.inputSeed)
		} else if hdKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.inputSeed)
		}
	}
}

// ExampleGenerateHDKey example using GenerateHDKey()
func ExampleGenerateHDKey() {
	hdKey, err := compat.GenerateHDKey(compat.SecureSeedLength)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	// Cannot show the private/public key since they change each time
	fmt.Printf("created HD key successfully! (length: %d)", len(hdKey.String()))

	// Output:created HD key successfully! (length: 111)
}

// BenchmarkGenerateHDKey benchmarks the method GenerateHDKey()
func BenchmarkGenerateHDKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = compat.GenerateHDKey(compat.RecommendedSeedLength)
	}
}

// BenchmarkGenerateHDKeySecure benchmarks the method GenerateHDKey()
func BenchmarkGenerateHDKeySecure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = compat.GenerateHDKey(compat.SecureSeedLength)
	}
}

// TestGenerateHDKeyPair will test the method GenerateHDKeyPair()
func TestGenerateHDKeyPair(t *testing.T) {

	t.Parallel()

	var tests = []struct {
		inputSeed     uint8
		expectedError bool
	}{
		{0, false},
		{1, true},
		{15, true},
		{65, true},
		{compat.RecommendedSeedLength, false},
		{compat.SecureSeedLength, false},
	}

	for _, test := range tests {
		if privateKey, publicKey, err := compat.GenerateHDKeyPair(test.inputSeed); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%d] inputted and error not expected but got: %s", t.Name(), test.inputSeed, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%d] inputted and error was expected", t.Name(), test.inputSeed)
		} else if err == nil && len(privateKey) == 0 {
			t.Fatalf("%s Failed: [%d] inputted and private key was empty", t.Name(), test.inputSeed)
		} else if err == nil && len(publicKey) == 0 {
			t.Fatalf("%s Failed: [%d] inputted and pubic key was empty", t.Name(), test.inputSeed)
		}
	}
}

// ExampleGenerateHDKeyPair example using GenerateHDKeyPair()
func ExampleGenerateHDKeyPair() {
	xPrivateKey, xPublicKey, err := compat.GenerateHDKeyPair(compat.SecureSeedLength)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	// Cannot show the private/public key since they change each time
	fmt.Printf("created HD key successfully! (xPrivateKey length: %d) (xPublicKey length: %d)", len(xPrivateKey), len(xPublicKey))

	// Output:created HD key successfully! (xPrivateKey length: 111) (xPublicKey length: 111)
}

// BenchmarkGenerateHDKeyPair benchmarks the method GenerateHDKeyPair()
func BenchmarkGenerateHDKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = compat.GenerateHDKeyPair(compat.RecommendedSeedLength)
	}
}

// BenchmarkGenerateHDKeyPairSecure benchmarks the method GenerateHDKeyPair()
func BenchmarkGenerateHDKeyPairSecure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = compat.GenerateHDKeyPair(compat.SecureSeedLength)
	}
}

// TestGetPrivateKeyByPath will test the method GetPrivateKeyByPath()
func TestGetPrivateKeyByPath(t *testing.T) {

	t.Parallel()

	// Generate a valid key
	validKey, err := compat.GenerateHDKey(compat.RecommendedSeedLength)
	if err != nil {
		t.Fatalf("error occurred: %s", err.Error())
	}

	var tests = []struct {
		inputHDKey    *compat.ExtendedKey
		inputChain    uint32
		inputNum      uint32
		expectedNil   bool
		expectedError bool
	}{
		// {nil, 0, 0, true, true},
		{validKey, 0, 0, false, false},
		{validKey, 10, 10, false, false},
		{validKey, 100, 100, false, false},
		{validKey, 2 ^ 31 + 1, 2 ^ 32 - 1, false, false},
		{validKey, 1 << 8, 1 << 8, false, false},
		{validKey, 1 << 9, 1 << 9, false, false},
		{validKey, 1 << 10, 1 << 10, false, false},
		{validKey, 1 << 11, 1 << 11, false, false},
		{validKey, 1 << 12, 1 << 12, false, false},
		{validKey, 1 << 16, 1 << 16, false, false},
		{validKey, 1<<32 - 1, 1<<32 - 1, false, false},
	}

	var privateKey *ec.PrivateKey
	for _, test := range tests {
		if privateKey, err = compat.GetPrivateKeyByPath(test.inputHDKey, test.inputChain, test.inputNum); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and error not expected but got: %s", t.Name(), test.inputHDKey, test.inputChain, test.inputNum, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and error was expected", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if privateKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and was nil but not expected", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if privateKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if privateKey != nil && len(hex.EncodeToString(privateKey.Serialize())) == 0 {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and should not be empty", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		}
	}
}

// TestGetPrivateKeyByPathPanic tests for nil case in GetPrivateKeyByPath()
func TestGetPrivateKeyByPathPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetPrivateKeyByPath(nil, 0, 1)
		assert.Error(t, err)
	})
}

// ExampleGetPrivateKeyByPath example using GetPrivateKeyByPath()
func ExampleGetPrivateKeyByPath() {

	hdKey, err := compat.GenerateHDKey(compat.SecureSeedLength)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	// Get a private key at the path
	var privateKey *ec.PrivateKey
	privateKey, err = compat.GetPrivateKeyByPath(hdKey, 0, 1)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	fmt.Printf("private key (%d) found at path %d/%d", len(privateKey.Serialize()), 0, 1)
	// Output:private key (32) found at path 0/1
}

// BenchmarkGetPrivateKeyByPath benchmarks the method GetPrivateKeyByPath()
func BenchmarkGetPrivateKeyByPath(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetPrivateKeyByPath(hdKey, 0, 1)
	}
}

// TestGetHDKeyByPath will test the method GetHDKeyByPath()
func TestGetHDKeyByPath(t *testing.T) {

	t.Parallel()

	// Generate a valid key
	validKey, err := compat.GenerateHDKey(compat.RecommendedSeedLength)
	require.NoError(t, err)
	assert.NotNil(t, validKey)

	var tests = []struct {
		inputHDKey    *compat.ExtendedKey
		inputChain    uint32
		inputNum      uint32
		expectedNil   bool
		expectedError bool
	}{
		{validKey, 0, 0, false, false},
		{validKey, 10, 10, false, false},
		{validKey, 100, 100, false, false},
		{validKey, 2 ^ 31 + 1, 2 ^ 32 - 1, false, false},
		{validKey, 1 << 8, 1 << 8, false, false},
		{validKey, 1 << 9, 1 << 9, false, false},
		{validKey, 1 << 10, 1 << 10, false, false},
		{validKey, 1 << 11, 1 << 11, false, false},
		{validKey, 1 << 12, 1 << 12, false, false},
		{validKey, 1 << 16, 1 << 16, false, false},
		{validKey, 1<<32 - 1, 1<<32 - 1, false, false},
	}

	for _, test := range tests {
		if hdKey, err := compat.GetHDKeyByPath(test.inputHDKey, test.inputChain, test.inputNum); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and error not expected but got: %s", t.Name(), test.inputHDKey, test.inputChain, test.inputNum, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and error was expected", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if hdKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and was nil but not expected", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if hdKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		} else if hdKey != nil && len(hdKey.String()) == 0 {
			t.Fatalf("%s Failed: [%v] [%d] [%d] inputted and should not be empty", t.Name(), test.inputHDKey, test.inputChain, test.inputNum)
		}
	}
}

// TestGetHDKeyByPathPanic tests for nil case in GetHDKeyByPath()
func TestGetHDKeyByPathPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetHDKeyByPath(nil, 0, 1)
		assert.Error(t, err)
	})
}

// ExampleGetHDKeyByPath example using GetHDKeyByPath()
func ExampleGetHDKeyByPath() {

	hdKey, err := compat.GenerateHDKey(compat.SecureSeedLength)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	// Get a child key
	var childKey *compat.ExtendedKey
	childKey, err = compat.GetHDKeyByPath(hdKey, 0, 1)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	fmt.Printf("hd key (%d) found at path %d/%d", len(childKey.String()), 0, 1)
	// Output:hd key (111) found at path 0/1
}

// BenchmarkGetHDKeyByPath benchmarks the method GetHDKeyByPath()
func BenchmarkGetHDKeyByPath(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetHDKeyByPath(hdKey, 0, 1)
	}
}

// TestGetHDKeyChild will test the method GetHDKeyChild()
func TestGetHDKeyChild(t *testing.T) {

	t.Parallel()

	// Generate a valid key
	validKey, err := compat.GenerateHDKey(compat.RecommendedSeedLength)
	require.NoError(t, err)
	assert.NotNil(t, validKey)

	var tests = []struct {
		inputHDKey    *compat.ExtendedKey
		inputNum      uint32
		expectedNil   bool
		expectedError bool
	}{
		// {nil, 0, true, true},
		{validKey, 0, false, false},
		{validKey, 10, false, false},
		{validKey, 100, false, false},
		{validKey, 2 ^ 31 + 1, false, false},
		{validKey, 1 << 8, false, false},
		{validKey, 1 << 9, false, false},
		{validKey, 1 << 10, false, false},
		{validKey, 1 << 11, false, false},
		{validKey, 1 << 12, false, false},
		{validKey, 1 << 16, false, false},
		{validKey, 1<<32 - 1, false, false},
	}

	for _, test := range tests {
		if hdKey, err := compat.GetHDKeyChild(test.inputHDKey, test.inputNum); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error not expected but got: %s", t.Name(), test.inputHDKey, test.inputNum, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error was expected", t.Name(), test.inputHDKey, test.inputNum)
		} else if hdKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was nil but not expected", t.Name(), test.inputHDKey, test.inputNum)
		} else if hdKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.inputHDKey, test.inputNum)
		} else if hdKey != nil && len(hdKey.String()) == 0 {
			t.Fatalf("%s Failed: [%v] [%d] inputted and should not be empty", t.Name(), test.inputHDKey, test.inputNum)
		}
	}
}

// TestGetHDKeyChildPanic tests for nil case in GetHDKeyChild()
func TestGetHDKeyChildPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetHDKeyChild(nil, 1)
		assert.Error(t, err)
	})
}

// ExampleGetHDKeyChild example using GetHDKeyChild()
func ExampleGetHDKeyChild() {

	hdKey, err := compat.GenerateHDKey(compat.SecureSeedLength)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	// Get a child key
	var childKey *compat.ExtendedKey
	childKey, err = compat.GetHDKeyChild(hdKey, 0)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	fmt.Printf("hd key (%d) found at path %d", len(childKey.String()), 0)
	// Output:hd key (111) found at path 0
}

// BenchmarkGetHDKeyChild benchmarks the method GetHDKeyChild()
func BenchmarkGetHDKeyChild(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetHDKeyChild(hdKey, 0)
	}
}

// TestGenerateHDKeyFromString will test the method GenerateHDKeyFromString()
func TestGenerateHDKeyFromString(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		input         string
		expectedNil   bool
		expectedError bool
	}{
		{"", true, true},
		{"0", true, true},
		{"1234567", true, true},
		{"xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE", false, false},
		{"xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUv", true, true},
		{"xprv9s21ZrQH143K3XJueaaswvbJ38UX3FhnXkcA7xF8kqeN62qEu116M1XnqaDpSE7SoKp8NxejVJG9dfpuvBC314VZNdB7W1kQN3Viwgkjr8L", false, false},
	}

	for _, test := range tests {
		if hdKey, err := compat.GenerateHDKeyFromString(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%s] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%s] inputted and error was expected", t.Name(), test.input)
		} else if hdKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%s] inputted and was nil but not expected", t.Name(), test.input)
		} else if hdKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%s] inputted and was NOT nil but expected to be nil", t.Name(), test.input)
		} else if hdKey != nil && hdKey.String() != test.input {
			t.Fatalf("%s Failed: [%s] inputted [%s] expected but got: %s", t.Name(), test.input, test.input, hdKey.String())
		}
	}
}

// ExampleGenerateHDKeyFromString example using GenerateHDKeyFromString()
func ExampleGenerateHDKeyFromString() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("hd key generated from: %s", hdKey.String())
	// Output:hd key generated from: xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE
}

// BenchmarkGenerateHDKeyFromString benchmarks the method GenerateHDKeyFromString()
func BenchmarkGenerateHDKeyFromString(b *testing.B) {
	xPriv, _, _ := compat.GenerateHDKeyPair(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GenerateHDKeyFromString(xPriv)
	}
}

// TestGetPrivateKeyFromHDKey will test the method GetPrivateKeyFromHDKey()
func TestGetPrivateKeyFromHDKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input         *compat.ExtendedKey
		expectedKey   string
		expectedNil   bool
		expectedError bool
	}{
		{new(compat.ExtendedKey), "", true, true},
		{validHdKey, "8511f5e1e35ab748e7639aa68666df71857866af13fda1d081d5917948a6cd34", false, false},
	}

	for _, test := range tests {
		if privateKey, err := compat.GetPrivateKeyFromHDKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if privateKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was nil but not expected", t.Name(), test.input)
		} else if privateKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was NOT nil but expected to be nil", t.Name(), test.input)
		} else if privateKey != nil && hex.EncodeToString(privateKey.Serialize()) != test.expectedKey {
			t.Fatalf("%s Failed: [%v] inputted [%s] expected but got: %s", t.Name(), test.input, test.expectedKey, hex.EncodeToString(privateKey.Serialize()))
		}
	}
}

// TestGetPrivateKeyFromHDKeyPanic tests for nil case in GetPrivateKeyFromHDKey()
func TestGetPrivateKeyFromHDKeyPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetPrivateKeyFromHDKey(nil)
		assert.Error(t, err)
	})
}

// ExampleGetPrivateKeyFromHDKey example using GetPrivateKeyFromHDKey()
func ExampleGetPrivateKeyFromHDKey() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var privateKey *ec.PrivateKey
	if privateKey, err = compat.GetPrivateKeyFromHDKey(hdKey); err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("private key: %s", hex.EncodeToString(privateKey.Serialize()))
	// Output:private key: 0ccf07f2cbe10dbe6f6034b7efbf62fc83cac3d44f49d67aa22ac8893d294e7a
}

// BenchmarkGetPrivateKeyFromHDKey benchmarks the method GetPrivateKeyFromHDKey()
func BenchmarkGetPrivateKeyFromHDKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetPrivateKeyFromHDKey(hdKey)
	}
}

// TestGetPrivateKeyStringFromHDKey will test the method GetPrivateKeyStringFromHDKey()
func TestGetPrivateKeyStringFromHDKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input         *compat.ExtendedKey
		expectedKey   string
		expectedError bool
	}{
		{new(compat.ExtendedKey), "", true},
		{validHdKey, "8511f5e1e35ab748e7639aa68666df71857866af13fda1d081d5917948a6cd34", false},
	}

	var privateKey string
	for _, test := range tests {
		if privateKey, err = compat.GetPrivateKeyStringFromHDKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if privateKey != test.expectedKey {
			t.Fatalf("%s Failed: [%v] inputted [%s] expected but got: %s", t.Name(), test.input, test.expectedKey, privateKey)
		}
	}
}

// TestGetPrivateKeyStringFromHDKeyPanic tests for nil case in GetPrivateKeyStringFromHDKey()
func TestGetPrivateKeyStringFromHDKeyPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetPrivateKeyStringFromHDKey(nil)
		assert.Error(t, err)
	})
}

// ExampleGetPrivateKeyStringFromHDKey example using GetPrivateKeyStringFromHDKey()
func ExampleGetPrivateKeyStringFromHDKey() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var privateKey string
	if privateKey, err = compat.GetPrivateKeyStringFromHDKey(hdKey); err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("private key: %s", privateKey)
	// Output:private key: 0ccf07f2cbe10dbe6f6034b7efbf62fc83cac3d44f49d67aa22ac8893d294e7a
}

// BenchmarkGetPrivateKeyStringFromHDKey benchmarks the method GetPrivateKeyStringFromHDKey()
func BenchmarkGetPrivateKeyStringFromHDKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetPrivateKeyStringFromHDKey(hdKey)
	}
}

// TestGetPublicKeyFromHDKey will test the method GetPublicKeyFromHDKey()
func TestGetPublicKeyFromHDKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input         *compat.ExtendedKey
		expectedKey   string
		expectedNil   bool
		expectedError bool
	}{
		{new(compat.ExtendedKey), "", true, true},
		{validHdKey, "02f2a2942b9d1dba033d36ab0c193e680415f5c8c1ff5d854f805c8c42ed9dd1fd", false, false},
	}

	var publicKey *ec.PublicKey
	for _, test := range tests {
		if publicKey, err = compat.GetPublicKeyFromHDKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if publicKey == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was nil but not expected", t.Name(), test.input)
		} else if publicKey != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was NOT nil but expected to be nil", t.Name(), test.input)
		} else if publicKey != nil && hex.EncodeToString(publicKey.Compressed()) != test.expectedKey {
			t.Fatalf("%s Failed: [%v] inputted [%s] expected but got: %s", t.Name(), test.input, test.expectedKey, hex.EncodeToString(publicKey.Compressed()))
		}
	}
}

// TestGetPublicKeyFromHDKeyPanic tests for nil case in GetPublicKeyFromHDKey()
func TestGetPublicKeyFromHDKeyPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetPublicKeyFromHDKey(nil)
		assert.Error(t, err)
	})
}

// ExampleGetPublicKeyFromHDKey example using GetPublicKeyFromHDKey()
func ExampleGetPublicKeyFromHDKey() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var publicKey *ec.PublicKey
	if publicKey, err = compat.GetPublicKeyFromHDKey(hdKey); err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("public key: %s", hex.EncodeToString(publicKey.Compressed()))
	// Output:public key: 03a25f6c10eedcd41eebac22c6bbc5278690fa1aab3afc2bbe8f2277c85e5c5def
}

// BenchmarkGetPublicKeyFromHDKey benchmarks the method GetPublicKeyFromHDKey()
func BenchmarkGetPublicKeyFromHDKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetPublicKeyFromHDKey(hdKey)
	}
}

// TestGetAddressFromHDKey will test the method GetAddressFromHDKey()
func TestGetAddressFromHDKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input           *compat.ExtendedKey
		expectedAddress string
		expectedNil     bool
		expectedError   bool
	}{
		{new(compat.ExtendedKey), "", true, true},
		{validHdKey, "13xHrMdZuqa2gpweHf37w8hu6tfv3JrnaW", false, false},
	}

	var address *script.Address
	for _, test := range tests {
		if address, err = compat.GetAddressFromHDKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if address == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was nil but not expected", t.Name(), test.input)
		} else if address != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] inputted and was NOT nil but expected to be nil", t.Name(), test.input)
		} else if address != nil && address.AddressString != test.expectedAddress {
			t.Fatalf("%s Failed: [%v] inputted [%s] expected but got: %s", t.Name(), test.input, test.expectedAddress, address.AddressString)
		}
	}
}

// TestGetAddressFromHDKeyPanic tests for nil case in GetAddressFromHDKey()
func TestGetAddressFromHDKeyPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetAddressFromHDKey(nil)
		assert.Error(t, err)
	})
}

// ExampleGetAddressFromHDKey example using GetAddressFromHDKey()
func ExampleGetAddressFromHDKey() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var address *script.Address
	if address, err = compat.GetAddressFromHDKey(hdKey); err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("address: %s", address.AddressString)
	// Output:address: 18G2YRH3nRKRx8pnqVFUM5nAJhTZJ3YA4W
}

// BenchmarkGetAddressFromHDKey benchmarks the method GetAddressFromHDKey()
func BenchmarkGetAddressFromHDKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetAddressFromHDKey(hdKey)
	}
}

// TestGetAddressStringFromHDKey will test the method GetAddressStringFromHDKey()
func TestGetAddressStringFromHDKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input           *compat.ExtendedKey
		expectedAddress string
		expectedError   bool
	}{
		{new(compat.ExtendedKey), "", true},
		{validHdKey, "13xHrMdZuqa2gpweHf37w8hu6tfv3JrnaW", false},
	}

	var address string
	for _, test := range tests {
		if address, err = compat.GetAddressStringFromHDKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if address != test.expectedAddress {
			t.Fatalf("%s Failed: [%v] inputted [%s] expected but got: %s", t.Name(), test.input, test.expectedAddress, address)
		}
	}
}

// TestGetAddressStringFromHDKeyPanic tests for nil case in GetAddressStringFromHDKey()
func TestGetAddressStringFromHDKeyPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetAddressStringFromHDKey(nil)
		assert.Error(t, err)
	})
}

// ExampleGetAddressStringFromHDKey example using GetAddressStringFromHDKey()
func ExampleGetAddressStringFromHDKey() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var address string
	if address, err = compat.GetAddressStringFromHDKey(hdKey); err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("address: %s", address)
	// Output:address: 18G2YRH3nRKRx8pnqVFUM5nAJhTZJ3YA4W
}

// BenchmarkGetAddressStringFromHDKey benchmarks the method GetAddressStringFromHDKey()
func BenchmarkGetAddressStringFromHDKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetAddressStringFromHDKey(hdKey)
	}
}

// TestGetPublicKeysForPath will test the method GetPublicKeysForPath()
func TestGetPublicKeysForPath(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	assert.NotNil(t, validHdKey)

	var tests = []struct {
		input           *compat.ExtendedKey
		inputNum        uint32
		expectedPubKey1 string
		expectedPubKey2 string
		expectedNil     bool
		expectedError   bool
	}{
		{new(compat.ExtendedKey), 1, "", "", true, true},
		{validHdKey, 1, "03cc3334f0a6f0fae0420d1442ca0ce64fad0da76d652f2cc3b333e7ed95b97259", "02ceb23902f8dcf6fbff656597ee0343e05c907c6dfcdd8aaf6d033e14e85fd955", false, false},
		{validHdKey, 2, "020cb908e3b9f3de7c9b40e7bcce63708c5617536d85cf4ab5635e3d3819c02c37", "030007ae60fc6eef98ea17b4f80f9b791e61ea94936e8a9e6ec343eeaa50a875e0", false, false},
		{validHdKey, 3, "0342593453c476ac6c78eb1b1e586df00b20352e61c42536fe1b33c9fdf3bfbb6f", "03786a41dbf0b099256da26cb0019e10063628f6ce31b96801703f1bb2e1b17724", false, false},
		{validHdKey, 4, "0366dcdebfc8abfd34bffc181ccb54f1706839a80ad4f0842ae5a43f39fdd35c1e", "03a095db29ae9ee0b22c775118b4444b59db40acdea137fd9ecd9c68dacf50a644", false, false},
	}

	var pubKeys []*ec.PublicKey
	for _, test := range tests {
		if pubKeys, err = compat.GetPublicKeysForPath(test.input, test.inputNum); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error not expected but got: %s", t.Name(), test.input, test.inputNum, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error was expected", t.Name(), test.input, test.inputNum)
		} else if pubKeys == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was nil but not expected", t.Name(), test.input, test.inputNum)
		} else if pubKeys != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.input, test.inputNum)
		} else if pubKeys != nil && hex.EncodeToString(pubKeys[0].Compressed()) != test.expectedPubKey1 {
			t.Fatalf("%s Failed: [%v] [%d] inputted key 1 [%s] expected but got: %s", t.Name(), test.input, test.inputNum, test.expectedPubKey1, hex.EncodeToString(pubKeys[0].Compressed()))
		} else if pubKeys != nil && hex.EncodeToString(pubKeys[1].Compressed()) != test.expectedPubKey2 {
			t.Fatalf("%s Failed: [%v] [%d] inputted key 2 [%s] expected but got: %s", t.Name(), test.input, test.inputNum, test.expectedPubKey2, hex.EncodeToString(pubKeys[1].Compressed()))
		}
	}
}

// TestGetPublicKeysForPathPanic tests for nil case in GetPublicKeysForPath()
func TestGetPublicKeysForPathPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_, err := compat.GetPublicKeysForPath(nil, 1)
		assert.Error(t, err)
	})
}

// ExampleGetPublicKeysForPath example using GetPublicKeysForPath()
func ExampleGetPublicKeysForPath() {

	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var publicKeys []*ec.PublicKey
	publicKeys, err = compat.GetPublicKeysForPath(hdKey, 5)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("found [%d] keys! Key 1: %s Key 2: %s", len(publicKeys), hex.EncodeToString(publicKeys[0].Compressed()), hex.EncodeToString(publicKeys[1].Compressed()))
	// Output:found [2] keys! Key 1: 03f87ac38fb0cfca12988b51a2f1cd3e85bb4aeb1b05f549682190ac8205a67d30 Key 2: 02e78303aeef1acce1347c6493fadc1914e6d85ef3189a8856afb3accd53fbd9c5
}

// BenchmarkGetPublicKeysForPath benchmarks the method GetPublicKeysForPath()
func BenchmarkGetPublicKeysForPath(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetPublicKeysForPath(hdKey, 5)
	}
}

// TestGetAddressesForPath will test the method GetAddressesForPath()
func TestGetAddressesForPath(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	require.NotNil(t, validHdKey)

	var tests = []struct {
		input            *compat.ExtendedKey
		inputNum         uint32
		expectedAddress1 string
		expectedAddress2 string
		expectedNil      bool
		expectedError    bool
	}{
		{new(compat.ExtendedKey), 1, "", "", true, true},
		{validHdKey, 1, "1KMxfSfRCkC1jrBAuYaLde4XBzdsWApbdH", "174DL9ZbBWx568ssAg8w2YwW6FTTBwXGEu", false, false},
		{validHdKey, 2, "18s3peTU7fMSwgui54avpnqm1126pRVccw", "1KgZZ3NsJDw3v1GPHBj8ASnxutA1kFxo2i", false, false},
	}

	var addresses []string
	for _, test := range tests {
		if addresses, err = compat.GetAddressesForPath(test.input, test.inputNum); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error not expected but got: %s", t.Name(), test.input, test.inputNum, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] [%d] inputted and error was expected", t.Name(), test.input, test.inputNum)
		} else if addresses == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was nil but not expected", t.Name(), test.input, test.inputNum)
		} else if addresses != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%v] [%d] inputted and was NOT nil but expected to be nil", t.Name(), test.input, test.inputNum)
		} else if addresses != nil && addresses[0] != test.expectedAddress1 {
			t.Fatalf("%s Failed: [%v] [%d] inputted address 1 [%s] expected but got: %s", t.Name(), test.input, test.inputNum, test.expectedAddress1, addresses[0])
		} else if addresses != nil && addresses[1] != test.expectedAddress2 {
			t.Fatalf("%s Failed: [%v] [%d] inputted address 2 [%s] expected but got: %s", t.Name(), test.input, test.inputNum, test.expectedAddress2, addresses[1])
		}
	}
}

// TestGetAddressesForPathPanic tests for nil case in GetAddressesForPath()
func TestGetAddressesForPathPanic(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_, err := compat.GetAddressesForPath(nil, 1)
		require.Error(t, err)
	})
}

// ExampleGetAddressesForPath example using GetAddressesForPath()
func ExampleGetAddressesForPath() {
	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var addresses []string
	addresses, err = compat.GetAddressesForPath(hdKey, 5)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	fmt.Printf("found [%d] addresses! Address 1: %s Address 2: %s", len(addresses), addresses[0], addresses[1])
	// Output:found [2] addresses! Address 1: 1JHGJTqsiFHo4yQYJ1WbTvbxYMZC7nZKYb Address 2: 1DTHBcGeJFRmS26S11tt2EddhSkFM8tmze
}

// BenchmarkGetAddressesForPath benchmarks the method GetAddressesForPath()
func BenchmarkGetAddressesForPath(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetAddressesForPath(hdKey, 5)
	}
}

// TestGetExtendedPublicKey will test the method GetExtendedPublicKey()
func TestGetExtendedPublicKey(t *testing.T) {
	t.Parallel()

	validHdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K4FdJCmPQe1CFUvK3PKVrcp3b5xVr5Bs3cP5ab6ytszeHggTmHoqTXpaa8CgYPxZZzigSGCDjtyWdUDJqPogb1JGWAPkBLdF")
	require.NoError(t, err)
	require.NotNil(t, validHdKey)

	var tests = []struct {
		input         *compat.ExtendedKey
		expectedKey   string
		expectedError bool
	}{
		{validHdKey, "xpub661MyMwAqRbcGjhmJnvR198z2x9XnnDhz2yBtLuTdXQ2VBQj8eJ9RnxmXxKnRPhYy6nLsmabmUfVkbajvP7aZASrrnoZkzmwgyjiNskiefG", false},
		{new(compat.ExtendedKey), "zeroed extended key", false},
	}

	var xPub string
	for _, test := range tests {
		if xPub, err = compat.GetExtendedPublicKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%v] inputted and error was expected", t.Name(), test.input)
		} else if xPub != test.expectedKey {
			t.Fatalf("%s Failed: [%v] inputted and [%s] expected but got: %s", t.Name(), test.input, test.expectedKey, xPub)
		}
	}
}

// TestGetExtendedPublicKeyPanic tests for nil case in GetExtendedPublicKey()
func TestGetExtendedPublicKeyPanic(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_, err := compat.GetExtendedPublicKey(nil)
		require.Error(t, err)
	})
}

// ExampleGetExtendedPublicKey example using GetExtendedPublicKey()
func ExampleGetExtendedPublicKey() {
	hdKey, err := compat.GenerateHDKeyFromString("xprv9s21ZrQH143K3PZSwbEeXEYq74EbnfMngzAiMCZcfjzyRpUvt2vQJnaHRTZjeuEmLXeN6BzYRoFsEckfobxE9XaRzeLGfQoxzPzTRyRb6oE")
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	var xPub string
	xPub, err = compat.GetExtendedPublicKey(hdKey)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}
	fmt.Printf("xPub: %s", xPub)
	// Output:xPub: xpub661MyMwAqRbcFsdv3cmetNVZf656C85e4D6K9ayEE5XxJcp5RaEeratmGmg7ggt3ZShibYcsusYPom69yDG9hf3UE1i4LrXJbuA9d7hPujt
}

// BenchmarkGetExtendedPublicKey benchmarks the method GetExtendedPublicKey()
func BenchmarkGetExtendedPublicKey(b *testing.B) {
	hdKey, _ := compat.GenerateHDKey(compat.SecureSeedLength)
	for i := 0; i < b.N; i++ {
		_, _ = compat.GetExtendedPublicKey(hdKey)
	}
}

// TestGetHDKeyFromExtendedPublicKey will test the method GetHDKeyFromExtendedPublicKey()
func TestGetHDKeyFromExtendedPublicKey(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		input         string
		expectedKey   string
		expectedError bool
		expectedNil   bool
	}{
		{
			"xpub661MyMwAqRbcGjhmJnvR198z2x9XnnDhz2yBtLuTdXQ2VBQj8eJ9RnxmXxKnRPhYy6nLsmabmUfVkbajvP7aZASrrnoZkzmwgyjiNskiefG",
			"xpub661MyMwAqRbcGjhmJnvR198z2x9XnnDhz2yBtLuTdXQ2VBQj8eJ9RnxmXxKnRPhYy6nLsmabmUfVkbajvP7aZASrrnoZkzmwgyjiNskiefG",
			false,
			false,
		},
		{
			"xpub661MyMwAqRbcH3WGvLjupmr43L1GVH3MP2WQWvdreDraBeFJy64Xxv4LLX9ZVWWz3ZjZkMuZtSsc9qH9JZR74bR4PWkmtEvP423r6DJR8kA",
			"xpub661MyMwAqRbcH3WGvLjupmr43L1GVH3MP2WQWvdreDraBeFJy64Xxv4LLX9ZVWWz3ZjZkMuZtSsc9qH9JZR74bR4PWkmtEvP423r6DJR8kA",
			false,
			false,
		},
		{
			"",
			"",
			true,
			true,
		},
		{
			"0",
			"",
			true,
			true,
		},
	}

	for _, test := range tests {
		if xPub, err := compat.GetHDKeyFromExtendedPublicKey(test.input); err != nil && !test.expectedError {
			t.Fatalf("%s Failed: [%s] inputted and error not expected but got: %s", t.Name(), test.input, err.Error())
		} else if err == nil && test.expectedError {
			t.Fatalf("%s Failed: [%s] inputted and error was expected", t.Name(), test.input)
		} else if xPub == nil && !test.expectedNil {
			t.Fatalf("%s Failed: [%s] inputted and was nil but not expected", t.Name(), test.input)
		} else if xPub != nil && test.expectedNil {
			t.Fatalf("%s Failed: [%s] inputted and was NOT nil but expected to be nil", t.Name(), test.input)
		} else if xPub != nil && xPub.String() != test.expectedKey {
			t.Fatalf("%s Failed: [%s] inputted and [%s] expected but got: %s", t.Name(), test.input, test.expectedKey, xPub)
		}
	}
}

// ExampleGetHDKeyFromExtendedPublicKey example using GetHDKeyFromExtendedPublicKey()
func ExampleGetHDKeyFromExtendedPublicKey() {
	// Start with an existing xPub
	xPub := "xpub661MyMwAqRbcH3WGvLjupmr43L1GVH3MP2WQWvdreDraBeFJy64Xxv4LLX9ZVWWz3ZjZkMuZtSsc9qH9JZR74bR4PWkmtEvP423r6DJR8kA"

	// Convert to a HD key
	key, err := compat.GetHDKeyFromExtendedPublicKey(xPub)
	if err != nil {
		fmt.Printf("error occurred: %s", err.Error())
		return
	}

	fmt.Printf("key: %s", key.String())
	// Output:key: xpub661MyMwAqRbcH3WGvLjupmr43L1GVH3MP2WQWvdreDraBeFJy64Xxv4LLX9ZVWWz3ZjZkMuZtSsc9qH9JZR74bR4PWkmtEvP423r6DJR8kA
}

// BenchmarkGetHDKeyFromExtendedPublicKey benchmarks the method GetHDKeyFromExtendedPublicKey()
func BenchmarkGetHDKeyFromExtendedPublicKey(b *testing.B) {
	xPub := "xpub661MyMwAqRbcH3WGvLjupmr43L1GVH3MP2WQWvdreDraBeFJy64Xxv4LLX9ZVWWz3ZjZkMuZtSsc9qH9JZR74bR4PWkmtEvP423r6DJR8kA"
	for i := 0; i < b.N; i++ {
		key, err := compat.GetHDKeyFromExtendedPublicKey(xPub)
		require.NoError(b, err)
		require.NotNil(b, key)
	}
}
