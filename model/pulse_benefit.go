package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// BenefitSourcePulseReward is the only source type accepted by the Pulse
	// internal receiver. The source_ref is the immutable Pulse reward grant ID.
	BenefitSourcePulseReward = "pulse_reward"
)

var (
	ErrPulseBenefitConflict = errors.New("pulse benefit source_ref payload conflict")
	ErrPulseBenefitNotFound = errors.New("pulse benefit not found")
)

// PulseBenefitGrantRequest is the narrow internal contract between Pulse and
// new-api. It intentionally contains no transferable quota: activity rewards
// must not be converted into user-created wallet redemption value.
type PulseBenefitGrantRequest struct {
	GrantID           string `json:"grant_id"`
	UserID            int    `json:"user_id"`
	Amount            int    `json:"amount"`
	TransferableQuota bool   `json:"transferable_quota"`
	SourceRef         string `json:"source_ref"`
	RewardType        string `json:"reward_type"`
	PayloadHash       string `json:"payload_hash,omitempty"`
}

type PulseBenefitResult struct {
	SourceRef   string `json:"source_ref"`
	Status      string `json:"status"`
	Applied     bool   `json:"applied"`
	PayloadHash string `json:"payload_hash,omitempty"`
}

// GrantPulseBenefit applies a Pulse grant and records it in the same database
// transaction as the quota mutation. The record is inserted before the quota
// update, so a duplicate-key race cannot apply the quota twice.
func GrantPulseBenefit(req PulseBenefitGrantRequest) (PulseBenefitResult, error) {
	if err := validatePulseGrant(req); err != nil {
		return PulseBenefitResult{}, err
	}
	fingerprint, err := pulseBenefitFingerprint(req)
	if err != nil {
		return PulseBenefitResult{}, err
	}
	if supplied := strings.TrimSpace(req.PayloadHash); supplied != "" && supplied != fingerprint {
		return PulseBenefitResult{}, ErrPulseBenefitConflict
	}

	result := PulseBenefitResult{SourceRef: req.SourceRef, PayloadHash: fingerprint}
	err = DB.Transaction(func(tx *gorm.DB) error {
		existing, findErr := findPulseGrantTx(tx, req.SourceRef)
		if findErr != nil {
			return findErr
		}
		if existing != nil {
			if existing.PayloadHash == "" || existing.PayloadHash != fingerprint || existing.UserId != req.UserID {
				return ErrPulseBenefitConflict
			}
			result.Status = "already_applied"
			result.Applied = true
			return nil
		}

		record := &BenefitChangeRecord{
			BenefitType: BenefitTypeQuota,
			Action:      BenefitActionGrant,
			SourceType:  BenefitSourcePulseReward,
			SourceRef:   req.SourceRef,
			UserId:      req.UserID,
			TargetType:  BenefitTargetUserQuota,
			// Pulse has exactly one grant per source_ref. A stable target ID
			// makes the existing cross-database unique index reject concurrent
			// requests even when an attacker changes user_id.
			TargetId:    0,
			PayloadHash: fingerprint,
			Detail: marshalBenefitDetail(&QuotaBenefitDetail{
				QuotaDelta: req.Amount,
				Context:    "pulse_reward:" + req.RewardType,
			}),
		}
		if err := tx.Create(record).Error; err != nil {
			if isBenefitDuplicateKeyErr(err) {
				return fmt.Errorf("%w: %v", errPulseBenefitDuplicate, err)
			}
			return err
		}
		if err := GrantUserQuotaTx(tx, req.UserID, req.Amount, 0); err != nil {
			return err
		}
		result.Status = "applied"
		result.Applied = true
		return nil
	})
	if err == nil {
		if result.Status == "applied" {
			_ = cacheIncrUserQuota(req.UserID, int64(req.Amount))
		}
		return result, nil
	}
	if !errors.Is(err, errPulseBenefitDuplicate) {
		return PulseBenefitResult{}, err
	}
	// A concurrent transaction won the unique source_ref race. Read its
	// committed fingerprint after the failed transaction has rolled back.
	existing, findErr := findPulseGrantTx(DB, req.SourceRef)
	if findErr != nil {
		return PulseBenefitResult{}, findErr
	}
	if existing == nil {
		return PulseBenefitResult{}, err
	}
	if existing.PayloadHash == "" || existing.PayloadHash != fingerprint || existing.UserId != req.UserID {
		return PulseBenefitResult{}, ErrPulseBenefitConflict
	}
	result.Status = "already_applied"
	result.Applied = true
	return result, nil
}

