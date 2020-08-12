package tokenizer

import (
	"fmt"
	"reflect"
)

type PaddingDirection int

const (
	Left PaddingDirection = iota
	Right
)

// Encoding represents the output of tokenizer
type Encoding struct {
	Ids              []uint32   // ID produced by the `tokenizer`
	TypeIds          []uint32   // Type of the ID
	Tokens           []string   // Tokens associated with each ID
	Offsets          []Offsets  // Offsets of the token/ID from the NormalizedString
	SpecialTokenMask []uint32   // Mask identifying special tokens
	AttentionMask    []uint32   // Mask identifying padding tokens for the attention mechanism
	Overflowing      []Encoding // A list of overflowing generated when being truncated
	Words            []uint32   // Optional - Indexes of the word associated with each token/ID
}

// NewEncoding initiate a new encoding from input data
func NewEncoding(ids []uint32, typeIds []uint32, tokens []string, offsets []Offsets, specialTokenMask []uint32, attentionMask []uint32, overflowing []Encoding, wordsOpt ...[]uint32) Encoding {
	var words []uint32
	if len(wordsOpt) > 0 {
		words = wordsOpt[0]
	} else {
		words = nil
	}
	return Encoding{
		ids,
		typeIds,
		tokens,
		offsets,
		specialTokenMask,
		attentionMask,
		overflowing,
		words,
	}
}

func NewEncodingWithCapacity(l int) (retVal Encoding) {
	return Encoding{
		Ids:              make([]uint32, l),
		TypeIds:          make([]uint32, l),
		Tokens:           make([]string, l),
		Offsets:          make([]Offsets, l),
		SpecialTokenMask: make([]uint32, l),
		AttentionMask:    make([]uint32, l),
		Overflowing:      []Encoding{},
		Words:            make([]uint32, l),
	}
}

// Default creates an encoding with default values
func DefaultEncoding() Encoding {
	return Encoding{
		Ids:              []uint32{0},
		TypeIds:          []uint32{0},
		Tokens:           []string{},
		Offsets:          []Offsets{},
		SpecialTokenMask: []uint32{},
		AttentionMask:    []uint32{},
		Overflowing:      []Encoding{},
		Words:            nil,
	}
}

// NewEncodingFromTokens initiate Encoding from input tokens
func NewEncodingFromTokens(tokens []Token, typeId uint32) (retVal Encoding) {
	var (
		ids     []uint32
		offsets []Offsets
		words   []uint32
		toks    []string
	)
	for i, t := range tokens {
		ids = append(ids, uint32(i))
		offsets = append(offsets, t.Offsets)
		words = append(words, t.Word)
		toks = append(toks, t.Value)
	}

	typeIds := make([]uint32, len(tokens))

	return Encoding{
		Ids:              ids,
		TypeIds:          typeIds,
		Tokens:           toks,
		Offsets:          offsets,
		SpecialTokenMask: make([]uint32, 0, len(tokens)),
		AttentionMask:    make([]uint32, 1, len(tokens)),
		Overflowing:      []Encoding{},
		Words:            words,
	}
}

// IsEmpty returns whether Encoding is empty
func (e Encoding) IsEmpty() (retVal bool) {
	return len(e.Ids) == 0
}

// Len returns number of encoding tokens
func (e Encoding) Len() (retVal int) {
	return len(e.Ids)
}

// GetToken returns tokens from encoding
func (e Encoding) GetTokens() []string {
	return e.Tokens
}

// GetWords returns word indexes on normalized string
func (e Encoding) GetWords() []uint32 {
	return e.Words
}

// GetIds returns Ids from encoding
func (e Encoding) GetIds() []uint32 {
	return e.Ids
}

// GetTypeIds returns type Ids from encoding
func (e Encoding) GetTypeIds() []uint32 {
	return e.TypeIds
}

// GetOffsets returns offsets from encoding
func (e Encoding) GetOffsets() []Offsets {
	return e.Offsets
}

// GetSpecialTokenMask returns specialTokenMask from encoding
func (e Encoding) GetSpecialTokenMask() []uint32 {
	return e.SpecialTokenMask
}

// GetAttentionMask returns attentionMask from encoding
func (e Encoding) GetAttentionMask() []uint32 {
	return e.AttentionMask
}

// GetOverflowing returns overflowing from encoding
func (e Encoding) GetOverflowing() []Encoding {
	return e.Overflowing
}

// TakeOverflowing returns overflowing and reset it to empty at encoding
func (e Encoding) TakeOverflowing() []Encoding {
	o := e.Overflowing
	e.Overflowing = []Encoding{}
	return o
}

