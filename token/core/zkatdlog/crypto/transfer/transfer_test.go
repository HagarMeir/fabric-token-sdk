/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transfer_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transfer", func() {
	var (
		prover   *transfer.Prover
		verifier *transfer.Verifier
	)
	BeforeEach(func() {
		prover, verifier = prepareZKTransfer()
	})
	Describe("Prove", func() {
		Context("parameters and witness are initialized correctly", func() {
			It("Succeeds", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("Output Values > Input Values", func() {
			BeforeEach(func() {
				prover, verifier = prepareZKTransferWithWrongSum()
			})
			It("fails", func() {
				proof, err := prover.Prove()
				Expect(err).NotTo(HaveOccurred())
				Expect(proof).NotTo(BeNil())
				err = verifier.Verify(proof)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid zero-knowledge transfer"))
			})
		})
		Context("Output Values out of range", func() {
			BeforeEach(func() {
				prover, verifier = prepareZKTransferWithInvalidRange()
			})
			It("fails during proof generation", func() {
				proof, err := prover.Prove()
				Expect(proof).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("can't compute range proof: value of token outside authorized range"))
			})
		})
	})

})

func prepareZKTransfer() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(100, 2, nil)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}
	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithWrongSum() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(100, 2, nil)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInvalidInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)

	return prover, verifier
}

func prepareZKTransferWithInvalidRange() (*transfer.Prover, *transfer.Verifier) {
	pp, err := crypto.Setup(10, 2, nil)
	Expect(err).NotTo(HaveOccurred())

	wfw, in, out := prepareInputsForZKTransfer(pp)

	inBF := wfw.GetInBlindingFators()
	outBF := wfw.GetOutBlindingFators()

	inValues := wfw.GetInValues()
	outValues := wfw.GetOutValues()

	ttype := "ABC"
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	prover := transfer.NewProver(intw, outtw, in, out, pp)
	verifier := transfer.NewVerifier(in, out, pp)
	return prover, verifier
}

func prepareInputsForZKTransfer(pp *crypto.PublicParams) (*transfer.WellFormednessWitness, []*bn256.G1, []*bn256.G1) {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*bn256.Zr, 2)
	outBF := make([]*bn256.Zr, 2)
	inValues := make([]*bn256.Zr, 2)
	outValues := make([]*bn256.Zr, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = bn256.RandModOrder(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = bn256.RandModOrder(rand)
	}
	ttype := "ABC"
	inValues[0] = bn256.NewZrInt(90)
	inValues[1] = bn256.NewZrInt(60)
	outValues[0] = bn256.NewZrInt(50)
	outValues[1] = bn256.NewZrInt(100)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.ZKATPedParams)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return transfer.NewWellFormednessWitness(intw, outtw), in, out
}

func prepareInvalidInputsForZKTransfer(pp *crypto.PublicParams) (*transfer.WellFormednessWitness, []*bn256.G1, []*bn256.G1) {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())

	inBF := make([]*bn256.Zr, 2)
	outBF := make([]*bn256.Zr, 2)
	inValues := make([]*bn256.Zr, 2)
	outValues := make([]*bn256.Zr, 2)
	for i := 0; i < 2; i++ {
		inBF[i] = bn256.RandModOrder(rand)
	}
	for i := 0; i < 2; i++ {
		outBF[i] = bn256.RandModOrder(rand)
	}
	ttype := "ABC"
	inValues[0] = bn256.NewZrInt(90)
	inValues[1] = bn256.NewZrInt(60)
	outValues[0] = bn256.NewZrInt(110)
	outValues[1] = bn256.NewZrInt(45)

	in, out := prepareInputsOutputs(inValues, outValues, inBF, outBF, ttype, pp.ZKATPedParams)
	intw := make([]*token.TokenDataWitness, len(inValues))
	for i := 0; i < len(intw); i++ {
		intw[i] = &token.TokenDataWitness{BlindingFactor: inBF[i], Value: inValues[i], Type: ttype}
	}

	outtw := make([]*token.TokenDataWitness, len(outValues))
	for i := 0; i < len(outtw); i++ {
		outtw[i] = &token.TokenDataWitness{BlindingFactor: outBF[i], Value: outValues[i], Type: ttype}
	}

	return transfer.NewWellFormednessWitness(intw, outtw), in, out
}