// QueryPulseBenefit returns the state for a stable source_ref. It is safe to
// call after a timeout and never creates a new source reference.
func QueryPulseBenefit(sourceRef string) (PulseBenefitResult, error) {
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" {
		return PulseBenefitResult{}, errors.New("pulse benefit source_ref is required")
	}
	grant, err := findPulseGrantTx(DB, sourceRef)
	if err != nil {
		return PulseBenefitResult{}, err
	}
	if grant == nil {
		return PulseBenefitResult{SourceRef: sourceRef, Status: "not_found", Applied: false}, ErrPulseBenefitNotFound
	}
	status := "applied"
	var rollback BenefitChangeRecord
	if err := DB.Where("origin_record_id = ? AND action = ?", grant.Id, BenefitActionRollback).First(&rollback).Error; err == nil {
		status = "rolled_back"
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return PulseBenefitResult{}, err
	}
	return PulseBenefitResult{SourceRef: sourceRef, Status: status, Applied: true, PayloadHash: grant.PayloadHash}, nil
}

// RollbackPulseBenefit reuses the original source_ref. The generic rollback
// engine appends a reversal record and is idempotent by the grant record ID.
func RollbackPulseBenefit(sourceRef, reason string) (PulseBenefitResult, error) {
	sourceRef = strings.TrimSpace(sourceRef)
	reason = strings.TrimSpace(reason)
	if sourceRef == "" || reason == "" {
		return PulseBenefitResult{}, errors.New("pulse benefit rollback requires source_ref and reason")
	}
	grant, err := findPulseGrantTx(DB, sourceRef)
	if err != nil {
		return PulseBenefitResult{}, err
	}
	if grant == nil {
		return PulseBenefitResult{}, ErrPulseBenefitNotFound
	}
	_, _, err = rollbackBenefitsBySource("pulse_reward", grant.Id, BenefitSourcePulseReward, sourceRef, reason)
	if err != nil {
		return PulseBenefitResult{}, err
	}
	return PulseBenefitResult{SourceRef: sourceRef, Status: "rolled_back", Applied: true, PayloadHash: grant.PayloadHash}, nil
}

var errPulseBenefitDuplicate = errors.New("pulse benefit duplicate")

func validatePulseGrant(req PulseBenefitGrantRequest) error {
	if strings.TrimSpace(req.GrantID) == "" || req.UserID <= 0 || req.Amount <= 0 || req.TransferableQuota || strings.TrimSpace(req.SourceRef) == "" || strings.TrimSpace(req.RewardType) == "" {
		return errors.New("invalid pulse benefit grant")
	}
	if req.GrantID != req.SourceRef {
		return errors.New("pulse grant_id must equal source_ref")
	}
	return nil
}

// Keep the JSON shape identical to Pulse's settlement outbox payload. The
// server computes this value instead of trusting a caller-supplied hash.
func pulseBenefitFingerprint(req PulseBenefitGrantRequest) (string, error) {
	payload, err := common.Marshal(struct {
		GrantID           string `json:"grant_id"`
		UserID            int    `json:"user_id"`
		Amount            int    `json:"amount"`
		TransferableQuota bool   `json:"transferable_quota"`
		SourceRef         string `json:"source_ref"`
		RewardType        string `json:"reward_type"`
	}{
		GrantID: req.GrantID, UserID: req.UserID, Amount: req.Amount,
		TransferableQuota: false, SourceRef: req.SourceRef, RewardType: req.RewardType,
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%x", digest[:]), nil
}

func findPulseGrantTx(tx *gorm.DB, sourceRef string) (*BenefitChangeRecord, error) {
	if tx == nil {
		tx = DB
	}
	query := tx.Where("source_type = ? AND source_ref = ? AND action = ?", BenefitSourcePulseReward, sourceRef, BenefitActionGrant)
	if tx != DB && !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var grant BenefitChangeRecord
	if err := query.First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}
