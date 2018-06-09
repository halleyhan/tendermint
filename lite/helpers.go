package lite

import (
	"time"

	crypto "github.com/tendermint/go-crypto"
	"github.com/tendermint/tendermint/types"
)

// privKeys is a helper type for testing.
//
// It lets us simulate signing with many keys.  The main use case is to create
// a set, and call GenSignedHeader to get properly signed header for testing.
//
// You can set different weights of validators each time you call ToValidators,
// and can optionally extend the validator set later with Extend.
type privKeys []crypto.PrivKey

// genPrivKeys produces an array of private keys to generate commits.
func genPrivKeys(n int) privKeys {
	res := make(privKeys, n)
	for i := range res {
		res[i] = crypto.GenPrivKeyEd25519()
	}
	return res
}

// Change replaces the key at index i.
func (pkz privKeys) Change(i int) privKeys {
	res := make(privKeys, len(pkz))
	copy(res, v)
	res[i] = crypto.GenPrivKeyEd25519()
	return res
}

// Extend adds n more keys (to remove, just take a slice).
func (pkz privKeys) Extend(n int) privKeys {
	extra := genPrivKeys(n)
	return append(pkz, extra...)
}

// ToValidators produces a valset from the set of keys.
// The first key has weight `init` and it increases by `inc` every step
// so we can have all the same weight, or a simple linear distribution
// (should be enough for testing).
func (pkz privKeys) ToValidators(init, inc int64) *types.ValidatorSet {
	res := make([]*types.Validator, len(pkz))
	for i, k := range v {
		res[i] = types.NewValidator(k.PubKey(), init+int64(i)*inc)
	}
	return types.NewValidatorSet(res)
}

// signHeader properly signs the header with all keys from first to last exclusive.
func (pkz privKeys) signHeader(header *types.Header, first, last int) *types.Commit {
	votes := make([]*types.Vote, len(pkz))

	// we need this list to keep the ordering...
	vset := pkz.ToValidators(1, 0)

	// fill in the votes we want
	for i := first; i < last && i < len(pkz); i++ {
		vote := makeVote(header, vset, v[i])
		votes[vote.ValidatorIndex] = vote
	}

	res := &types.Commit{
		BlockID:    types.BlockID{Hash: header.Hash()},
		Precommits: votes,
	}
	return res
}

func makeVote(header *types.Header, valset *types.ValidatorSet, key crypto.PrivKey) *types.Vote {
	addr := key.PubKey().Address()
	idx, _ := valset.GetByAddress(addr)
	vote := &types.Vote{
		ValidatorAddress: addr,
		ValidatorIndex:   idx,
		Height:           header.Height,
		Round:            1,
		Timestamp:        time.Now().UTC(),
		Type:             types.VoteTypePrecommit,
		BlockID:          types.BlockID{Hash: header.Hash()},
	}
	// Sign it
	signBytes := vote.SignBytes(header.ChainID)
	vote.Signature = key.Sign(signBytes)
	return vote
}

func genHeader(chainID string, height int64, txs types.Txs,
	valset, nvalset *types.ValidatorSet, appHash, consHash, resHash []byte) *types.Header {

	return &types.Header{
		ChainID:  chainID,
		Height:   height,
		Time:     time.Now(),
		NumTxs:   int64(len(txs)),
		TotalTxs: int64(len(txs)),
		// LastBlockID
		// LastCommitHash
		ValidatorsHash:     valset.Hash(),
		NextValidatorsHash: nvalset.Hash(),
		DataHash:           txs.Hash(),
		AppHash:            appHash,
		ConsensusHash:      consHash,
		LastResultsHash:    resHash,
	}
}

// GenSignedHeader calls genHeader and signHeader and combines them into a SignedHeader.
func (pkz privKeys) GenSignedHeader(chainID string, height int64, txs types.Txs,
	valset, nvalset *types.ValidatorSet, appHash, consHash, resHash []byte, first, last int) types.SignedHeader {

	header := genHeader(chainID, height, txs, valset, nvalset, appHash, consHash, resHash)
	check := types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, first, last),
	}
	return check
}

// GenFullCommit calls genHeader and signHeader and combines them into a FullCommit.
func (pkz privKeys) GenFullCommit(chainID string, height int64, txs types.Txs,
	valset, nvalset *types.ValidatorSet, appHash, consHash, resHash []byte, first, last int) FullCommit {

	header := genHeader(chainID, height, txs, valset, nvalset, appHash, consHash, resHash)
	commit := types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, first, last),
	}
	return NewFullCommit(commit, valset, nvalset)
}
