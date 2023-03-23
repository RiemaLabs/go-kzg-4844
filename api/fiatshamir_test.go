package api

import (
	"bytes"
	"testing"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/crate-crypto/go-proto-danksharding-crypto/serialization"
)

// This is both an interop test and a regression check
// If the way computeChallenge is computed is updated
// then this test will fail
func TestComputeChallengeInterop(t *testing.T) {
	blob := serialization.Blob{}
	commitment := serialization.SerializeKZGCommitment(&bls12381.G1Affine{})
	challenge := computeChallenge(&blob, &commitment)
	expected := []byte{
		59, 127, 233, 79, 178, 22, 242, 95,
		176, 209, 125, 10, 193, 90, 102, 229,
		56, 104, 204, 58, 237, 60, 121, 97,
		77, 194, 248, 45, 172, 7, 224, 74,
	}
	got := serialization.SerializeScalar(&challenge)
	if !bytes.Equal(expected, got[:]) {
		t.Fatalf("computeChallenge has changed and or regressed")
	}
}

func TestTo16Bytes(t *testing.T) {
	number := uint64(4096)
	// Generated using the following python snippet:
	// FIELD_ELEMENTS_PER_BLOB = 4096
	// degree_poly = int.to_bytes(FIELD_ELEMENTS_PER_BLOB, 16, 'little')
	// " ".join(format(x, "d") for x in degree_poly)
	expected := []byte{0, 16, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	got := u64ToByteArray16(number)
	if !bytes.Equal(expected, got) {
		t.Fatalf("unexpected byte array when converting a u64 to bytes,\n got %v \n expected %v", got, expected)
	}
}
