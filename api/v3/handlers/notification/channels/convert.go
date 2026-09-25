package channels

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromAPIChannelSortField maps a v3 API sort field name to the domain OrderBy. The
// allow-list and default ("id") mirror the adapter's own fallback ordering
// (openmeter/notification/adapter/channel.go), so an unset sort produces the same
// order the service already defaults to.
func FromAPIChannelSortField(ctx context.Context, field string) (notification.OrderBy, error) {
	switch field {
	case "id":
		return notification.OrderByID, nil
	case "type":
		return notification.OrderByType, nil
	case "created_at":
		return notification.OrderByCreatedAt, nil
	case "updated_at":
		return notification.OrderByUpdatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "id", "type", "created_at", "updated_at")
	}
}

// ToDomainChannelType maps the v3 wire-format channel type ("webhook") to the domain
// ChannelType ("WEBHOOK"). The wire value is lowercase to satisfy the v3 enum casing
// convention while the domain/DB value stays uppercase for V1 backwards compatibility;
// an unknown value returns a validation error here rather than passing through, since a
// silently invalid ChannelType would otherwise either fail deep inside domain
// validation with a confusing error (create/update) or silently match zero rows
// (filter[type]).
func ToDomainChannelType(v api.NotificationChannelType) (notification.ChannelType, error) {
	switch v {
	case api.NotificationChannelTypeWebhook:
		return notification.ChannelTypeWebhook, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification channel type: %s", v))
	}
}

// ToAPIChannelType maps the domain ChannelType to the v3 wire-format channel type.
func ToAPIChannelType(v notification.ChannelType) (api.NotificationChannelType, error) {
	switch v {
	case notification.ChannelTypeWebhook:
		return api.NotificationChannelTypeWebhook, nil
	default:
		return "", fmt.Errorf("invalid notification channel type: %s", v)
	}
}

// ToAPIChannel maps a domain Channel to its v3 API representation.
func ToAPIChannel(c notification.Channel) (api.NotificationChannel, error) {
	apiType, err := ToAPIChannelType(c.Type)
	if err != nil {
		return api.NotificationChannel{}, err
	}

	channel := api.NotificationChannel{
		Id:        c.ID,
		Name:      c.Name,
		Type:      apiType,
		Disabled:  lo.ToPtr(c.Disabled),
		Url:       c.Config.WebHook.URL,
		Labels:    labels.FromMetadataAnnotations(c.Metadata, c.Annotations),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,
	}

	if len(c.Config.WebHook.CustomHeaders) > 0 {
		headers := c.Config.WebHook.CustomHeaders
		channel.CustomHeaders = &headers
	}

	if c.Config.WebHook.SigningSecret != "" {
		channel.SigningSecret = lo.ToPtr(c.Config.WebHook.SigningSecret)
	}

	return channel, nil
}

// FromAPICreateChannelRequest maps a v3 create request body to the domain input.
func FromAPICreateChannelRequest(ns string, body api.CreateNotificationChannelRequest) (notification.CreateChannelInput, error) {
	domainType, err := ToDomainChannelType(body.Type)
	if err != nil {
		return notification.CreateChannelInput{}, err
	}

	ma, err := labels.ToMetadataAnnotations(body.Labels)
	if err != nil {
		return notification.CreateChannelInput{}, err
	}

	return notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            domainType,
		Name:            body.Name,
		Disabled:        lo.FromPtrOr(body.Disabled, false),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{Type: domainType},
			WebHook: notification.WebHookChannelConfig{
				URL:           body.Url,
				CustomHeaders: lo.FromPtr(body.CustomHeaders),
				SigningSecret: lo.FromPtr(body.SigningSecret),
			},
		},
		Metadata:    ma.Metadata,
		Annotations: ma.Annotations,
	}, nil
}

// FromAPIUpdateChannelRequest maps a v3 update request body to the domain input.
// Updates are full replacements per the spec: type, name, and url are required on
// the wire, while an omitted disabled/labels/custom_headers resets the field. An
// omitted signing_secret maps to the empty string, which the service layer treats
// as "keep the current secret" — the credential is never cleared by omission.
func FromAPIUpdateChannelRequest(ns string, id string, body api.UpdateNotificationChannelRequest) (notification.UpdateChannelInput, error) {
	ma, err := labels.ToMetadataAnnotations(body.Labels)
	if err != nil {
		return notification.UpdateChannelInput{}, err
	}

	domainType, err := ToDomainChannelType(body.Type)
	if err != nil {
		return notification.UpdateChannelInput{}, err
	}

	return notification.UpdateChannelInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        id,
		},
		Type:     domainType,
		Name:     body.Name,
		Disabled: lo.FromPtr(body.Disabled),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: domainType,
			},
			WebHook: notification.WebHookChannelConfig{
				URL:           body.Url,
				CustomHeaders: lo.FromPtr(body.CustomHeaders),
				SigningSecret: lo.FromPtr(body.SigningSecret),
			},
		},
		Metadata:    ma.Metadata,
		Annotations: ma.Annotations,
	}, nil
}

// mapAPIChannelTypeFilter translates a Type filter's wire-format values ("webhook")
// into the domain/DB value ("WEBHOOK") so filter[type][eq]=webhook actually matches
// rows in the notification_channels table (the column stores the uppercase domain
// value, not the wire value). filters.FromAPIFilterStringExact always produces a
// single flat *filter.FilterString (never And-wrapped), so only Eq/Ne/In need
// translating here.
func mapAPIChannelTypeFilter(f *filter.FilterString) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	mapped := *f

	if f.Eq != nil {
		v, err := ToDomainChannelType(api.NotificationChannelType(*f.Eq))
		if err != nil {
			return nil, err
		}
		mapped.Eq = lo.ToPtr(string(v))
	}

	if f.Ne != nil {
		v, err := ToDomainChannelType(api.NotificationChannelType(*f.Ne))
		if err != nil {
			return nil, err
		}
		mapped.Ne = lo.ToPtr(string(v))
	}

	if f.In != nil {
		values := make([]string, 0, len(*f.In))
		for _, raw := range *f.In {
			v, err := ToDomainChannelType(api.NotificationChannelType(raw))
			if err != nil {
				return nil, err
			}
			values = append(values, string(v))
		}
		mapped.In = &values
	}

	return &mapped, nil
}
