package sporttery

const (
	LotteryTypeBJDC  = "226" // 北京单场
	LotteryTypeFoot  = "227" // 竞彩足球
	LotteryTypeBask  = "228" // 竞彩篮球
)

const (
	SportFootball = "1"
	SportBasketball = "2"
)

// Football sub-types
const (
	FootSubHHAD = "1" // 让球胜平负
	FootSubTTG  = "2" // 总进球
	FootSubCRS  = "3" // 比分
	FootSubHAFU = "4" // 半全场
	FootSubHAD  = "6" // 胜平负
)

// Basketball sub-types
const (
	BaskSubMNL = "1" // 胜负
	BaskSubHDC = "2" // 让分胜负
	BaskSubWSF = "3" // 胜分差
	BaskSubHHU = "4" // 大小分
)

// Football pool codes (used in sporttery API URL)
const (
	PoolHAD  = "had"
	PoolHHAD = "hhad"
	PoolCRS  = "crs"
	PoolTTG  = "ttg"
	PoolHAFU = "hafu"
)

// Basketball pool codes
const (
	PoolMNL = "mnl"
	PoolHDC = "hdc"
	PoolWSF = "wsf"
	PoolHHU = "hhu"
)

// SubTypeLabel returns a human-readable label for the sub-type.
func SubTypeLabel(lotteryType, subType string) string {
	switch lotteryType {
	case LotteryTypeFoot:
		switch subType {
		case FootSubHHAD:
			return "让球胜平负"
		case FootSubTTG:
			return "总进球"
		case FootSubCRS:
			return "比分"
		case FootSubHAFU:
			return "半全场"
		case FootSubHAD:
			return "胜平负"
		}
	case LotteryTypeBask:
		switch subType {
		case BaskSubMNL:
			return "胜负"
		case BaskSubHDC:
			return "让分胜负"
		case BaskSubWSF:
			return "胜分差"
		case BaskSubHHU:
			return "大小分"
		}
	}
	return ""
}

// PoolCodeToSubType converts a sporttery pool code to our sub-type.
func PoolCodeToSubType(lotteryType, poolCode string) string {
	if lotteryType == LotteryTypeFoot {
		switch poolCode {
		case PoolHAD:
			return FootSubHAD
		case PoolHHAD:
			return FootSubHHAD
		case PoolCRS:
			return FootSubCRS
		case PoolTTG:
			return FootSubTTG
		case PoolHAFU:
			return FootSubHAFU
		}
	}
	if lotteryType == LotteryTypeBask {
		switch poolCode {
		case PoolMNL:
			return BaskSubMNL
		case PoolHDC:
			return BaskSubHDC
		case PoolWSF:
			return BaskSubWSF
		case PoolHHU:
			return BaskSubHHU
		}
	}
	return ""
}

// FootballPoolCodes returns all football pool codes to crawl.
func FootballPoolCodes() []string {
	return []string{PoolHAD, PoolHHAD, PoolCRS, PoolTTG, PoolHAFU}
}

// BasketballPoolCodes returns all basketball pool codes to crawl.
func BasketballPoolCodes() []string {
	return []string{PoolMNL, PoolHDC, PoolWSF, PoolHHU}
}
