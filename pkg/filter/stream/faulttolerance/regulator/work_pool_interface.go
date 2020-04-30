package regulator

type WorkPool interface {
	Schedule(model *MeasureModel)
}
