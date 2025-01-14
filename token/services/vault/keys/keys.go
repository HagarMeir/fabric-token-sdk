/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package keys

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/pkg/errors"
)

const (
	minUnicodeRuneValue                = 0            //U+0000
	MaxUnicodeRuneValue                = utf8.MaxRune //U+10FFFF - maximum (and unallocated) code point
	CompositeKeyNamespace              = "\x00"
	TokenKeyPrefix                     = "ztoken"
	FabTokenKeyPrefix                  = "token"
	AuditTokenKeyPrefix                = "audittoken"
	TokenMineKeyPrefix                 = "mine"
	TokenSetupKeyPrefix                = "setup"
	IssuedHistoryTokenKeyPrefix        = "issued"
	TokenAuditorKeyPrefix              = "auditor"
	TokenNameSpace                     = "zkat"
	numComponentsInKey                 = 2 // 2 components: txid, index, excluding TokenKeyPrefix
	Action                             = "action"
	ActionIssue                        = "issue"
	ActionTransfer                     = "transfer"
	Precision                   uint64 = 64
	Info                               = "info"
	TokenRequestKeyPrefix              = "token_request"
	OwnerSeparator                     = "/"
	SerialNumber                       = "sn"
)

func GetTokenIdFromKey(key string) (*token2.Id, error) {
	_, components, err := SplitCompositeKey(key)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error splitting input composite key: '%s'", err))
	}

	// 4 components in key: ownerType, ownerRaw, txid, index
	if len(components) != numComponentsInKey {
		return nil, errors.New(fmt.Sprintf("not enough components in output ID composite key; expected 3, received '%s'", components))
	}

	// txid and index are the last 2 components
	txID := components[numComponentsInKey-2]
	index, err := strconv.Atoi(components[numComponentsInKey-1])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error parsing output index '%s': '%s'", components[numComponentsInKey-1], err))
	}
	return &token2.Id{TxId: txID, Index: uint32(index)}, nil
}

func SplitCompositeKey(compositeKey string) (string, []string, error) {
	componentIndex := 1
	components := []string{}
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	// there is an extra tokenIdPrefix component in the beginning, trim it off
	if len(components) < numComponentsInKey+1 {
		return "", nil, errors.Errorf("invalid composite key - not enough components found in key '%s', [%d][%v]", compositeKey, len(components), components)
	}
	return components[0], components[1:], nil
}

// CreateTokenKey Creates a rwset key for an individual output in a token transaction, as a function of
// the token owner, transaction ID, and index of the output
// TODO: move index to uint32 of uint64
func CreateTokenKey(txID string, index int) (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{txID, strconv.Itoa(index)})
}

func CreateSNKey(sn string) (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{SerialNumber, sn})
}

// TODO: move index to uint32 of uint64
func CreateFabtokenKey(txID string, index int) (string, error) {
	return CreateCompositeKey(FabTokenKeyPrefix, []string{txID, strconv.Itoa(index)})
}

func CreateAuditTokenKey(txID string, index int) (string, error) {
	return CreateCompositeKey(AuditTokenKeyPrefix, []string{txID, strconv.Itoa(index)})
}

func CreateTokenMineKey(txID string, index int) (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{TokenMineKeyPrefix, txID, strconv.Itoa(index)})
}

func CreateSetupKey() (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{TokenSetupKeyPrefix})
}

func CreateSetupBundleKey() (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{TokenSetupKeyPrefix, "bundle"})
}

func CreateTokenRequestKey(txID string) (string, error) {
	return CreateCompositeKey(TokenKeyPrefix, []string{TokenRequestKeyPrefix, txID})
}

// CreateCompositeKey and its related functions and consts copied from core/chaincode/shim/chaincode.go
func CreateCompositeKey(objectType string, attributes []string) (string, error) {
	if err := ValidateCompositeKeyAttribute(objectType); err != nil {
		return "", err
	}
	ck := CompositeKeyNamespace + objectType + string(minUnicodeRuneValue)
	for _, att := range attributes {
		if err := ValidateCompositeKeyAttribute(att); err != nil {
			return "", err
		}
		ck += att + string(minUnicodeRuneValue)
	}
	return ck, nil
}

func ValidateCompositeKeyAttribute(str string) error {
	if !utf8.ValidString(str) {
		return errors.Errorf("not a valid utf8 string: [%x]", str)
	}
	for index, runeValue := range str {
		if runeValue == minUnicodeRuneValue || runeValue == MaxUnicodeRuneValue {
			return errors.Errorf(`input contain unicode %#U starting at position [%d]. %#U and %#U are not allowed in the input attribute of a composite key`,
				runeValue, index, minUnicodeRuneValue, MaxUnicodeRuneValue)
		}
	}
	return nil
}

func CreateIssuedHistoryTokenKey(txID string, index int) (string, error) {
	return CreateCompositeKey(IssuedHistoryTokenKeyPrefix, []string{txID, strconv.Itoa(index)})
}

/*
func GetSNFromKey(key string) (string, error) {
	_, components, err := SplitCompositeKey(key)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error splitting input composite key: '%s'", err))
	}

	// 2 components in key: serial number key and seial number value
	if len(components) != 2 {
		return "", errors.New(fmt.Sprintf("not enough components in output ID composite key; expected 2, received '%d'", len(components)))
	}
	if components[0] != SerialNumber {
		return "", errors.New(fmt.Sprintf("invalid serial number"))

	}

	return components[1], nil
}
*/
