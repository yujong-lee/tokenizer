package tokenizer

import (
	"errors"

	"github.com/sugarme/sermo/normalizer"
)

type PaddingDir int

const (
	Left PaddingDir = iota
	Right
)

// Encoding represents the output of tokenizer
type Encoding struct {
	Normalized       normalizer.Normalized
	Ids              []uint32
	TypeIds          []uint32
	Tokens           []string
	Offsets          []Offset
	SpecialTokenMask []uint32
	AttentionMask    []uint32
	Overflowing      []Encoding
}

// NewEncoding initiate a new encoding from input data
func NewEncoding(normalized normalizer.Normalized, ids []uint32, typeIds []uint32, tokens []string, offsets []Offset, specialTokenMask []uint32, attentionMask []uint32, overflowing []Encoding) Encoding {
	return Encoding{
		normalized,
		ids,
		typeIds,
		tokens,
		offsets,
		specialTokenMask,
		attentionMask,
		overflowing,
	}
}

// GetNormalized returns normalized string from encoding
func (e *Encoding) GetNormalized() normalizer.Normalized {
	return e.Normalized
}

// GetTokenreturns tokens from encoding
func (e *Encoding) GetTokens() []string {
	return e.Tokens
}

// GetIds returns Ids from encoding
func (e *Encoding) GetIds() []uint32 {
	return e.Ids
}

// GetTypeIds returns type Ids from encoding
func (e *Encoding) GetTypeIds() []uint32 {
	return e.TypeIds
}

// GetOffsets returns offsets from encoding
func (e *Encoding) GetOffsets() []Offset {
	return e.Offsets
}

// GetSpecialTokenMask returns specialTokenMask from encoding
func (e *Encoding) GetSpecialTokenMask() []uint32 {
	return e.SpecialTokenMask
}

// GetAttentionMask returns attentionMask from encoding
func (e *Encoding) GetAttentionMask() []uint32 {
	return e.AttentionMask
}

// GetOverflowing returns overflowing from encoding
func (e *Encoding) GetOverflowing() []Encoding {
	return e.Overflowing
}

// TakeOverflowing returns overflowing and reset it to empty at encoding
func (e *Encoding) TakeOverflowing() []Encoding {
	o := e.Overflowing
	e.Overflowing = []Encoding{}
	return o
}

// Truncate truncates the current encoding
func (e *Encoding) Truncate(maxLen uint, stride uint) error {

	if stride >= maxLen || maxLen == 0 {
		return errors.New("Invalid input maxLen or stride (stride must be less than maxLen and maxLen must be greater than zero.)")
	}

	if maxLen >= uint(len(e.Ids)) {
		// do nothing
		return nil
	}

	// Truncating at maxLen (exclusive) to keep.
	// The rest (overflowing) from maxLen (inclusive)
	newIds := e.Ids[0:maxLen]
	oIds := e.Ids[maxLen:len(e.Ids)] // overflowing
	newTypeIds := e.TypeIds[0:maxLen]
	oTypeIds := e.TypeIds[maxLen:len(e.TypeIds)]
	newTokens := e.Tokens[0:maxLen]
	oTokens := e.Tokens[maxLen:len(e.Tokens)]
	newOffsets := e.Offsets[0:maxLen]
	oOffsets := e.Offsets[maxLen:len(e.Offsets)]
	newSpeToks := e.SpecialTokenMask[0:maxLen]
	oSpeToks := e.SpecialTokenMask[maxLen:len(e.SpecialTokenMask)]
	newAttent := e.AttentionMask[0:maxLen]
	oAttent := e.AttentionMask[maxLen:len(e.AttentionMask)]

	// Separate the overflowing part into as many Encoding as needed
	partSize := maxLen - stride
	overflowing := make([]Encoding, 0)
	partId := 0
	prevEncoding := e

	// while loop
	for int(partSize)*partId < len(oIds) {

		o := Encoding{
			Normalized:       e.Normalized,
			Ids:              (getCurrentPart(prevEncoding.Ids, oIds, partSize, uint(partId), stride)).([]uint32),
			TypeIds:          (getCurrentPart(prevEncoding.TypeIds, oTypeIds, partSize, uint(partId), stride)).([]uint32),
			Tokens:           (getCurrentPart(prevEncoding.Tokens, oTokens, partSize, uint(partId), stride)).([]string),
			Offsets:          (getCurrentPart(prevEncoding.Offsets, oOffsets, partSize, uint(partId), stride)).([]Offset),
			SpecialTokenMask: (getCurrentPart(prevEncoding.SpecialTokenMask, oSpeToks, partSize, uint(partId), stride)).([]uint32),
			AttentionMask:    (getCurrentPart(prevEncoding.AttentionMask, oAttent, partSize, uint(partId), stride)).([]uint32),
			Overflowing:      make([]Encoding, 0),
		}

		partId += 1
		overflowing = append(overflowing, o)
		prevEncoding = &overflowing[len(overflowing)-1]

	}

	// replace previous encoding with truncated one
	e = &Encoding{
		Normalized:       e.Normalized,
		Ids:              newIds,
		TypeIds:          newTypeIds,
		Tokens:           newTokens,
		Offsets:          newOffsets,
		SpecialTokenMask: newSpeToks,
		AttentionMask:    newAttent,
		Overflowing:      overflowing,
	}

	return nil

}

