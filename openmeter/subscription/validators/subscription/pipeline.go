package subscriptionvalidators

// All a pipeline does is execute a series of functions on a source value
// and return the result of the last function

func pipe4[A any, B any, C any, D any, E any](
	source A,
	operator1 func(A) B,
	operator2 func(B) C,
	operator3 func(C) D,
	operator4 func(D) E,
) E {
	return operator4(
		operator3(
			operator2(
				operator1(source),
			),
		),
	)
}

func pipe5[A any, B any, C any, D any, E any, F any](
	source A,
	operator1 func(A) B,
	operator2 func(B) C,
	operator3 func(C) D,
	operator4 func(D) E,
	operator5 func(E) F,
) F {
	return operator5(
		operator4(
			operator3(
				operator2(
					operator1(source),
				),
			),
		),
	)
}

func pipe6[A any, B any, C any, D any, E any, F any, G any](
	source A,
	operator1 func(A) B,
	operator2 func(B) C,
	operator3 func(C) D,
	operator4 func(D) E,
	operator5 func(E) F,
	operator6 func(F) G,
) G {
	return operator6(
		operator5(
			operator4(
				operator3(
					operator2(
						operator1(source),
					),
				),
			),
		),
	)
}
