package app

import (
	"math"

	"github.com/pkg/errors"
)

var ErrNegativeAmount = errors.New("amount should be positive value")
var ErrNotRoundedAmount = errors.New("amount should be multiple of 0.01")

const amountMultiplier = 100
const float64ComparisonThreshold = 1e-9

type Amount interface {
	Value() float64
	RawValue() uint64
}

func AmountFromRawValue(value uint64) Amount {
	return &amount{value: value}
}

func AmountFromFloat(value float64) (Amount, error) {
	val := math.Round(value * amountMultiplier)
	if val <= 0 {
		return nil, errors.WithStack(ErrNegativeAmount)
	}
	diff := math.Abs(val - value*amountMultiplier)
	if diff > float64ComparisonThreshold {
		return nil, errors.WithStack(ErrNotRoundedAmount)
	}
	return &amount{value: uint64(val)}, nil
}

type amount struct {
	value uint64
}

func (a *amount) Value() float64 {
	return float64(a.value) / amountMultiplier
}

func (a *amount) RawValue() uint64 {
	return a.value
}
