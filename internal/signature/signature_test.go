package signature

import (
	"fmt"
	"math/big"
	"testing"

	"crypto/rand"

	"github.com/stretchr/testify/assert"
)

func generateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("rand.Read: %w", err)
	}

	return bytes, nil
}

func BenchmarkCalculateHashSum(b *testing.B) {
	// hex.EncodeToString(bytes)[:length]

	bytesData := make([][]byte, 10000)

	// Set max random value
	mx := big.NewInt(100)

	// Generate a random number using crypto/rand with max as the upper bound
	randInt, err := rand.Int(rand.Reader, mx)
	assert.NoError(b, err)

	for i := 0; i < len(bytesData); i++ {
		var err error

		bytesData[i], err = generateRandomBytes(int(randInt.Int64()))
		assert.NoError(b, err)
	}

	// Reset counter to avoid including the initialization in the benchmark
	b.ResetTimer()

	counter := 0

	for i := 0; i < b.N; i++ {
		_, err := CalculateHashSum([]byte("key"), bytesData[counter])
		assert.NoError(b, err)

		counter++

		if counter > len(bytesData)-1 {
			counter = 0
		}
	}
}
