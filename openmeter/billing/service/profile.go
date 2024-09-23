package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var (
	ErrDefaultProfileAlreadyExists = errors.New("default profile already exists")
	ErrProfileWithKeyAlreadyExists = errors.New("a profile with the specified key already exists")
)

func (s *Service) CreateProfile(ctx context.Context, params billing.CreateProfileInput) (*billing.Profile, error) {
	if err := params.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.repo, func(ctx context.Context, repo billing.TxRepository) (*billing.Profile, error) {
		// Given that we have multiple constraints let's validate those here for better error reporting
		if params.Default {
			defaultProfile, err := repo.GetDefaultProfile(ctx, billing.RepoGetDefaultProfileInput{
				Namespace: params.Namespace,
			})
			if err != nil {
				return nil, err
			}

			if defaultProfile != nil {
				return nil, billing.ValidationError{
					Err: fmt.Errorf("%w [id=%s,key=%s]", ErrDefaultProfileAlreadyExists, defaultProfile.ID, defaultProfile.Key),
				}
			}
		}

		// Let's check for key uniqueness
		existingProfileWithKey, err := repo.GetProfileByKey(ctx, billing.RepoGetProfileByKeyInput{
			Namespace: params.Namespace,
			Key:       params.Key,
		})
		if err != nil {
			return nil, err
		}

		if existingProfileWithKey != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", ErrProfileWithKeyAlreadyExists, existingProfileWithKey.ID),
			}
		}

		profile, err := s.repo.CreateProfile(ctx, params)
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		return profile, nil
	})
}
