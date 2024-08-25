package signature

import (
	"math/big"
	"testing"

	"crypto/rand"

	"github.com/stretchr/testify/assert"
)

func generateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func BenchmarkCalculateHashSum(b *testing.B) {
	// hex.EncodeToString(bytes)[:length]

	bytesData := make([][]byte, 10000)

	// Set max random value
	max := big.NewInt(100)

	// Generate a random number using crypto/rand with max as the upper bound
	randInt, err := rand.Int(rand.Reader, max)
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
		CalculateHashSum([]byte("key"), bytesData[counter])

		counter++

		if counter > len(bytesData)-1 {
			counter = 0
		}
	}
}
