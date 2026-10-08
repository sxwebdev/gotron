package client

import (
	"math"
	"math/big"

	"github.com/sxwebdev/gotron/pkg/units"
)

// MinDelegateBalance is the least stake a DelegateResource may lend.
// java-tron's DelegateResourceActuator refuses less on every node and whatever
// the owner holds (ErrDelegateBelowMinimum): the bound is a protocol constant,
// not a network parameter, so a delegation sized under it is never built.
const MinDelegateBalance SUN = units.SunPerTRX

// ResourceRates are the network-wide totals a stake's energy and bandwidth are
// computed from, as java-tron computes them. GetAccountResource supplies the
// weights and TotalNetLimit, ChainParams TotalEnergyCurrentLimit and
// AllowHardenResourceCalculation.
//
// They answer what the chain will count, which is not the exact share the
// Convert* helpers return: the chain truncates, and before it hardened its
// resource calculation it computed in doubles, so a stake that reaches a
// number of units in exact arithmetic can come a unit short of it on chain.
type ResourceRates struct {
	TotalEnergyCurrentLimit int64
	TotalEnergyWeight       int64 // TRX
	TotalNetLimit           int64
	TotalNetWeight          int64 // TRX
	// Harden is the getAllowHardenResourceCalculation proposal: the chain
	// computes in integers instead of doubles.
	Harden bool
}

// Limit is the resource an account gets from staked, as the chain counts it
// for the account (EnergyProcessor / BandwidthProcessor
// calculateGlobal*LimitV2): (long)(staked / TRX_PRECISION * ((double) limit /
// weight)), or staked * limit / (TRX_PRECISION * weight) in integers once the
// network hardened its resource calculation. A delegated stake counts on the
// receiver's side by the same formula, over everything it holds.
func (r ResourceRates) Limit(resource ResourceType, staked SUN) int64 {
	limit, weight := r.totals(resource)
	if weight <= 0 || staked <= 0 {
		return 0
	}
	if r.Harden {
		n := new(big.Int).Mul(big.NewInt(int64(staked)), big.NewInt(limit))
		n.Quo(n, new(big.Int).Mul(big.NewInt(units.SunPerTRX), big.NewInt(weight)))
		if !n.IsInt64() {
			return math.MaxInt64
		}
		return n.Int64()
	}
	return javaLong(float64(staked) / units.SunPerTRX * (float64(limit) / float64(weight)))
}

// StakeFor is the least stake whose Limit reaches resourceUnits: what to
// delegate for a receiver to get them. It starts from the exact share rounded
// up and walks to the figure Limit gives, which a double can put a unit either
// side of it. A network with no limit or no weight cannot yield anything and
// gets math.MaxInt64, as does a share past the int64 range.
func (r ResourceRates) StakeFor(resource ResourceType, resourceUnits int64) SUN {
	if resourceUnits <= 0 {
		return 0
	}
	limit, weight := r.totals(resource)
	if limit <= 0 || weight <= 0 {
		return math.MaxInt64
	}
	n := new(big.Int).Mul(big.NewInt(resourceUnits), big.NewInt(units.SunPerTRX))
	n.Mul(n, big.NewInt(weight))
	d := big.NewInt(limit)
	n.Add(n, new(big.Int).Sub(d, big.NewInt(1)))
	n.Quo(n, d)
	if !n.IsInt64() {
		return math.MaxInt64
	}
	staked := SUN(n.Int64())
	// Defensive: a double a hair above the exact share would let a SUN less
	// reach the units. The walk up is the one real networks need.
	for staked > 0 && r.Limit(resource, staked-1) >= resourceUnits {
		staked--
	}
	for staked < math.MaxInt64 && r.Limit(resource, staked) < resourceUnits {
		staked++
	}
	return staked
}

// CallStakeFor is the least stake that gives a smart-contract call energy
// units. The virtual machine counts an account's energy stake in whole TRX
// (java-tron RepositoryImpl.calculateGlobalEnergyLimit), so it is StakeFor
// rounded up to a TRX: the whole TRX reach the units, the TRX below does not.
// Delegate it for energy a contract call will spend, such as a token transfer.
func (r ResourceRates) CallStakeFor(energy int64) SUN {
	staked := r.StakeFor(ResourceTypeEnergy, energy)
	if staked <= 0 || staked > math.MaxInt64-units.SunPerTRX {
		return staked
	}
	return (staked + units.SunPerTRX - 1) / units.SunPerTRX * units.SunPerTRX
}

// MinDelegateUnits is the fewest units of resource whose StakeFor reaches
// MinDelegateBalance. Fewer units need less stake than a delegation may lend,
// so the node refuses a delegation sized for them (ErrDelegateBelowMinimum):
// lend the minimum for them instead, which gives at least MinDelegateUnits-1
// units and so covers any of them. From MinDelegateUnits up, StakeFor is the
// delegation and reaches the minimum by itself. The minimum alone usually
// gives a unit less than MinDelegateUnits, so do not size MinDelegateUnits
// units by it.
//
// Limit never falls as the stake grows, so StakeFor(n) reaches the minimum
// exactly when a stake a SUN short of it gives fewer than n units: the answer
// is one unit past what that stake gives, in whichever mode r computes. A
// network with no limit or no weight gives 1, as no stake reaches any units
// there; so do totals no node reports, such as a negative limit. The figure
// cannot overflow: a stake under a TRX gives under the whole limit, which is
// an int64.
func (r ResourceRates) MinDelegateUnits(resource ResourceType) int64 {
	return max(r.Limit(resource, MinDelegateBalance-1)+1, 1)
}

func (r ResourceRates) totals(resource ResourceType) (limit, weight int64) {
	if resource == ResourceTypeEnergy {
		return r.TotalEnergyCurrentLimit, r.TotalEnergyWeight
	}
	return r.TotalNetLimit, r.TotalNetWeight
}

// javaLong is Java's (long) of a double: towards zero, saturating, NaN as 0
// (JLS 5.1.3). Go leaves out-of-range conversions to the platform.
func javaLong(f float64) int64 {
	switch {
	case math.IsNaN(f):
		return 0
	case f >= math.MaxInt64:
		return math.MaxInt64
	case f <= math.MinInt64:
		return math.MinInt64
	default:
		return int64(f)
	}
}