// Word2Tokens gets the encoded tokens corresponding the word
// at the given index in the input sequence
// in the form `(startToken, endToken + 1)`
//
// NOTE. e.Words is optional, therefore, there's case of `none` result
// if `none` result, `ok` will be false.
func (e Encoding) Word2Tokens(word uint32) (startTok, endTok uint32, ok bool) {

	var start, end *int

	var words []uint32
	for _, w := range e.Words {
		if w == word {
			words = append(words, w)
		}
	}
	for i, _ := range words {
		if start == nil || i < *start {
			start = &i
		}

		if end == nil || i >= *end {
			tmp := i + 1
			end = &tmp
		}
	}

	if start != nil && end != nil {
		return uint32(*start), uint32(*end), true
	} else {
		return startTok, endTok, false
	}
}

// Word2Chars get the offsets of the word at a given index in
// the input sequence
func (e Encoding) Word2Chars(word uint32) (retVal Offsets, ok bool) {
	start, end, ok := e.Word2Tokens(word)
	if end == 0 {
		return retVal, false
	} else {
		oStart := e.Offsets[start].Start
		oEnd := e.Offsets[end-1].End
		return Offsets{oStart, oEnd}, true // Should we check whether `ok`?
	}
}

// Token2Chars get the offsets of the token at the given index
func (e Encoding) Token2Chars(tokenIdx int) (retVal Offsets, ok bool) {
	if tokenIdx < 0 || tokenIdx > len(e.Offsets) {
		return retVal, false
	} else {
		return e.Offsets[tokenIdx], true
	}
}

// Token2Word get the word index of corresponding token if existing
func (e Encoding) Token2Word(tokenIdx int) (retVal uint32, ok bool) {
	// naive search. TODO. improve algorithm
	for _, w := range e.Words {
		if w == uint32(tokenIdx) {
			return w, true
		}
	}
	return retVal, false
}

// Char2Token returns a token index that contains the given `char` index
func (e Encoding) Char2Token(pos int) (retVal int, ok bool) {
	for i, o := range e.Offsets {
		if pos >= o.Start && pos < o.End {
			return i, true
		}
	}

	return -1, false
}

// Char2Word get the word index that contain the given `char` index
func (e Encoding) Char2Word(pos int) (retVal uint32, ok bool) {
	if idx, ok := e.Char2Token(pos); ok {
		return e.Token2Word(idx)
	}

	return retVal, false
}

// Truncate truncates the current encoding
func (e Encoding) Truncate(maxLen uint, stride uint) (retVal Encoding, err error) {

	if stride >= maxLen || maxLen == 0 {
		return retVal, fmt.Errorf("Invalid input maxLen or stride (stride must be less than maxLen and maxLen must be greater than zero.)")
	}

	if maxLen >= uint(len(e.Ids)) {
		// do nothing
		return e, nil
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
	newWords := e.Words[0:maxLen]
	oWords := e.Words[maxLen:len(e.Words)]

	e.Ids = newIds
	e.TypeIds = newTypeIds
	e.Tokens = newTokens
	e.Offsets = newOffsets
	e.SpecialTokenMask = newSpeToks
	e.AttentionMask = newAttent
	e.Words = newWords

	// Separate the overflowing part into as many Encoding as needed
	partSize := maxLen - stride
	overflowing := make([]Encoding, 0)
	partId := 0
	prevEncoding := e

	// while loop
	for int(partSize)*partId < len(oIds) {
		o := Encoding{
			// Which way is better? using reflect or just type assertion
			// Ids:        (getCurrentPart(prevEncoding.Ids, oIds, partSize, uint(partId), stride)).([]uint32),
			Ids:              reflect.ValueOf(getCurrentPart(prevEncoding.Ids, oIds, partSize, uint(partId), stride)).Interface().([]uint32),
			TypeIds:          reflect.ValueOf(getCurrentPart(prevEncoding.TypeIds, oTypeIds, partSize, uint(partId), stride)).Interface().([]uint32),
			Tokens:           reflect.ValueOf(getCurrentPart(prevEncoding.Tokens, oTokens, partSize, uint(partId), stride)).Interface().([]string),
			Offsets:          reflect.ValueOf(getCurrentPart(prevEncoding.Offsets, oOffsets, partSize, uint(partId), stride)).Interface().([]Offsets),
			SpecialTokenMask: reflect.ValueOf(getCurrentPart(prevEncoding.SpecialTokenMask, oSpeToks, partSize, uint(partId), stride)).Interface().([]uint32),
			AttentionMask:    reflect.ValueOf(getCurrentPart(prevEncoding.AttentionMask, oAttent, partSize, uint(partId), stride)).Interface().([]uint32),
			Words:            reflect.ValueOf(getCurrentPart(prevEncoding.Words, oWords, partSize, uint(partId), stride)).Interface().([]uint32),
			Overflowing:      make([]Encoding, 0),
		}

		partId += 1
		overflowing = append(overflowing, o)
		prevEncoding = overflowing[len(overflowing)-1]
	}

	e.Overflowing = overflowing

	return e, nil
}

// Merge merges all Encodings together
func (e Encoding) Merge(encodings []Encoding) (retVal Encoding) {
	retVal = e
	for _, encoding := range encodings {
		retVal = retVal.MergeWith(encoding)
	}

	return retVal
}

// MergeWith merges the current encoding with other (pair) encoding
func (e Encoding) MergeWith(pair Encoding) (retVal Encoding) {
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
		overflowings = append(overflowings, newE)
	}

	// 3. Current encoding and other encoding
	e.Ids = append(e.Ids, pair.Ids...)
	e.TypeIds = append(e.TypeIds, pair.TypeIds...)
	e.Tokens = append(e.Tokens, pair.Tokens...)
	// Offsets
	var startingOffset int = 0
	for _, o := range e.Offsets {
		if o.End > startingOffset {
			startingOffset = o.End
		}
	}
	for _, o := range pair.Offsets {
		adjustedO := Offsets{
			Start: o.Start + startingOffset,
			End:   o.End + startingOffset,
		}
		e.Offsets = append(e.Offsets, adjustedO)
	}
	e.SpecialTokenMask = append(e.SpecialTokenMask, pair.SpecialTokenMask...)
	e.AttentionMask = append(e.AttentionMask, pair.AttentionMask...)
	e.Overflowing = overflowings

	// 4. Re-indexing word index
	wOffset := len(e.Words)
	for _, w := range pair.Words {
		newW := w + uint32(wOffset)
		e.Words = append(e.Words, newW)
	}

	return e
}