// MergeWith merges the current encoding with other (pair) encoding
func (e *Encoding) MergeWith(pair Encoding) {
	// Merge overflowing
	overflowings := make([]Encoding, 0)
	// 1. All current overflowing with all other overflowing
	for _, o := range e.Overflowing {
		currO := o
		// 1.1. The pair itself
		currO.MergeWith(pair) // recursively call
		overflowings = append(overflowings, currO)
		currO = o // reset

		// 1.2. The pair's overflowing
		for _, otherO := range pair.Overflowing {
			currO.MergeWith(otherO)
			overflowings = append(overflowings, currO)
			currO = o // reset
		}
	}

	// 2. Current encoding with all other overflowing
	for _, otherO := range pair.Overflowing {
		newE := e
		newE.MergeWith(otherO)
		overflowings = append(overflowings, *newE)
	}

	// 3. Current encoding and other encoding
	e.Normalized.MergeWith(pair.Normalized.Get())
	e.Ids = append(e.Ids, pair.Ids...)
	e.TypeIds = append(e.TypeIds, pair.TypeIds...)
	e.Tokens = append(e.Tokens, pair.Tokens...)
	// Offsets
	var startingOffset uint = 0
	for _, o := range e.Offsets {
		if o.End > startingOffset {
			startingOffset = o.End
		}
	}
	for _, o := range pair.Offsets {
		adjustedO := Offset{
			Start: o.Start + startingOffset,
			End:   o.End + startingOffset,
		}
		e.Offsets = append(e.Offsets, adjustedO)
	}
	e.SpecialTokenMask = append(e.SpecialTokenMask, pair.SpecialTokenMask...)
	e.AttentionMask = append(e.AttentionMask, pair.AttentionMask...)
	e.Overflowing = overflowings

}

// Pad pads current encoding with given length, values to either Left or Right direction
func (e *Encoding) Pad(targetLength uint, padId uint32, padTypeId uint32, padToken string, direction PaddingDir) {
	// 1. Recursively call for overflowing part
	for _, o := range e.Overflowing {
		o.Pad(targetLength, padId, padTypeId, padToken, direction)
	}

	// 2. Check whether we should pad encoding itself
	// if wanted padding length is smaller, then do nothing
	if len(e.Ids) >= int(targetLength) {
		return
	}

	padLength := int(targetLength) - len(e.Ids)

	switch direction {
	case Left:
		newIds := make([]uint32, padLength)
		for i := 0; i < len(newIds); i++ {
			newIds[i] = padId
		}
		newIds = append(newIds, e.Ids...)
		e.Ids = newIds

		newTypeIds := make([]uint32, padLength)
		for i := 0; i < len(newTypeIds); i++ {
			newTypeIds[i] = padTypeId
		}
		newTypeIds = append(newTypeIds, e.Ids...)
		e.TypeIds = newTypeIds

		newTokens := make([]string, padLength)
		for i := 0; i < len(newTokens); i++ {
			newTokens[i] = padToken
		}
		newTokens = append(newTokens, e.Tokens...)
		e.Tokens = newTokens

		newSpecialTokenMask := make([]uint32, padLength)
		for i := 0; i < len(newSpecialTokenMask); i++ {
			newSpecialTokenMask[i] = 1
		}
		newSpecialTokenMask = append(newSpecialTokenMask, e.SpecialTokenMask...)
		e.SpecialTokenMask = newSpecialTokenMask

		newAttentionMask := make([]uint32, padLength)
		for i := 0; i < len(newAttentionMask); i++ {
			newAttentionMask[i] = 0
		}
		newAttentionMask = append(newAttentionMask, e.AttentionMask...)
		e.AttentionMask = newAttentionMask

		newOffsets := make([]Offset, padLength)
		for i := 0; i < len(newIds); i++ {
			newOffsets[i] = Offset{0, 0}
		}
		newOffsets = append(newOffsets, e.Offsets...)
		e.Offsets = newOffsets

	case Right:
		for i := 0; i < padLength; i++ {
			e.Ids = append(e.Ids, padId)
			e.TypeIds = append(e.TypeIds, padTypeId)
			e.Tokens = append(e.Tokens, padToken)
			e.SpecialTokenMask = append(e.SpecialTokenMask, 1)
			e.AttentionMask = append(e.AttentionMask, 0)
			e.Offsets = append(e.Offsets, Offset{0, 0})
		}

	}
}

func getCurrentPart(previous, current interface{}, size, idx, stride uint) interface{} {

	var (
		curr []interface{}
		prev []interface{}
	)
	if int((idx+1)*size) > len((current).([]interface{})) {
		curr = current.([]interface{})[:(idx * size)]
	} else {
		curr = current.([]interface{})[(idx * size) : (idx+1)*size]
	}

	prev = previous.([]interface{})[:len(previous.([]interface{}))-int(stride)]

	// concat
	return append(prev, curr...)

}