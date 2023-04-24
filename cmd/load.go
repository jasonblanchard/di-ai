/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jasonblanchard/di-ai/db/store"
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tiktoken-go/tokenizer"
)

type EntryRecord struct {
	ID        int32
	CreatorID string
	Text      string
	CreatedAt string
	UpdatedAt string
}

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := cmd.Flags().GetString("file")

		if err != nil {
			return fmt.Errorf("error getting file flag: %w", err)
		}

		dryRunOnly, err := cmd.Flags().GetBool("dry-run")

		if err != nil {
			return fmt.Errorf("error getting dry-run flag: %w", err)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error getting opening file: %w", err)
		}

		csvReader := csv.NewReader(file)

		records, err := csvReader.ReadAll()
		if err != nil {
			return fmt.Errorf("error reading CSV: %w", err)
		}

		entryRecords := []EntryRecord{}
		tokenSum := 0

		enc, err := tokenizer.ForModel(tokenizer.TextEmbeddingAda002)
		if err != nil {
			return fmt.Errorf("error creating tokenizer: %w", err)
		}

		for _, record := range records[1:] {
			id, err := strconv.Atoi(record[0])
			text := record[1]
			creatorId := record[2]
			createdAt := record[3]
			updatedAt := record[4]

			if err != nil {
				return fmt.Errorf("error converting ID to int: %w", err)
			}

			_, tokens, err := enc.Encode(text)

			tokenSum = tokenSum + len(tokens)

			if err != nil {
				return fmt.Errorf("error getting tokens: %w", err)
			}

			entryRecords = append(entryRecords, EntryRecord{
				ID:        int32(id),
				Text:      text,
				CreatorID: creatorId,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			})
		}

		fmt.Printf("total records: %v\n", len(entryRecords))
		fmt.Printf("total tokens: %v\n", tokenSum)

		costPer1000 := 0.0004
		cost := float64((tokenSum / 1000)) * costPer1000

		fmt.Printf("expected cost: $%v\n", cost)

		if dryRunOnly {
			return nil
		}

		ctx := context.Background()

		db, err := sql.Open("postgres", "user=postgres dbname=postgres sslmode=disable host=0.0.0.0 password=sekret") // TODO: Get from env
		if err != nil {
			return fmt.Errorf("error connecting to database: %w", err)
		}

		queries := store.New(db)

		key := viper.GetString("openaikey")
		openaiclient := openai.NewClient(key)

		// Get list of records already loaded
		loadedIds, err := queries.GetLoadedEntryIds(ctx)

		if err != nil {
			return fmt.Errorf("error getting entry IDs: %w", err)
		}

		for _, record := range entryRecords {
			// TODO: If not in list of records already loaded
			if contains(loadedIds, record.ID) {
				fmt.Printf("Skipping record %v\n", record.ID)
				continue
			}

			fmt.Printf("Loading record %v...\n", record.ID)

			text := strings.ReplaceAll(record.Text, "\n", " ")

			if text == "" {
				fmt.Printf("Skipping %v due to empty string\n", text)
				continue
			}

			response, err := openaiclient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
				Model: openai.AdaEmbeddingV2,
				Input: []string{text},
			})

			if err != nil {
				return fmt.Errorf("error creating embedding: %w", err)
			}

			embedding := pgvector.NewVector(response.Data[0].Embedding)

			layout := "2006-01-02 15:04:05.000"
			createdAt, err := time.Parse(layout, record.CreatedAt)

			if err != nil {
				return fmt.Errorf("error parsing created at time: %w", err)
			}

			var updatedAt time.Time

			if record.UpdatedAt != "" {
				updatedAt, err = time.Parse(layout, record.UpdatedAt)
				if err != nil {
					return fmt.Errorf("error parsing updatedAt at time: %w", err)
				}
			}

			var sqlUpdatedAt sql.NullTime

			if updatedAt.IsZero() {
				sqlUpdatedAt = sql.NullTime{
					Valid: false,
				}
			} else {
				sqlUpdatedAt = sql.NullTime{
					Valid: true,
					Time:  updatedAt,
				}
			}

			err = queries.LoadEntry(ctx, store.LoadEntryParams{
				ID: int32(record.ID),
				Text: sql.NullString{
					Valid:  true,
					String: record.Text,
				},
				CreatorID: record.CreatorID,
				CreatedAt: createdAt,
				UpdatedAt: sqlUpdatedAt,
				Embedding: embedding,
			})

			if err != nil {
				return fmt.Errorf("error loading entry into database: %w", err)
			}

			fmt.Printf("Loaded record %v...\n", record.ID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	loadCmd.Flags().StringP("file", "f", "", "CSV file with entries")
	loadCmd.Flags().BoolP("dry-run", "d", false, "get statistics but don't actually execute it")
}

func contains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
