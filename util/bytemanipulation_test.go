package util_test

import (
	"encoding/hex"
	"testing"

	crypto "github.com/bitcoin-sv/go-sdk/primitives/hash"
	"github.com/stretchr/testify/require"

	"github.com/bitcoin-sv/go-sdk/util"
)

func TestLittleEndianBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    uint32
		length   uint32
		expected []byte
	}{
		{
			name:     "Zero value",
			input:    0,
			length:   4,
			expected: []byte{0, 0, 0, 0},
		},
		{
			name:     "Small number",
			input:    258,
			length:   4,
			expected: []byte{2, 1, 0, 0},
		},
		{
			name:     "Large number",
			input:    0xFFFFFFFF,
			length:   4,
			expected: []byte{255, 255, 255, 255},
		},
		{
			name:     "Custom length (3 bytes)",
			input:    0x12345678,
			length:   3,
			expected: []byte{0x78, 0x56, 0x34},
		},
		{
			name:     "Custom length (2 bytes)",
			input:    0x12345678,
			length:   2,
			expected: []byte{0x78, 0x56},
		},
		{
			name:     "Custom length (1 byte)",
			input:    0x12345678,
			length:   1,
			expected: []byte{0x78},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := util.LittleEndianBytes(tc.input, tc.length)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestReverseBytes(t *testing.T) {
	t.Parallel()

	t.Run("Empty slice", func(t *testing.T) {
		input := []byte{}
		result := util.ReverseBytes(input)
		require.Equal(t, input, result)
	})

	t.Run("Single byte", func(t *testing.T) {
		input := []byte{0x01}
		result := util.ReverseBytes(input)
		require.Equal(t, input, result)
	})

	t.Run("Multiple bytes", func(t *testing.T) {
		input := []byte{0x01, 0x02, 0x03, 0x04}
		expected := []byte{0x04, 0x03, 0x02, 0x01}
		result := util.ReverseBytes(input)
		require.Equal(t, expected, result)
	})

	t.Run("Odd number of bytes", func(t *testing.T) {
		input := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		expected := []byte{0x05, 0x04, 0x03, 0x02, 0x01}
		result := util.ReverseBytes(input)
		require.Equal(t, expected, result)
	})

	t.Run("genesis hash", func(t *testing.T) {
		b, err := hex.DecodeString("01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4d04ffff001d0104455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73ffffffff0100f2052a01000000434104678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5fac00000000")
		require.NoError(t, err)

		h := util.ReverseBytes(crypto.Sha256d(b))

		require.Equal(t,
			"4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b",
			hex.EncodeToString(h),
		)
	})

	t.Run("genesis block hash", func(t *testing.T) {
		b, err := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c0101000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4d04ffff001d0104455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73ffffffff0100f2052a01000000434104678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5fac00000000")
		require.NoError(t, err)

		h := util.ReverseBytes(crypto.Sha256d(b[0:80]))

		require.Equal(t,
			"000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			hex.EncodeToString(h),
		)
	})
}

func TestReverseBytesInPlace(t *testing.T) {
	t.Parallel()

	t.Run("Empty slice", func(t *testing.T) {
		input := []byte{}
		util.ReverseBytesInPlace(input)
		require.Equal(t, []byte{}, input)
	})

	t.Run("Single byte", func(t *testing.T) {
		input := []byte{0x01}
		util.ReverseBytesInPlace(input)
		require.Equal(t, []byte{0x01}, input)
	})

	t.Run("Multiple bytes", func(t *testing.T) {
		input := []byte{0x01, 0x02, 0x03, 0x04}
		expected := []byte{0x04, 0x03, 0x02, 0x01}
		util.ReverseBytesInPlace(input)
		require.Equal(t, expected, input)
	})

	t.Run("Odd number of bytes", func(t *testing.T) {
		input := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		expected := []byte{0x05, 0x04, 0x03, 0x02, 0x01}
		util.ReverseBytesInPlace(input)
		require.Equal(t, expected, input)
	})
}
