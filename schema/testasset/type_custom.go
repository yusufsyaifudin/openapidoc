package testasset

import "math/big"

type ResourceName string

type Format string

type Quantity struct {
	i big.Float
	Format
}

func (q Quantity) String() string {
	return string(q.Format)
}

func NewQuantity(str string) Quantity {
	f, _, err := big.ParseFloat(str, 10, 10000, big.ToNearestAway)
	if err != nil {
		f = big.NewFloat(10)
	}

	return Quantity{
		i:      *f,
		Format: Format(str),
	}
}
