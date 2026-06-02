package httpdriver

func (h *handler) isCreditsEnabled(ns string) (bool, error) {
	if !h.Credits.Enabled {
		return false, nil
	}
	if h.FeatureGate == nil {
		return true, nil
	}
	if h.Credits.FeatureFlag == "" {
		return true, nil
	}

	return h.FeatureGate.EvaluateBool(ns, h.Credits.FeatureFlag, false)
}