// Pad pads current encoding with given length, values to either Left or Right direction
func (e Encoding) Pad(targetLength uint, padId uint32, padTypeId uint32, padToken string, direction PaddingDirection) (retVal Encoding) {
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

		newOffsets := make([]Offsets, padLength)
		for i := 0; i < len(newIds); i++ {
			newOffsets[i] = Offsets{0, 0}
		}
		newOffsets = append(newOffsets, e.Offsets...)
		e.Offsets = newOffsets

		newWords := make([]uint32, padLength)
		for i := 0; i < len(newWords); i++ {
			newWords[i] = 0 // Should be `none` value. TODO. implement
		}
		newWords = append(newWords, e.Words...)
		e.Words = newWords

	case Right:
		for i := 0; i < padLength; i++ {
			e.Ids = append(e.Ids, padId)
			e.TypeIds = append(e.TypeIds, padTypeId)
			e.Tokens = append(e.Tokens, padToken)
			e.SpecialTokenMask = append(e.SpecialTokenMask, 1)
			e.AttentionMask = append(e.AttentionMask, 0)
			e.Offsets = append(e.Offsets, Offsets{0, 0})
			e.Words = append(e.Words, 0) // Should be `none` value. TODO. implement
		}
	}

	return e
}

func getCurrentPart(previous, current interface{}, size, idx, stride uint) interface{} {

	switch current.(type) {
	case []uint32:
		var curr, prev []uint32
		if int((idx+1)*size) > reflect.ValueOf(current).Len() {
			curr = current.([]uint32)[(idx * size):]
		} else {
			curr = current.([]uint32)[(idx * size) : (idx+1)*size]
		}
		prev = previous.([]uint32)[len(previous.([]uint32))-int(stride):]
		return append(prev, curr...)
	case []string:
		var curr, prev []string
		if int((idx+1)*size) > reflect.ValueOf(current).Len() {
			curr = current.([]string)[(idx * size):]
		} else {
			curr = current.([]string)[(idx * size) : (idx+1)*size]
		}
		prev = previous.([]string)[len(previous.([]string))-int(stride):]
		return append(prev, curr...)
	case []Offsets:
		var curr, prev []Offsets
		if int((idx+1)*size) > reflect.ValueOf(current).Len() {
			curr = current.([]Offsets)[(idx * size):]
		} else {
			curr = current.([]Offsets)[(idx * size) : (idx+1)*size]
		}
		prev = previous.([]Offsets)[len(previous.([]Offsets))-int(stride):]
		return append(prev, curr...)
	}

	return nil
}