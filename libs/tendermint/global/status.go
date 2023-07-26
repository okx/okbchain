package global

var repairState bool

func SetRepairState(state bool) {
	repairState = state
}

func GetRepairState() bool {
	return repairState
}
