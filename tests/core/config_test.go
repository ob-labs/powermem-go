package core_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid config with SQLite",
			envVars: map[string]string{
				"DATABASE_PROVIDER":  "sqlite",
				"SQLITE_PATH":        "./test.db",
				"LLM_PROVIDER":       "openai",
				"LLM_API_KEY":        "test-key",
				"LLM_MODEL":          "gpt-4",
				"EMBEDDING_PROVIDER": "openai",
				"EMBEDDING_API_KEY":  "test-key",
				"EMBEDDING_MODEL":    "text-embedding-3-small",
			},
			wantErr: false,
		},
		{
			name: "valid config with Qwen",
			envVars: map[string]string{
				"DATABASE_PROVIDER":  "sqlite",
				"SQLITE_PATH":        "./test.db",
				"LLM_PROVIDER":       "qwen",
				"LLM_API_KEY":        "test-key",
				"LLM_MODEL":          "qwen-plus",
				"EMBEDDING_PROVIDER": "qwen",
				"EMBEDDING_API_KEY":  "test-key",
				"EMBEDDING_MODEL":    "text-embedding-v4",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			config, err := powermem.LoadConfigFromEnv()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.envVars["DATABASE_PROVIDER"], config.VectorStore.Provider)
				assert.Equal(t, tt.envVars["LLM_PROVIDER"], config.LLM.Provider)
				assert.Equal(t, tt.envVars["EMBEDDING_PROVIDER"], config.Embedder.Provider)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *powermem.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &powermem.Config{
				LLM: powermem.LLMConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Model:    "gpt-4",
				},
				Embedder: powermem.EmbedderConfig{
					Provider: "openai",
					APIKey:   "test-key",
					Model:    "text-embedding-3-small",
				},
				VectorStore: powermem.VectorStoreConfig{
					Provider: "sqlite",
					Config: map[string]interface{}{
						"db_path": "./test.db",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing LLM provider",
			config: &powermem.Config{
				LLM: powermem.LLMConfig{
					Provider: "",
				},
				Embedder: powermem.EmbedderConfig{
					Provider: "openai",
				},
				VectorStore: powermem.VectorStoreConfig{
					Provider: "sqlite",
				},
			},
			wantErr: true,
		},
		{
			name: "missing Embedder provider",
			config: &powermem.Config{
				LLM: powermem.LLMConfig{
					Provider: "openai",
				},
				Embedder: powermem.EmbedderConfig{
					Provider: "",
				},
				VectorStore: powermem.VectorStoreConfig{
					Provider: "sqlite",
				},
			},
			wantErr: true,
		},
		{
			name: "missing VectorStore provider",
			config: &powermem.Config{
				LLM: powermem.LLMConfig{
					Provider: "openai",
				},
				Embedder: powermem.EmbedderConfig{
					Provider: "openai",
				},
				VectorStore: powermem.VectorStoreConfig{
					Provider: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindEnvFile(t *testing.T) {
	// Test finding .env file
	envPath, found := powermem.FindEnvFile()

	// This test depends on actual file system state
	// We only verify the function doesn't panic
	assert.NotNil(t, envPath)
	// found may be true or false, depending on whether .env file exists
	_ = found
}

func TestDefaultConfig(t *testing.T) {
	// Test default config values
	config := &powermem.Config{
		LLM: powermem.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Embedder: powermem.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
		},
		VectorStore: powermem.VectorStoreConfig{
			Provider: "sqlite",
			Config: map[string]interface{}{
				"db_path": "./test.db",
			},
		},
	}

	err := config.Validate()
	require.NoError(t, err)

	assert.Equal(t, "openai", config.LLM.Provider)
	assert.Equal(t, "openai", config.Embedder.Provider)
	assert.Equal(t, "sqlite", config.VectorStore.Provider)
}
