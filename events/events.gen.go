// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    entitlementsBalanceSnapshotV1, err := UnmarshalEntitlementsBalanceSnapshotV1(bytes)
//    bytes, err = entitlementsBalanceSnapshotV1.Marshal()
//
//    period, err := UnmarshalPeriod(bytes)
//    bytes, err = period.Marshal()
//
//    types, err := UnmarshalTypes(bytes)
//    bytes, err = types.Marshal()

package events

import "encoding/json"

func UnmarshalEntitlementsBalanceSnapshotV1(data []byte) (EntitlementsBalanceSnapshotV1, error) {
	var r EntitlementsBalanceSnapshotV1
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *EntitlementsBalanceSnapshotV1) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalPeriod(data []byte) (Period, error) {
	var r Period
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Period) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalTypes(data []byte) (Types, error) {
	var r Types
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Types) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type EntitlementsBalanceSnapshotV1 struct {
	Schema               string                                  `json:"$schema"`
	Description          string                                  `json:"description"`
	Type                 string                                  `json:"type"`
	AdditionalProperties AdditionalProperties                    `json:"additionalProperties"`
	Properties           EntitlementsBalanceSnapshotV1Properties `json:"properties"`
	Required             []string                                `json:"required"`
}

type AdditionalProperties struct {
	CloudeventsType string `json:"cloudevents.type"`
}

type EntitlementsBalanceSnapshotV1Properties struct {
	EntitlementID      CurrentUsagePeriod `json:"entitlementId"`
	Namespace          Feature            `json:"namespace"`
	Subject            Feature            `json:"subject"`
	Feature            Feature            `json:"feature"`
	Operation          Operation          `json:"operation"`
	Balance            Balance            `json:"balance"`
	CurrentUsagePeriod CurrentUsagePeriod `json:"currentUsagePeriod"`
}

type Balance struct {
	Type       string            `json:"type"`
	Properties BalanceProperties `json:"properties"`
}

type BalanceProperties struct {
	Balance   Feature `json:"balance"`
	Config    Feature `json:"config"`
	HasAccess Feature `json:"hasAccess"`
	Overage   Feature `json:"overage"`
	Usage     Feature `json:"usage"`
}

type Feature struct {
	Description string `json:"description"`
	Type        string `json:"type"`
}

type CurrentUsagePeriod struct {
	Description string `json:"description"`
	Ref         string `json:"$ref"`
}

type Operation struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Enum        []string `json:"enum"`
}

type Period struct {
	Schema      string           `json:"$schema"`
	Description string           `json:"description"`
	Type        string           `json:"type"`
	Properties  PeriodProperties `json:"properties"`
	Required    []string         `json:"required"`
}

type PeriodProperties struct {
	From CurrentUsagePeriod `json:"from"`
	To   CurrentUsagePeriod `json:"to"`
}

type Types struct {
	Schema      string      `json:"$schema"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Definitions Definitions `json:"definitions"`
}

type Definitions struct {
	Period PeriodClass `json:"period"`
	Ulid   Ulid        `json:"ulid"`
}

type PeriodClass struct {
	Type       string           `json:"type"`
	Properties PeriodProperties `json:"properties"`
	Required   []string         `json:"required"`
}

type Ulid struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}
