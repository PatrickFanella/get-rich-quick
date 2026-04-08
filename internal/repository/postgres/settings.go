package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// SettingsPersister implements api.SettingsPersister using the app_settings table.
// The table holds exactly one row (id = 1) with JSONB columns for non-secret settings.
type SettingsPersister struct {
	pool *pgxpool.Pool
}

// NewSettingsPersister returns a SettingsPersister backed by the given pool.
func NewSettingsPersister(pool *pgxpool.Pool) *SettingsPersister {
	return &SettingsPersister{pool: pool}
}

// Load retrieves persisted settings from the database.
// Returns zero-value structs without error when the row exists but columns are empty.
func (s *SettingsPersister) Load(ctx context.Context) (domain.LLMPersisted, domain.RiskSettings, error) {
	var llmJSON, riskJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT llm_config, risk_config FROM app_settings WHERE id = 1`,
	).Scan(&llmJSON, &riskJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.LLMPersisted{}, domain.RiskSettings{}, nil
		}
		return domain.LLMPersisted{}, domain.RiskSettings{}, fmt.Errorf("postgres: load settings: %w", err)
	}

	var llm domain.LLMPersisted
	if len(llmJSON) > 2 { // more than just '{}'
		if err := json.Unmarshal(llmJSON, &llm); err != nil {
			return domain.LLMPersisted{}, domain.RiskSettings{}, fmt.Errorf("postgres: unmarshal llm settings: %w", err)
		}
	}

	var risk domain.RiskSettings
	if len(riskJSON) > 2 {
		if err := json.Unmarshal(riskJSON, &risk); err != nil {
			return domain.LLMPersisted{}, domain.RiskSettings{}, fmt.Errorf("postgres: unmarshal risk settings: %w", err)
		}
	}

	return llm, risk, nil
}

// Save persists non-secret settings to the database using an UPSERT.
func (s *SettingsPersister) Save(ctx context.Context, llm domain.LLMPersisted, risk domain.RiskSettings) error {
	llmJSON, err := json.Marshal(llm)
	if err != nil {
		return fmt.Errorf("postgres: marshal llm settings: %w", err)
	}
	riskJSON, err := json.Marshal(risk)
	if err != nil {
		return fmt.Errorf("postgres: marshal risk settings: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_settings (id, llm_config, risk_config, updated_at)
		 VALUES (1, $1, $2, NOW())
		 ON CONFLICT (id) DO UPDATE
		   SET llm_config  = EXCLUDED.llm_config,
		       risk_config = EXCLUDED.risk_config,
		       updated_at  = EXCLUDED.updated_at`,
		llmJSON, riskJSON,
	)
	if err != nil {
		return fmt.Errorf("postgres: save settings: %w", err)
	}
	return nil
}
