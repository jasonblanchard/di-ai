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
	"fmt"

	"github.com/jasonblanchard/di-ai/db/store"
	"github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// summarizeCmd represents the summarize command
var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		query, err := cmd.Flags().GetString("query")

		if err != nil {
			return fmt.Errorf("error getting query flag: %w", err)
		}

		if query == "" {
			return fmt.Errorf("query string cannot be empty")
		}

		key := viper.GetString("openaikey")
		openaiclient := openai.NewClient(key)

		ctx := context.Background()

		embeddingResponse, err := openaiclient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Model: openai.AdaEmbeddingV2,
			Input: []string{query},
		})

		if err != nil {
			return fmt.Errorf("error creating query embedding: %w", err)
		}

		queryEmbedding := pgvector.NewVector(embeddingResponse.Data[0].Embedding)

		db, err := sql.Open("postgres", "user=postgres dbname=postgres sslmode=disable host=0.0.0.0 password=sekret") // TODO: Get from env
		if err != nil {
			return fmt.Errorf("error connecting to database: %w", err)
		}

		queries := store.New(db)

		results, err := queries.ListEntriesByCosineSimilarity(ctx, store.ListEntriesByCosineSimilarityParams{
			Embedding: queryEmbedding,
			Limit:     10,
		})

		if err != nil {
			return fmt.Errorf("error listing entries from database: %w", err)
		}

		var context string

		// TODO: Limit size
		for _, result := range results {
			context = context + result.Text.String + "\n\n"
		}

		promptResponse, err := openaiclient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf("You are a neutral agent helping to summarize content from a journal. \n Here are some posts to give you context: \n %s", context),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Using only the context above, Write a summary about %s", query),
				},
			},
		})

		if err != nil {
			return fmt.Errorf("error getting chat completion: %w", err)
		}

		content := promptResponse.Choices[0].Message.Content

		fmt.Println("")
		fmt.Println(content)
		fmt.Println("")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// summarizeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// summarizeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	summarizeCmd.Flags().StringP("query", "q", "", "Query string to search with")
}
