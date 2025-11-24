package credit

import "github.com/openmeterio/openmeter/pkg/models"

const ErrCodeGrantAmountMustBePositive models.ErrorCode = "grant_amount_must_be_positive"

var ErrGrantAmountMustBePositive = models.NewValidationIssue(
	ErrCodeGrantAmountMustBePositive,
	"amount must be positive",
	models.WithFieldString("amount"),
)

const ErrCodeEffectiveAtMustBeSet models.ErrorCode = "grant_effective_at_must_be_set"

var ErrGrantEffectiveAtMustBeSet = models.NewValidationIssue(
	ErrCodeEffectiveAtMustBeSet,
	"effective at must be set",
	models.WithFieldString("effectiveAt"),
)
