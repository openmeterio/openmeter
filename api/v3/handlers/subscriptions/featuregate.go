package subscriptions

func (h *handler) isCreditsEnabled(ns string) (bool, error) {
	if !h.credits.Enabled {
		return false, nil
	}
	if h.featureGate == nil {
		return true, nil
	}
	if h.credits.FeatureFlag == "" {
		return true, nil
	}

	return h.featureGate.EvaluateBool(ns, h.credits.FeatureFlag, false)
}
