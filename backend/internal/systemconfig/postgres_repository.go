package systemconfig

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool             *pgxpool.Pool
	encryptionSecret string
}

// NewPostgresRepository 创建并初始化 New Postgres Repository 实例。
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SetEncryptionSecret(secret string) {
	r.encryptionSecret = secret
}

// GetCollectorFilter 查询并返回指定的 Get Collector Filter。
func (r *PostgresRepository) GetCollectorFilter(ctx context.Context) (CollectorFilterConfig, error) {
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM diting_system_configs WHERE key = $1`, CollectorFilterKey).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultCollectorFilterConfig(), nil
	}
	if err != nil {
		return CollectorFilterConfig{}, err
	}
	return unmarshalCollectorFilterConfig(data)
}

// SaveCollectorFilter 处理 Save Collector Filter 相关逻辑。
func (r *PostgresRepository) SaveCollectorFilter(ctx context.Context, config CollectorFilterConfig) error {
	data, err := marshalCollectorFilterConfig(normalizeCollectorFilterConfig(config))
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO diting_system_configs (key, value, description, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW()
`, CollectorFilterKey, string(data), "Collector noise filter configuration")
	return err
}

// marshalCollectorFilterConfig 处理 marshal Collector Filter Config 相关逻辑。
func marshalCollectorFilterConfig(config CollectorFilterConfig) ([]byte, error) {
	return json.Marshal(normalizeCollectorFilterConfig(config))
}

// unmarshalCollectorFilterConfig 处理 unmarshal Collector Filter Config 相关逻辑。
func unmarshalCollectorFilterConfig(data []byte) (CollectorFilterConfig, error) {
	var config CollectorFilterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return CollectorFilterConfig{}, err
	}
	return normalizeCollectorFilterConfig(config), nil
}

func (r *PostgresRepository) GetAIConfig(ctx context.Context) (AIProviderConfig, error) {
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM diting_system_configs WHERE key = $1`, AIConfigKey).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultAIProviderConfig(), nil
	}
	if err != nil {
		return AIProviderConfig{}, err
	}
	config, err := r.unmarshalAIProviderConfig(data)
	if err != nil {
		return AIProviderConfig{}, err
	}
	return normalizeAIProviderConfig(config), nil
}

func (r *PostgresRepository) SaveAIConfig(ctx context.Context, config AIProviderConfig) error {
	existing, _ := r.GetAIConfig(ctx)
	if config.APIKey == "" {
		config.EncryptedAPIKey = existing.EncryptedAPIKey
		config.APIKeySet = existing.APIKeySet
		config.MaskedAPIKey = existing.MaskedAPIKey
	} else {
		encrypted, err := encryptSecret(config.APIKey, r.encryptionSecret)
		if err != nil {
			return err
		}
		config.EncryptedAPIKey = encrypted
		config.APIKeySet = true
		config.MaskedAPIKey = maskSecret(config.APIKey)
		config.APIKey = ""
	}
	data, err := json.Marshal(normalizeAIProviderConfig(config))
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO diting_system_configs (key, value, description, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW()
`, AIConfigKey, string(data), "OpenAI-compatible AI risk analysis configuration")
	return err
}

func (r *PostgresRepository) unmarshalAIProviderConfig(data []byte) (AIProviderConfig, error) {
	var config AIProviderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return AIProviderConfig{}, err
	}
	if config.EncryptedAPIKey != "" {
		apiKey, err := decryptSecret(config.EncryptedAPIKey, r.encryptionSecret)
		if err != nil {
			return AIProviderConfig{}, err
		}
		config.APIKey = apiKey
		config.APIKeySet = true
		config.MaskedAPIKey = maskSecret(apiKey)
	}
	return config, nil
}

func encryptSecret(value, secret string) (string, error) {
	gcm, err := secretGCM(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func decryptSecret(value, secret string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	gcm, err := secretGCM(secret)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted secret")
	}
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func secretGCM(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
