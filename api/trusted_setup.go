package api

import (
	"bytes"
	_ "embed"
	"errors"
	"sync"

	"encoding/hex"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/crate-crypto/go-proto-danksharding-crypto/internal/kzg"
	"github.com/crate-crypto/go-proto-danksharding-crypto/serialization"
)

// This library will not :
// - Check that the points are in the correct subgroup.
// - Check that setupG1Lagrange is the lagrange version of setupG1.
//
// Note: There is an embedded `JSONTrustedSetup` to which we do check those properties
// in a test function.
type JSONTrustedSetup struct {
	SetupG1         [serialization.ScalarsPerBlob]G1CompressedHexStr `json:"setup_G1"`
	SetupG2         []G2CompressedHexStr                             `json:"setup_G2"`
	SetupG1Lagrange [serialization.ScalarsPerBlob]G2CompressedHexStr `json:"setup_G1_lagrange"`
}

// Hex string for a compressed G1 point without the `0x` prefix
type G1CompressedHexStr = string

// Hex string for a compressed G2 point without the `0x` prefix
type G2CompressedHexStr = string

var (
	// This is the test trusted setup, which SHOULD NOT BE USED IN PRODUCTION.
	// The secret for this 1337.
	//
	//go:embed trusted_setup.json
	testKzgSetupStr string
)

// Checks whether the trusted setup is well-formed.
// This checks that:
// - Length of the monomial version of G1 Points is equal to the length of the
// lagrange version of G1 points
// - All elements are in the correct subgroup
// - Lagrange G1 points are obtained by doing an IFFT of monomial G1 points
func CheckTrustedSetupWellFormed(trustedSetup *JSONTrustedSetup) error {

	if len(trustedSetup.SetupG1) != len(trustedSetup.SetupG1Lagrange) {
		return errLagrangeMonomialLengthMismatch
	}

	var setupG1Points []bls12381.G1Affine
	for i := 0; i < len(trustedSetup.SetupG1); i++ {
		var point bls12381.G1Affine
		byts, err := hex.DecodeString(trustedSetup.SetupG1[i])
		if err != nil {
			return err
		}
		_, err = point.SetBytes(byts)
		if err != nil {
			return err
		}
		setupG1Points = append(setupG1Points, point)
	}

	domain := kzg.NewDomain(serialization.ScalarsPerBlob)
	// The G1 points will be in monomial form
	// Convert them to lagrange form
	// See 3.1 onwards in https://eprint.iacr.org/2017/602.pdf for further details
	setupLagrangeG1 := domain.IfftG1(setupG1Points)

	for i := 0; i < len(setupLagrangeG1); i++ {
		serializedPoint := setupLagrangeG1[i].Bytes()
		if hex.EncodeToString(serializedPoint[:]) != trustedSetup.SetupG1Lagrange[i] {
			return errors.New("unexpected lagrange setup being used")
		}
	}

	for i := 0; i < len(trustedSetup.SetupG2); i++ {
		var point bls12381.G2Affine
		byts, err := hex.DecodeString(trustedSetup.SetupG2[i])
		if err != nil {
			return err
		}
		_, err = point.SetBytes(byts)
		if err != nil {
			return err
		}
	}

	return nil
}

// Parses the trusted setup into corresponding group elements.
// Elements are assumed to be trusted.
func parseTrustedSetup(trustedSetup *JSONTrustedSetup) (bls12381.G1Affine, []bls12381.G1Affine, []bls12381.G2Affine, error) {
	// Take the generator point from the monomial SRS
	if len(trustedSetup.SetupG1) < 1 {
		return bls12381.G1Affine{}, nil, nil, kzg.ErrMinSRSSize
	}
	genG1, err := parseG1PointNoSubgroupCheck(trustedSetup.SetupG1[0])
	if err != nil {
		return bls12381.G1Affine{}, nil, nil, err
	}

	setupLagrangeG1Points, err := parseG1PointsNoSubgroupCheck(trustedSetup.SetupG1Lagrange[:])
	if err != nil {
		return bls12381.G1Affine{}, nil, nil, err
	}

	g2Points, err := parseG2PointsNoSubgroupCheck(trustedSetup.SetupG2)
	if err != nil {
		return bls12381.G1Affine{}, nil, nil, err
	}

	return genG1, setupLagrangeG1Points, g2Points, nil
}

func parseG1PointNoSubgroupCheck(hexString string) (bls12381.G1Affine, error) {
	byts, err := hex.DecodeString(hexString)
	if err != nil {
		return bls12381.G1Affine{}, err
	}

	var point bls12381.G1Affine
	noSubgroupCheck := bls12381.NoSubgroupChecks()
	d := bls12381.NewDecoder(bytes.NewReader(byts), noSubgroupCheck)

	return point, d.Decode(&point)

}
func parseG2PointNoSubgroupCheck(hexString string) (bls12381.G2Affine, error) {
	byts, err := hex.DecodeString(hexString)
	if err != nil {
		return bls12381.G2Affine{}, err
	}

	var point bls12381.G2Affine
	noSubgroupCheck := bls12381.NoSubgroupChecks()
	d := bls12381.NewDecoder(bytes.NewReader(byts), noSubgroupCheck)

	return point, d.Decode(&point)
}

func parseG1PointsNoSubgroupCheck(hexStrings []string) ([]bls12381.G1Affine, error) {
	numG1 := len(hexStrings)
	g1Points := make([]bls12381.G1Affine, numG1)

	var wg sync.WaitGroup
	wg.Add(numG1)
	for i := 0; i < numG1; i++ {
		go func(_i int) {
			g1Point, err := parseG1PointNoSubgroupCheck(hexStrings[_i])
			if err != nil {
				panic(err)
			}
			g1Points[_i] = g1Point
			wg.Done()
		}(i)
	}
	wg.Wait()

	return g1Points, nil
}
func parseG2PointsNoSubgroupCheck(hexStrings []string) ([]bls12381.G2Affine, error) {
	numG2 := len(hexStrings)
	g2Points := make([]bls12381.G2Affine, numG2)

	var wg sync.WaitGroup
	wg.Add(numG2)
	for i := 0; i < numG2; i++ {
		go func(_i int) {
			g2Point, err := parseG2PointNoSubgroupCheck(hexStrings[_i])
			if err != nil {
				panic(err)
			}
			g2Points[_i] = g2Point
			wg.Done()
		}(i)
	}
	wg.Wait()

	return g2Points, nil
}
